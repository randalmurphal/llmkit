package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/randalmurphal/llmkit/v2"
)

func init() {
	llmkit.Register("claude", newFromProviderConfig)
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
	if cfg.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(cfg.MaxTurns))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, WithTimeout(cfg.Timeout))
	}
	if cfg.MaxBudgetUSD > 0 {
		opts = append(opts, WithMaxBudgetUSD(cfg.MaxBudgetUSD))
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
	if len(cfg.Env) > 0 {
		opts = append(opts, WithEnv(cfg.Env))
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

		for event := range events {
			if event.Error != nil {
				out <- llmkit.StreamChunk{Error: event.Error}
				return
			}

			if event.Type != StreamEventAssistant || event.Assistant == nil {
				continue
			}

			if event.Assistant.Text != "" {
				out <- llmkit.StreamChunk{Content: event.Assistant.Text}
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
				out <- llmkit.StreamChunk{ToolCalls: toolCalls}
			}

			totalUsage.InputTokens += event.Assistant.Usage.InputTokens
			totalUsage.OutputTokens += event.Assistant.Usage.OutputTokens
			totalUsage.CacheCreationInputTokens += event.Assistant.Usage.CacheCreationInputTokens
			totalUsage.CacheReadInputTokens += event.Assistant.Usage.CacheReadInputTokens
		}

		final, err := result.Wait(ctx)
		if err != nil {
			out <- llmkit.StreamChunk{Error: err, Done: true}
			return
		}

		totalUsage.TotalTokens = totalUsage.InputTokens + totalUsage.OutputTokens
		out <- llmkit.StreamChunk{Done: true, Usage: &totalUsage}

		if final.IsError {
			out <- llmkit.StreamChunk{Error: fmt.Errorf("streaming failed: %s", final.Result)}
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
