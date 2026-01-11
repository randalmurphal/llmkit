package opencode

import (
	"context"

	"github.com/randalmurphal/llmkit/provider"
)

func init() {
	provider.Register("opencode", newFromProviderConfig)
}

// newFromProviderConfig creates an OpenCodeCLI from a provider.Config.
// This is the factory function registered with the provider registry.
func newFromProviderConfig(cfg provider.Config) (provider.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := make([]OpenCodeOption, 0, 12)

	// Map common config fields
	if cfg.SystemPrompt != "" {
		opts = append(opts, WithSystemPrompt(cfg.SystemPrompt))
	}
	if cfg.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(cfg.MaxTurns))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, WithTimeout(cfg.Timeout))
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

	// Map OpenCode-specific options from Options map
	if cfg.Options != nil {
		// Quiet mode (boolean)
		if cfg.GetBoolOption("quiet", true) {
			opts = append(opts, WithQuiet(true))
		} else {
			opts = append(opts, WithQuiet(false))
		}

		// Agent selection
		if agent := cfg.GetStringOption("agent", ""); agent != "" {
			opts = append(opts, WithAgent(Agent(agent)))
		}

		// Debug mode
		if cfg.GetBoolOption("debug", false) {
			opts = append(opts, WithDebug(true))
		}

		// Output format
		if of := cfg.GetStringOption("output_format", ""); of != "" {
			opts = append(opts, WithOutputFormat(OutputFormat(of)))
		}

		// OpenCode binary path
		if op := cfg.GetStringOption("opencode_path", ""); op != "" {
			opts = append(opts, WithOpenCodePath(op))
		}
	}

	return &openCodeProviderAdapter{
		cli: NewOpenCodeCLI(opts...),
	}, nil
}

// openCodeProviderAdapter wraps OpenCodeCLI to implement provider.Client.
// This adapter converts between opencode package types and provider package types.
type openCodeProviderAdapter struct {
	cli *OpenCodeCLI
}

// Complete implements provider.Client.
func (a *openCodeProviderAdapter) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	// Convert provider.Request to opencode.CompletionRequest
	ocReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}

	// Convert messages
	ocReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		ocReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		ocReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			ocReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	// Call underlying implementation
	resp, err := a.cli.Complete(ctx, ocReq)
	if err != nil {
		return nil, err
	}

	return a.convertResponse(resp), nil
}

// Stream implements provider.Client.
func (a *openCodeProviderAdapter) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	// Convert provider.Request to opencode.CompletionRequest
	ocReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}

	// Convert messages
	ocReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		ocReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		ocReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			ocReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	// Call underlying implementation
	ocStream, err := a.cli.Stream(ctx, ocReq)
	if err != nil {
		return nil, err
	}

	// Convert stream chunks
	providerStream := make(chan provider.StreamChunk)
	go func() {
		defer close(providerStream)
		for chunk := range ocStream {
			providerChunk := provider.StreamChunk{
				Content: chunk.Content,
				Done:    chunk.Done,
				Error:   chunk.Error,
			}

			// Convert tool calls
			if len(chunk.ToolCalls) > 0 {
				providerChunk.ToolCalls = make([]provider.ToolCall, len(chunk.ToolCalls))
				for i, tc := range chunk.ToolCalls {
					providerChunk.ToolCalls[i] = provider.ToolCall{
						ID:        tc.ID,
						Name:      tc.Name,
						Arguments: tc.Arguments,
					}
				}
			}

			// Convert usage
			if chunk.Usage != nil {
				providerChunk.Usage = &provider.TokenUsage{
					InputTokens:  chunk.Usage.InputTokens,
					OutputTokens: chunk.Usage.OutputTokens,
					TotalTokens:  chunk.Usage.TotalTokens,
				}
			}

			providerStream <- providerChunk
		}
	}()

	return providerStream, nil
}

// Provider implements provider.Client.
func (a *openCodeProviderAdapter) Provider() string {
	return "opencode"
}

// Capabilities implements provider.Client.
func (a *openCodeProviderAdapter) Capabilities() provider.Capabilities {
	caps := a.cli.Capabilities()
	return provider.Capabilities{
		Streaming:   caps.Streaming,
		Tools:       caps.Tools,
		MCP:         caps.MCP,
		Sessions:    caps.Sessions,
		Images:      caps.Images,
		NativeTools: caps.NativeTools,
		ContextFile: caps.ContextFile,
	}
}

// Close implements provider.Client.
func (a *openCodeProviderAdapter) Close() error {
	return a.cli.Close()
}

// convertResponse converts opencode.CompletionResponse to provider.Response.
func (a *openCodeProviderAdapter) convertResponse(resp *CompletionResponse) *provider.Response {
	if resp == nil {
		return nil
	}

	providerResp := &provider.Response{
		Content:      resp.Content,
		Model:        resp.Model,
		FinishReason: resp.FinishReason,
		Duration:     resp.Duration,
		Usage: provider.TokenUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	if len(resp.ToolCalls) > 0 {
		providerResp.ToolCalls = make([]provider.ToolCall, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			providerResp.ToolCalls[i] = provider.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			}
		}
	}

	return providerResp
}
