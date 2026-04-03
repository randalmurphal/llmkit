package codex

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/randalmurphal/llmkit/v2"
)

func init() {
	llmkit.Register("codex", newFromProviderConfig)
	llmkit.RegisterProviderDefinition(llmkit.ProviderDefinition{
		Name:      "codex",
		Supported: true,
		Shared: llmkit.SharedSupport{
			SystemPrompt:       true,
			AppendSystemPrompt: false,
			AllowedTools:       false,
			DisallowedTools:    false,
			Tools:              false,
			MCPServers:         true,
			StrictMCPConfig:    false,
			MaxBudgetUSD:       false,
			MaxTurns:           false,
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

	opts := make([]CodexOption, 0, 8)
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
	}
	if cfg.BinaryPath != "" {
		opts = append(opts, WithCodexPath(cfg.BinaryPath))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, WithTimeout(cfg.Timeout))
	}
	if cfg.WorkDir != "" {
		opts = append(opts, WithWorkdir(cfg.WorkDir))
	}
	if len(cfg.Env) > 0 {
		opts = append(opts, WithEnv(cfg.Env))
	}
	if len(cfg.AddDirs) > 0 {
		opts = append(opts, WithAddDirs(cfg.AddDirs))
	}
	if sessionID := sessionIDFromMetadata(cfg.Session); sessionID != "" {
		if !cfg.ResumeSession {
			return nil, fmt.Errorf("%w: codex client only supports resuming existing sessions", llmkit.ErrUnsupportedFeature)
		}
		opts = append(opts, WithSessionID(sessionID))
	}
	reasoningEffort := cfg.ReasoningEffort
	webSearchMode := cfg.WebSearchMode
	if cfg.Runtime.Providers.Codex != nil {
		codexCfg := cfg.Runtime.Providers.Codex
		if codexCfg.ReasoningEffort != "" {
			reasoningEffort = codexCfg.ReasoningEffort
		}
		if codexCfg.WebSearchMode != "" {
			webSearchMode = codexCfg.WebSearchMode
		}
		if codexCfg.BypassApprovalsAndSandbox {
			opts = append(opts, WithDangerouslyBypassApprovalsAndSandbox())
		} else {
			if codexCfg.SandboxMode != "" {
				opts = append(opts, WithSandboxMode(SandboxMode(codexCfg.SandboxMode)))
			}
			if codexCfg.ApprovalMode != "" {
				opts = append(opts, WithApprovalMode(ApprovalMode(codexCfg.ApprovalMode)))
			}
		}
	}
	if reasoningEffort != "" {
		opts = append(opts, WithReasoningEffort(reasoningEffort))
	}
	if webSearchMode != "" {
		opts = append(opts, WithWebSearchMode(WebSearchMode(webSearchMode)))
	}

	return &codexProviderAdapter{cli: NewCodexCLI(opts...)}, nil
}

type codexProviderAdapter struct {
	cli *CodexCLI
}

func (a *codexProviderAdapter) Complete(ctx context.Context, req llmkit.Request) (*llmkit.Response, error) {
	codexReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		JSONSchema:   req.JSONSchema,
	}

	codexReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		codexReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	if len(req.Tools) > 0 {
		codexReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			codexReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	resp, err := a.cli.Complete(ctx, codexReq)
	if err != nil {
		return nil, err
	}

	return a.convertResponse(resp), nil
}

func (a *codexProviderAdapter) Stream(ctx context.Context, req llmkit.Request) (<-chan llmkit.StreamChunk, error) {
	codexReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		JSONSchema:   req.JSONSchema,
	}

	codexReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		codexReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	if len(req.Tools) > 0 {
		codexReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			codexReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	codexStream, err := a.cli.Stream(ctx, codexReq)
	if err != nil {
		return nil, err
	}

	out := make(chan llmkit.StreamChunk)
	go func() {
		defer close(out)
		for chunk := range codexStream {
			session := codexSession(chunk.SessionID)
			if chunk.Content != "" {
				out <- llmkit.StreamChunk{
					Type:      "assistant",
					Content:   chunk.Content,
					Role:      "assistant",
					SessionID: chunk.SessionID,
					Session:   session,
				}
			}
			if len(chunk.ToolCalls) > 0 {
				toolCalls := make([]llmkit.ToolCall, len(chunk.ToolCalls))
				for i, tc := range chunk.ToolCalls {
					toolCalls[i] = llmkit.ToolCall{
						ID:        tc.ID,
						Name:      tc.Name,
						Arguments: tc.Arguments,
					}
				}
				out <- llmkit.StreamChunk{
					Type:      "tool_call",
					Role:      "assistant",
					SessionID: chunk.SessionID,
					Session:   session,
					ToolCalls: toolCalls,
				}
			}
			if len(chunk.ToolResults) > 0 {
				toolResults := make([]llmkit.ToolResult, len(chunk.ToolResults))
				for i, tr := range chunk.ToolResults {
					toolResults[i] = llmkit.ToolResult{
						ID:       tr.ID,
						Name:     tr.Name,
						Output:   tr.Output,
						Status:   tr.Status,
						ExitCode: tr.ExitCode,
					}
				}
				out <- llmkit.StreamChunk{
					Type:        "tool_result",
					Role:        "tool",
					SessionID:   chunk.SessionID,
					Session:     session,
					ToolResults: toolResults,
				}
			}
			if chunk.Usage != nil || chunk.FinalContent != "" || chunk.Done {
				finalContent := chunk.FinalContent
				if finalContent == "" && chunk.Done {
					finalContent = chunk.Content
				}
				converted := llmkit.StreamChunk{
					Type:         "final",
					FinalContent: finalContent,
					SessionID:    chunk.SessionID,
					Session:      session,
					Done:         chunk.Done,
					Error:        chunk.Error,
				}
				if chunk.Usage != nil {
					converted.Usage = &llmkit.TokenUsage{
						InputTokens:              chunk.Usage.InputTokens,
						OutputTokens:             chunk.Usage.OutputTokens,
						TotalTokens:              chunk.Usage.TotalTokens,
						CacheCreationInputTokens: chunk.Usage.CacheCreationInputTokens,
						CacheReadInputTokens:     chunk.Usage.CacheReadInputTokens,
					}
				}
				out <- converted
			}
			if chunk.Error != nil {
				out <- llmkit.StreamChunk{
					Type:      "error",
					SessionID: chunk.SessionID,
					Session:   session,
					Error:     chunk.Error,
					Done:      chunk.Done,
				}
			}
		}
	}()

	return out, nil
}

func (a *codexProviderAdapter) Provider() string {
	return "codex"
}

func (a *codexProviderAdapter) Capabilities() llmkit.Capabilities {
	return llmkit.CodexCapabilities
}

func (a *codexProviderAdapter) Close() error {
	return a.cli.Close()
}

func (a *codexProviderAdapter) convertResponse(resp *CompletionResponse) *llmkit.Response {
	if resp == nil {
		return nil
	}

	out := &llmkit.Response{
		Content:      resp.Content,
		Model:        resp.Model,
		FinishReason: resp.FinishReason,
		Duration:     resp.Duration,
		SessionID:    resp.SessionID,
		Session:      codexSession(resp.SessionID),
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

func codexSession(sessionID string) *llmkit.SessionMetadata {
	if sessionID == "" {
		return nil
	}
	data, _ := json.Marshal(map[string]string{"session_id": sessionID})
	return &llmkit.SessionMetadata{
		Provider: "codex",
		Data:     data,
	}
}
