package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/randalmurphal/llmkit/v2"
)

func init() {
	llmkit.Register("claude", newFromProviderConfig)
	llmkit.RegisterProviderDefinition(llmkit.ProviderDefinition{
		Name:      "claude",
		Supported: true,
		Shared: llmkit.SharedSupport{
			SystemPrompt:       true,
			AppendSystemPrompt: true,
			AllowedTools:       true,
			DisallowedTools:    true,
			Tools:              true,
			MCPServers:         true,
			StrictMCPConfig:    true,
			MaxBudgetUSD:       true,
			MaxTurns:           true,
			Env:                true,
			AddDirs:            true,
		},
		Environment: llmkit.EnvironmentSupport{
			Hooks:        true,
			MCP:          true,
			Skills:       true,
			Instructions: true,
			CustomAgents: true,
		},
	})
}

func newFromProviderConfig(cfg llmkit.Config) (llmkit.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := make([]ClaudeOption, 0, 10)
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
	}
	if cfg.FallbackModel != "" {
		opts = append(opts, WithFallbackModel(cfg.FallbackModel))
	}
	if cfg.SystemPrompt != "" {
		opts = append(opts, WithSystemPrompt(cfg.SystemPrompt))
	}
	if cfg.AppendSystemPrompt != "" {
		opts = append(opts, WithAppendSystemPrompt(cfg.AppendSystemPrompt))
	}
	if cfg.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(cfg.MaxTurns))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, WithTimeout(cfg.Timeout))
	}
	if cfg.MaxBudgetUSD > 0 {
		opts = append(opts, WithMaxBudgetUSD(cfg.MaxBudgetUSD))
	}
	if cfg.BinaryPath != "" {
		opts = append(opts, WithClaudePath(cfg.BinaryPath))
	}
	if cfg.WorkDir != "" {
		opts = append(opts, WithWorkdir(cfg.WorkDir))
	}
	if len(cfg.AllowedTools) > 0 {
		opts = append(opts, WithAllowedTools(cfg.AllowedTools))
	}
	if len(cfg.DisallowedTools) > 0 {
		opts = append(opts, WithDisallowedTools(cfg.DisallowedTools))
	}
	if len(cfg.Tools) > 0 {
		opts = append(opts, WithTools(cfg.Tools))
	}
	if len(cfg.MCPServers) > 0 {
		servers := make(map[string]MCPServerConfig, len(cfg.MCPServers))
		for name, server := range cfg.MCPServers {
			servers[name] = MCPServerConfig{
				Type:    server.Type,
				Command: server.Command,
				Args:    append([]string(nil), server.Args...),
				Env:     cloneStringMap(server.Env),
				URL:     server.URL,
				Headers: mapHeadersToSlice(server.Headers),
			}
		}
		opts = append(opts, WithMCPServers(servers))
	}
	if cfg.StrictMCPConfig {
		opts = append(opts, WithStrictMCPConfig())
	}
	if len(cfg.Env) > 0 {
		opts = append(opts, WithEnv(cfg.Env))
	}
	if len(cfg.AddDirs) > 0 {
		opts = append(opts, WithAddDirs(cfg.AddDirs))
	}
	if sessionID := sessionIDFromMetadata(cfg.Session); sessionID != "" {
		if cfg.ResumeSession {
			opts = append(opts, WithResume(sessionID))
		} else {
			opts = append(opts, WithSessionID(sessionID))
		}
	}
	if cfg.ReasoningEffort != "" {
		opts = append(opts, WithEffort(cfg.ReasoningEffort))
	}
	if cfg.Runtime.Providers.Claude != nil {
		claudeCfg := cfg.Runtime.Providers.Claude
		if claudeCfg.DangerouslySkipPermissions {
			opts = append(opts, WithDangerouslySkipPermissions())
		}
		if claudeCfg.PermissionMode != "" {
			opts = append(opts, WithPermissionMode(PermissionMode(claudeCfg.PermissionMode)))
		}
		if len(claudeCfg.SettingSources) > 0 {
			opts = append(opts, WithSettingSources(claudeCfg.SettingSources))
		}
		if claudeCfg.AgentRef != "" {
			opts = append(opts, WithAgent(claudeCfg.AgentRef))
		}
		if len(claudeCfg.InlineAgents) > 0 {
			data, err := json.Marshal(claudeCfg.InlineAgents)
			if err != nil {
				return nil, fmt.Errorf("marshal inline agents: %w", err)
			}
			opts = append(opts, WithAgentsJSON(string(data)))
		}
	}

	return &claudeProviderAdapter{cli: NewClaudeCLI(opts...)}, nil
}

