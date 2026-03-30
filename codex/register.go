package codex

import (
	"context"
	"os"

	"github.com/randalmurphal/llmkit/v2"
)

func init() {
	llmkit.Register("codex", newFromProviderConfig)
}

func newFromProviderConfig(cfg llmkit.Config) (llmkit.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := make([]CodexOption, 0, 8)
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
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
	}
	cleanup := func() {}
	if len(req.JSONSchema) > 0 {
		path, err := writeSchemaFile(req.JSONSchema)
		if err != nil {
			return nil, err
		}
		codexReq.OutputSchemaPath = path
		cleanup = func() { _ = os.Remove(path) }
	}
	defer cleanup()

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
	}
	schemaPath := ""
	if len(req.JSONSchema) > 0 {
		path, err := writeSchemaFile(req.JSONSchema)
		if err != nil {
			return nil, err
		}
		codexReq.OutputSchemaPath = path
		schemaPath = path
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
		if schemaPath != "" {
			_ = os.Remove(schemaPath)
		}
		return nil, err
	}

	out := make(chan llmkit.StreamChunk)
	go func() {
		defer close(out)
		if schemaPath != "" {
			defer func() { _ = os.Remove(schemaPath) }()
		}
		for chunk := range codexStream {
			converted := llmkit.StreamChunk{
				Content:      chunk.Content,
				FinalContent: chunk.FinalContent,
				SessionID:    chunk.SessionID,
				Done:         chunk.Done,
				Error:        chunk.Error,
			}

			if len(chunk.ToolCalls) > 0 {
				converted.ToolCalls = make([]llmkit.ToolCall, len(chunk.ToolCalls))
				for i, tc := range chunk.ToolCalls {
					converted.ToolCalls[i] = llmkit.ToolCall{
						ID:        tc.ID,
						Name:      tc.Name,
						Arguments: tc.Arguments,
					}
				}
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
