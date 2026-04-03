package llmkit

import (
	"context"
	"fmt"
	"strings"

	claudesession "github.com/randalmurphal/llmkit/v2/claude/session"
	codexsession "github.com/randalmurphal/llmkit/v2/codex/session"
)

type SessionStatus string

const (
	SessionStatusCreating SessionStatus = "creating"
	SessionStatusActive   SessionStatus = "active"
	SessionStatusClosing  SessionStatus = "closing"
	SessionStatusClosed   SessionStatus = "closed"
	SessionStatusError    SessionStatus = "error"
)

type SessionInfo struct {
	Provider  string        `json:"provider"`
	ID        string        `json:"id"`
	Status    SessionStatus `json:"status"`
	Model     string        `json:"model,omitempty"`
	WorkDir   string        `json:"work_dir,omitempty"`
	TurnCount int           `json:"turn_count,omitempty"`
	CostUSD   float64       `json:"cost_usd,omitempty"`
}

type Session interface {
	Provider() string
	ID() string
	Status() SessionStatus
	Info() SessionInfo
	Send(ctx context.Context, req Request) error
	Events() <-chan StreamChunk
	Close() error
}

type SteerableSession interface {
	Session
	Steer(ctx context.Context, req Request) error
}

func NewSession(ctx context.Context, provider string, cfg Config) (Session, error) {
	if provider == "" {
		provider = cfg.Provider
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	cfg.Provider = provider
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	switch provider {
	case "claude":
		return newClaudeRootSession(ctx, cfg)
	case "codex":
		return newCodexRootSession(ctx, cfg)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
	}
}

type claudeRootSession struct {
	session claudesession.Session
	manager claudesession.SessionManager
	events  chan StreamChunk
}

func newClaudeRootSession(ctx context.Context, cfg Config) (Session, error) {
	opts := make([]claudesession.SessionOption, 0, 16)
	if cfg.BinaryPath != "" {
		opts = append(opts, claudesession.WithClaudePath(cfg.BinaryPath))
	}
	if cfg.Model != "" {
		opts = append(opts, claudesession.WithModel(cfg.Model))
	}
	if cfg.FallbackModel != "" {
		opts = append(opts, claudesession.WithFallbackModel(cfg.FallbackModel))
	}
	if cfg.ReasoningEffort != "" {
		opts = append(opts, claudesession.WithEffort(cfg.ReasoningEffort))
	}
	if cfg.WorkDir != "" {
		opts = append(opts, claudesession.WithWorkdir(cfg.WorkDir))
	}
	if cfg.SystemPrompt != "" {
		opts = append(opts, claudesession.WithSystemPrompt(cfg.SystemPrompt))
	}
	if cfg.AppendSystemPrompt != "" {
		opts = append(opts, claudesession.WithAppendSystemPrompt(cfg.AppendSystemPrompt))
	}
	if len(cfg.AllowedTools) > 0 {
		opts = append(opts, claudesession.WithAllowedTools(cfg.AllowedTools))
	}
	if len(cfg.DisallowedTools) > 0 {
		opts = append(opts, claudesession.WithDisallowedTools(cfg.DisallowedTools))
	}
	if len(cfg.Tools) > 0 {
		opts = append(opts, claudesession.WithTools(cfg.Tools))
	}
	if cfg.MaxBudgetUSD > 0 {
		opts = append(opts, claudesession.WithMaxBudgetUSD(cfg.MaxBudgetUSD))
	}
	if cfg.MaxTurns > 0 {
		opts = append(opts, claudesession.WithMaxTurns(cfg.MaxTurns))
	}
	if len(cfg.Env) > 0 {
		opts = append(opts, claudesession.WithEnv(cfg.Env))
	}
	if len(cfg.AddDirs) > 0 {
		opts = append(opts, claudesession.WithAddDirs(cfg.AddDirs))
	}
	if cfg.Runtime.Providers.Claude != nil {
		claudeCfg := cfg.Runtime.Providers.Claude
		if claudeCfg.DangerouslySkipPermissions {
			opts = append(opts, claudesession.WithPermissions(true))
		}
		if claudeCfg.PermissionMode != "" {
			opts = append(opts, claudesession.WithPermissionMode(claudeCfg.PermissionMode))
		}
		if len(claudeCfg.SettingSources) > 0 {
			opts = append(opts, claudesession.WithSettingSources(claudeCfg.SettingSources))
		}
		if len(claudeCfg.Hooks) > 0 {
			opts = append(opts, claudesession.WithIncludeHookOutput(true))
		}
	}
	if sessionID := SessionID(cfg.Session); sessionID != "" {
		if cfg.ResumeSession {
			opts = append(opts, claudesession.WithResume(sessionID))
		} else {
			opts = append(opts, claudesession.WithSessionID(sessionID))
		}
	}

	manager := claudesession.NewManager()
	sess, err := manager.Create(ctx, opts...)
	if err != nil {
		_ = manager.CloseAll()
		return nil, err
	}

	root := &claudeRootSession{
		session: sess,
		manager: manager,
		events:  make(chan StreamChunk, 128),
	}
	go root.forward()
	return root, nil
}

func (s *claudeRootSession) Provider() string { return "claude" }
func (s *claudeRootSession) ID() string       { return s.session.ID() }

func (s *claudeRootSession) Status() SessionStatus {
	return mapClaudeStatus(s.session.Status())
}

func (s *claudeRootSession) Info() SessionInfo {
	info := s.session.Info()
	return SessionInfo{
		Provider:  "claude",
		ID:        info.ID,
		Status:    mapClaudeStatus(info.Status),
		Model:     info.Model,
		WorkDir:   info.CWD,
		TurnCount: info.TurnCount,
		CostUSD:   info.TotalCostUSD,
	}
}

func (s *claudeRootSession) Send(ctx context.Context, req Request) error {
	prompt, err := sessionPrompt(req)
	if err != nil {
		return err
	}
	return s.session.Send(ctx, claudesession.NewUserMessage(prompt))
}

func (s *claudeRootSession) Events() <-chan StreamChunk { return s.events }

func (s *claudeRootSession) Close() error {
	return s.manager.CloseAll()
}

func (s *claudeRootSession) Steer(ctx context.Context, req Request) error {
	return s.Send(ctx, req)
}

func (s *claudeRootSession) forward() {
	defer close(s.events)

	for msg := range s.session.Output() {
		session := SessionMetadataForID("claude", msg.SessionID)
		switch {
		case msg.IsInit():
			model := ""
			metadata := map[string]any{}
			if msg.Init != nil {
				model = msg.Init.Model
				metadata["cwd"] = msg.Init.CWD
				metadata["permission_mode"] = msg.Init.PermissionMode
				metadata["claude_code_version"] = msg.Init.ClaudeCodeVersion
			}
			s.events <- StreamChunk{
				Type:      "session",
				SessionID: msg.SessionID,
				Session:   session,
				Model:     model,
				Metadata:  metadata,
			}
		case msg.IsAssistant():
			text := msg.GetText()
			if text != "" {
				chunk := StreamChunk{
					Type:      "assistant",
					Content:   text,
					Role:      "assistant",
					SessionID: msg.SessionID,
					Session:   session,
				}
				if msg.Assistant != nil {
					chunk.MessageID = msg.Assistant.Message.ID
					chunk.Model = msg.Assistant.Message.Model
					chunk.Usage = &TokenUsage{
						InputTokens:              msg.Assistant.Message.Usage.InputTokens,
						OutputTokens:             msg.Assistant.Message.Usage.OutputTokens,
						TotalTokens:              msg.Assistant.Message.Usage.InputTokens + msg.Assistant.Message.Usage.OutputTokens,
						CacheCreationInputTokens: msg.Assistant.Message.Usage.CacheCreationInputTokens,
						CacheReadInputTokens:     msg.Assistant.Message.Usage.CacheReadInputTokens,
					}
				}
				s.events <- chunk
			}
			if msg.Assistant != nil {
				var toolCalls []ToolCall
				for _, block := range msg.Assistant.Message.Content {
					if block.Type == "tool_use" {
						toolCalls = append(toolCalls, ToolCall{
							ID:        block.ID,
							Name:      block.Name,
							Arguments: block.Input,
						})
					}
				}
				if len(toolCalls) > 0 {
					s.events <- StreamChunk{
						Type:      "tool_call",
						Role:      "assistant",
						Model:     msg.Assistant.Message.Model,
						MessageID: msg.Assistant.Message.ID,
						SessionID: msg.SessionID,
						Session:   session,
						ToolCalls: toolCalls,
					}
				}
			}
		case msg.IsHook():
			if msg.Hook == nil {
				continue
			}
			s.events <- StreamChunk{
				Type:      "hook",
				Role:      "system",
				SessionID: msg.SessionID,
				Session:   session,
				Metadata: map[string]any{
					"hook_name":  msg.Hook.HookName,
					"hook_event": msg.Hook.HookEvent,
					"stdout":     msg.Hook.Stdout,
					"stderr":     msg.Hook.Stderr,
					"exit_code":  msg.Hook.ExitCode,
				},
			}
		case msg.IsResult():
			final := ""
			usage := &TokenUsage{}
			if msg.Result != nil {
				final = msg.Result.Result
				usage = &TokenUsage{
					InputTokens:              msg.Result.Usage.InputTokens,
					OutputTokens:             msg.Result.Usage.OutputTokens,
					TotalTokens:              msg.Result.Usage.InputTokens + msg.Result.Usage.OutputTokens,
					CacheCreationInputTokens: msg.Result.Usage.CacheCreationInputTokens,
					CacheReadInputTokens:     msg.Result.Usage.CacheReadInputTokens,
				}
			}
			s.events <- StreamChunk{
				Type:         "final",
				FinalContent: final,
				SessionID:    msg.SessionID,
				Session:      session,
				Usage:        usage,
				Done:         true,
			}
			if msg.IsError() {
				errText := final
				if errText == "" {
					errText = "claude session turn failed"
				}
				s.events <- StreamChunk{
					Type:      "error",
					SessionID: msg.SessionID,
					Session:   session,
					Error:     fmt.Errorf("%s", errText),
				}
			}
		}
	}
}

type codexRootSession struct {
	session codexsession.Session
	manager codexsession.SessionManager
	events  chan StreamChunk
}

func newCodexRootSession(ctx context.Context, cfg Config) (Session, error) {
	opts := make([]codexsession.SessionOption, 0, 12)
	if cfg.BinaryPath != "" {
		opts = append(opts, codexsession.WithCodexPath(cfg.BinaryPath))
	}
	if cfg.Model != "" {
		opts = append(opts, codexsession.WithModel(cfg.Model))
	}
	if cfg.WorkDir != "" {
		opts = append(opts, codexsession.WithWorkdir(cfg.WorkDir))
	}
	if cfg.SystemPrompt != "" {
		opts = append(opts, codexsession.WithSystemPrompt(cfg.SystemPrompt))
	}
	if cfg.ReasoningEffort != "" {
		opts = append(opts, codexsession.WithReasoningEffort(cfg.ReasoningEffort))
	}
	if len(cfg.Env) > 0 {
		opts = append(opts, codexsession.WithEnv(cfg.Env))
	}
	if sessionID := SessionID(cfg.Session); sessionID != "" {
		if !cfg.ResumeSession {
			return nil, fmt.Errorf("%w: codex sessions cannot start with a caller-chosen session ID", ErrUnsupportedFeature)
		}
		opts = append(opts, codexsession.WithResume(sessionID))
	}
	if cfg.Runtime.Providers.Codex != nil {
		codexCfg := cfg.Runtime.Providers.Codex
		if codexCfg.ReasoningEffort != "" {
			opts = append(opts, codexsession.WithReasoningEffort(codexCfg.ReasoningEffort))
		}
		if codexCfg.BypassApprovalsAndSandbox {
			opts = append(opts, codexsession.WithFullAuto())
		} else {
			if codexCfg.SandboxMode != "" {
				opts = append(opts, codexsession.WithSandboxMode(codexCfg.SandboxMode))
			}
			if codexCfg.ApprovalMode != "" {
				opts = append(opts, codexsession.WithApprovalMode(codexCfg.ApprovalMode))
			}
		}
	}

	manager := codexsession.NewManager()
	sess, err := manager.Create(ctx, opts...)
	if err != nil {
		_ = manager.CloseAll()
		return nil, err
	}

	root := &codexRootSession{
		session: sess,
		manager: manager,
		events:  make(chan StreamChunk, 128),
	}
	go root.forward()
	return root, nil
}

func (s *codexRootSession) Provider() string { return "codex" }
func (s *codexRootSession) ID() string       { return s.session.ID() }

func (s *codexRootSession) Status() SessionStatus {
	return mapCodexStatus(s.session.Status())
}

func (s *codexRootSession) Info() SessionInfo {
	info := s.session.Info()
	return SessionInfo{
		Provider:  "codex",
		ID:        info.ID,
		Status:    mapCodexStatus(info.Status),
		Model:     info.Model,
		WorkDir:   info.CWD,
		TurnCount: info.TurnCount,
	}
}

func (s *codexRootSession) Send(ctx context.Context, req Request) error {
	prompt, err := sessionPrompt(req)
	if err != nil {
		return err
	}
	return s.session.Send(ctx, codexsession.NewUserMessage(prompt))
}

func (s *codexRootSession) Steer(ctx context.Context, req Request) error {
	prompt, err := sessionPrompt(req)
	if err != nil {
		return err
	}
	return s.session.Steer(ctx, codexsession.NewUserMessage(prompt))
}

func (s *codexRootSession) Events() <-chan StreamChunk { return s.events }

func (s *codexRootSession) Close() error {
	return s.manager.CloseAll()
}

func (s *codexRootSession) forward() {
	defer close(s.events)

	for msg := range s.session.Output() {
		session := SessionMetadataForID("codex", msg.ThreadID)
		switch {
		case msg.IsThreadStarted():
			s.events <- StreamChunk{
				Type:      "session",
				SessionID: msg.ThreadID,
				Session:   session,
			}
		case msg.IsAgentMessage() || msg.IsAgentMessageDelta():
			text := msg.GetText()
			if text == "" {
				continue
			}
			s.events <- StreamChunk{
				Type:      "assistant",
				Content:   text,
				Role:      "assistant",
				SessionID: msg.ThreadID,
				Session:   session,
			}
		case msg.IsTurnComplete():
			s.events <- StreamChunk{
				Type:      "final",
				SessionID: msg.ThreadID,
				Session:   session,
				Done:      true,
			}
		case msg.IsTurnFailed():
			errText := msg.GetText()
			if errText == "" {
				errText = "codex session turn failed"
			}
			s.events <- StreamChunk{
				Type:      "error",
				SessionID: msg.ThreadID,
				Session:   session,
				Error:     fmt.Errorf("%s", errText),
			}
			s.events <- StreamChunk{
				Type:      "final",
				SessionID: msg.ThreadID,
				Session:   session,
				Done:      true,
			}
		}
	}
}

func sessionPrompt(req Request) (string, error) {
	if len(req.Messages) == 0 {
		return "", fmt.Errorf("%w: request.messages is required for session sends", ErrInvalidRequest)
	}
	if req.SystemPrompt != "" || req.Model != "" || req.MaxTokens != 0 || req.Temperature != 0 || len(req.Tools) > 0 || len(req.JSONSchema) > 0 {
		return "", fmt.Errorf("%w: session sends only support request.messages", ErrUnsupportedFeature)
	}

	var prompt strings.Builder
	for _, msg := range req.Messages {
		text := msg.GetText()
		switch msg.Role {
		case RoleUser:
			prompt.WriteString(text)
			prompt.WriteString("\n")
		case RoleAssistant:
			if prompt.Len() > 0 {
				prompt.WriteString("\nAssistant: ")
				prompt.WriteString(text)
				prompt.WriteString("\n\nUser: ")
			}
		case RoleSystem:
			prompt.WriteString(text)
			prompt.WriteString("\n\n")
		}
	}

	joined := strings.TrimSpace(prompt.String())
	if joined == "" {
		return "", fmt.Errorf("%w: request.messages did not produce any text", ErrInvalidRequest)
	}
	return joined, nil
}

func mapClaudeStatus(status claudesession.SessionStatus) SessionStatus {
	switch status {
	case claudesession.StatusCreating:
		return SessionStatusCreating
	case claudesession.StatusActive:
		return SessionStatusActive
	case claudesession.StatusClosing, claudesession.StatusTerminating:
		return SessionStatusClosing
	case claudesession.StatusClosed:
		return SessionStatusClosed
	default:
		return SessionStatusError
	}
}

func mapCodexStatus(status codexsession.SessionStatus) SessionStatus {
	switch status {
	case codexsession.StatusCreating:
		return SessionStatusCreating
	case codexsession.StatusActive:
		return SessionStatusActive
	case codexsession.StatusClosing:
		return SessionStatusClosing
	case codexsession.StatusClosed:
		return SessionStatusClosed
	default:
		return SessionStatusError
	}
}