type claudeProviderAdapter struct {
	cli *ClaudeCLI
}

func (a *claudeProviderAdapter) Complete(ctx context.Context, req llmkit.Request) (*llmkit.Response, error) {
	claudeReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
	}
	if len(req.JSONSchema) > 0 {
		claudeReq.JSONSchema = string(req.JSONSchema)
	}

	claudeReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		claudeReq.Messages[i] = convertMessageToClaude(m)
	}

	if len(req.Tools) > 0 {
		claudeReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			claudeReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	resp, err := a.cli.Complete(ctx, claudeReq)
	if err != nil {
		return nil, err
	}
	return a.convertResponse(resp), nil
}

func (a *claudeProviderAdapter) Stream(ctx context.Context, req llmkit.Request) (<-chan llmkit.StreamChunk, error) {
	claudeReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
	}
	if len(req.JSONSchema) > 0 {
		claudeReq.JSONSchema = string(req.JSONSchema)
	}

	claudeReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		claudeReq.Messages[i] = convertMessageToClaude(m)
	}

	if len(req.Tools) > 0 {
		claudeReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			claudeReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	events, result, err := a.cli.StreamJSON(ctx, claudeReq)
	if err != nil {
		return nil, err
	}

	out := make(chan llmkit.StreamChunk)
	go func() {
		defer close(out)

		var totalUsage llmkit.TokenUsage
		var session *llmkit.SessionMetadata

		for event := range events {
			if event.Error != nil {
				out <- llmkit.StreamChunk{Type: "error", Error: event.Error}
				return
			}

			switch event.Type {
			case StreamEventInit:
				session = claudeSession(event.SessionID)
				out <- llmkit.StreamChunk{
					Type:      "session",
					Session:   session,
					SessionID: event.SessionID,
					Model:     event.Init.Model,
					Metadata: map[string]any{
						"cwd":                 event.Init.CWD,
						"permission_mode":     event.Init.PermissionMode,
						"claude_code_version": event.Init.ClaudeCodeVersion,
					},
				}
			case StreamEventAssistant:
				if event.Assistant == nil {
					continue
				}
				if event.Assistant.Text != "" {
					out <- llmkit.StreamChunk{
						Type:      "assistant",
						Content:   event.Assistant.Text,
						MessageID: event.Assistant.MessageID,
						Role:      "assistant",
						Model:     event.Assistant.Model,
						SessionID: event.SessionID,
						Session:   session,
						Usage: &llmkit.TokenUsage{
							InputTokens:              event.Assistant.Usage.InputTokens,
							OutputTokens:             event.Assistant.Usage.OutputTokens,
							TotalTokens:              event.Assistant.Usage.InputTokens + event.Assistant.Usage.OutputTokens,
							CacheCreationInputTokens: event.Assistant.Usage.CacheCreationInputTokens,
							CacheReadInputTokens:     event.Assistant.Usage.CacheReadInputTokens,
						},
					}
				}
				var toolCalls []llmkit.ToolCall
				for _, block := range event.Assistant.Content {
					if block.Type == "tool_use" {
						toolCalls = append(toolCalls, llmkit.ToolCall{
							ID:        block.ID,
							Name:      block.Name,
							Arguments: block.Input,
						})
					}
				}
				if len(toolCalls) > 0 {
					out <- llmkit.StreamChunk{
						Type:      "tool_call",
						Role:      "assistant",
						Model:     event.Assistant.Model,
						SessionID: event.SessionID,
						Session:   session,
						MessageID: event.Assistant.MessageID,
						ToolCalls: toolCalls,
					}
				}

				totalUsage.InputTokens += event.Assistant.Usage.InputTokens
				totalUsage.OutputTokens += event.Assistant.Usage.OutputTokens
				totalUsage.CacheCreationInputTokens += event.Assistant.Usage.CacheCreationInputTokens
				totalUsage.CacheReadInputTokens += event.Assistant.Usage.CacheReadInputTokens
			case StreamEventUser:
				if event.User == nil {
					continue
				}
				content := event.User.GetToolUseResultError()
				var toolResults []llmkit.ToolResult
				for _, block := range event.User.Message.Content {
					output := block.GetContent()
					if output != "" {
						content = output
					}
					toolResults = append(toolResults, llmkit.ToolResult{
						ID:     block.ToolUseID,
						Name:   block.ToolUseID,
						Output: output,
					})
				}
				out <- llmkit.StreamChunk{
					Type:        "tool_result",
					Role:        "tool",
					Content:     content,
					SessionID:   event.User.SessionID,
					Session:     claudeSession(event.User.SessionID),
					ToolResults: toolResults,
				}
			case StreamEventHook:
				if event.Hook == nil {
					continue
				}
				out <- llmkit.StreamChunk{
					Type:      "hook",
					Role:      "system",
					SessionID: event.Hook.SessionID,
					Session:   claudeSession(event.Hook.SessionID),
					Metadata: map[string]any{
						"hook_name":  event.Hook.HookName,
						"hook_event": event.Hook.HookEvent,
						"stdout":     event.Hook.Stdout,
						"stderr":     event.Hook.Stderr,
						"exit_code":  event.Hook.ExitCode,
					},
				}
			}
		}

		final, err := result.Wait(ctx)
		if err != nil {
			out <- llmkit.StreamChunk{Type: "error", Error: err, Done: true, SessionID: llmkit.SessionID(session), Session: session}
			return
		}

		totalUsage.TotalTokens = totalUsage.InputTokens + totalUsage.OutputTokens + totalUsage.CacheCreationInputTokens + totalUsage.CacheReadInputTokens
		finalContent := final.Result
		if len(final.StructuredOutput) > 0 {
			finalContent = string(final.StructuredOutput)
		}
		out <- llmkit.StreamChunk{
			Type:         "final",
			Done:         true,
			SessionID:    final.SessionID,
			Usage:        &totalUsage,
			Session:      claudeSession(final.SessionID),
			FinalContent: finalContent,
			CostUSD:      final.TotalCostUSD,
			NumTurns:     final.NumTurns,
		}

		if final.IsError {
			out <- llmkit.StreamChunk{Type: "error", Error: fmt.Errorf("streaming failed: %s", final.Result), SessionID: final.SessionID, Session: claudeSession(final.SessionID)}
		}
	}()

	return out, nil
}

func (a *claudeProviderAdapter) Provider() string {
	return "claude"
}

func (a *claudeProviderAdapter) Capabilities() llmkit.Capabilities {
	return llmkit.ClaudeCapabilities
}

func (a *claudeProviderAdapter) Close() error {
	return a.cli.Close()
}

func (a *claudeProviderAdapter) convertResponse(resp *CompletionResponse) *llmkit.Response {
	if resp == nil {
		return nil
	}

	out := &llmkit.Response{
		Content:      resp.Content,
		Model:        resp.Model,
		FinishReason: resp.FinishReason,
		Duration:     resp.Duration,
		SessionID:    resp.SessionID,
		Session:      claudeSession(resp.SessionID),
		CostUSD:      resp.CostUSD,
		NumTurns:     resp.NumTurns,
		Usage: llmkit.TokenUsage{
			InputTokens:              resp.Usage.InputTokens,
			OutputTokens:             resp.Usage.OutputTokens,
			TotalTokens:              resp.Usage.TotalTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		},
	}

	if len(resp.ToolCalls) > 0 {
		out.ToolCalls = make([]llmkit.ToolCall, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			out.ToolCalls[i] = llmkit.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			}
		}
	}

	return out
}

func sessionIDFromMetadata(session *llmkit.SessionMetadata) string {
	if session == nil {
		return ""
	}
	var payload struct {
		ID        string `json:"id"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(session.Data, &payload); err != nil {
		return ""
	}
	if payload.SessionID != "" {
		return payload.SessionID
	}
	return payload.ID
}

func claudeSession(sessionID string) *llmkit.SessionMetadata {
	if sessionID == "" {
		return nil
	}
	data, _ := json.Marshal(map[string]string{"session_id": sessionID})
	return &llmkit.SessionMetadata{
		Provider: "claude",
		Data:     data,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mapHeadersToSlice(in map[string]string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for name, value := range in {
		out = append(out, name+": "+value)
	}
	return out
}

func convertMessageToClaude(m llmkit.Message) Message {
	msg := Message{
		Role: Role(m.Role),
		Name: m.Name,
	}

	if m.IsMultimodal() {
		var parts []string
		var imagePaths []string

		for _, part := range m.ContentParts {
			switch part.Type {
			case "text":
				if part.Text != "" {
					parts = append(parts, part.Text)
				}
			case "image":
				if part.FilePath != "" {
					imagePaths = append(imagePaths, part.FilePath)
				}
			case "file":
				if part.FilePath != "" {
					parts = append(parts, "[File: "+part.FilePath+"]")
				}
			}
		}

		if len(parts) > 0 {
			msg.Content = strings.Join(parts, "\n")
		}
		if len(imagePaths) > 0 && msg.Content != "" {
			for _, path := range imagePaths {
				msg.Content += "\n[Image: " + path + "]"
			}
		}
	} else {
		msg.Content = m.Content
	}

	return msg
}
