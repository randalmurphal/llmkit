package gemini

import (
	"context"

	"github.com/randalmurphal/llmkit/claudeconfig"
	"github.com/randalmurphal/llmkit/provider"
)

func init() {
	provider.Register("gemini", newFromProviderConfig)
}

// newFromProviderConfig creates a GeminiCLI from a provider.Config.
// This is the factory function registered with the provider registry.
func newFromProviderConfig(cfg provider.Config) (provider.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := make([]GeminiOption, 0, 16)

	// Map common config fields
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
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

	// Map Gemini-specific options from Options map
	if cfg.Options != nil {
		// Yolo mode (auto-approve)
		if cfg.GetBoolOption("yolo", false) {
			opts = append(opts, WithYolo())
		}

		// Output format
		if of := cfg.GetStringOption("output_format", ""); of != "" {
			opts = append(opts, WithOutputFormat(OutputFormat(of)))
		}

		// Gemini binary path
		if gp := cfg.GetStringOption("gemini_path", ""); gp != "" {
			opts = append(opts, WithGeminiPath(gp))
		}

		// Sandbox mode
		if sb := cfg.GetStringOption("sandbox", ""); sb != "" {
			opts = append(opts, WithSandbox(sb))
		}

		// Include directories
		if dirs, ok := cfg.Options["include_dirs"].([]string); ok && len(dirs) > 0 {
			opts = append(opts, WithIncludeDirs(dirs))
		}
		// Also handle []interface{} (from JSON unmarshaling)
		if dirs, ok := cfg.Options["include_dirs"].([]interface{}); ok && len(dirs) > 0 {
			strDirs := make([]string, 0, len(dirs))
			for _, d := range dirs {
				if s, ok := d.(string); ok {
					strDirs = append(strDirs, s)
				}
			}
			if len(strDirs) > 0 {
				opts = append(opts, WithIncludeDirs(strDirs))
			}
		}
	}

	// Handle MCP configuration
	if cfg.MCP != nil && len(cfg.MCP.MCPServers) > 0 {
		mcpServers := convertMCPConfig(cfg.MCP)
		if len(mcpServers) > 0 {
			opts = append(opts, WithMCPServers(mcpServers))
		}
	}

	return &geminiProviderAdapter{
		cli: NewGeminiCLI(opts...),
	}, nil
}

// geminiProviderAdapter wraps GeminiCLI to implement provider.Client.
// This adapter converts between gemini package types and provider package types.
type geminiProviderAdapter struct {
	cli *GeminiCLI
}

// Complete implements provider.Client.
func (a *geminiProviderAdapter) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	// Convert provider.Request to gemini.CompletionRequest
	geminiReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}

	// Convert messages
	geminiReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		msg := Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
		// Handle multimodal content
		if len(m.ContentParts) > 0 {
			msg.ContentParts = make([]ContentPart, len(m.ContentParts))
			for j, cp := range m.ContentParts {
				msg.ContentParts[j] = ContentPart{
					Type:        cp.Type,
					Text:        cp.Text,
					ImageURL:    cp.ImageURL,
					ImageBase64: cp.ImageBase64,
					MediaType:   cp.MediaType,
					FilePath:    cp.FilePath,
				}
			}
		}
		geminiReq.Messages[i] = msg
	}

	// Convert tools
	if len(req.Tools) > 0 {
		geminiReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			geminiReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	// Call underlying implementation
	resp, err := a.cli.Complete(ctx, geminiReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	return a.convertResponse(resp), nil
}

// Stream implements provider.Client.
func (a *geminiProviderAdapter) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	// Convert provider.Request to gemini.CompletionRequest
	geminiReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}

	// Convert messages
	geminiReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		msg := Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
		// Handle multimodal content
		if len(m.ContentParts) > 0 {
			msg.ContentParts = make([]ContentPart, len(m.ContentParts))
			for j, cp := range m.ContentParts {
				msg.ContentParts[j] = ContentPart{
					Type:        cp.Type,
					Text:        cp.Text,
					ImageURL:    cp.ImageURL,
					ImageBase64: cp.ImageBase64,
					MediaType:   cp.MediaType,
					FilePath:    cp.FilePath,
				}
			}
		}
		geminiReq.Messages[i] = msg
	}

	// Convert tools
	if len(req.Tools) > 0 {
		geminiReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			geminiReq.Tools[i] = Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}
		}
	}

	// Call underlying implementation
	geminiStream, err := a.cli.Stream(ctx, geminiReq)
	if err != nil {
		return nil, err
	}

	// Convert stream chunks
	providerStream := make(chan provider.StreamChunk)
	go func() {
		defer close(providerStream)
		for chunk := range geminiStream {
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
func (a *geminiProviderAdapter) Provider() string {
	return "gemini"
}

// Capabilities implements provider.Client.
func (a *geminiProviderAdapter) Capabilities() provider.Capabilities {
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
func (a *geminiProviderAdapter) Close() error {
	return a.cli.Close()
}

// convertResponse converts gemini.CompletionResponse to provider.Response.
func (a *geminiProviderAdapter) convertResponse(resp *CompletionResponse) *provider.Response {
	if resp == nil {
		return nil
	}

	providerResp := &provider.Response{
		Content:      resp.Content,
		Model:        resp.Model,
		FinishReason: resp.FinishReason,
		Duration:     resp.Duration,
		NumTurns:     resp.NumTurns,
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

// convertMCPConfig converts claudeconfig.MCPConfig to gemini.MCPServerConfig map.
// Handles all transport types (stdio, http, sse) and skips disabled servers.
func convertMCPConfig(mcp *claudeconfig.MCPConfig) map[string]MCPServerConfig {
	if mcp == nil || len(mcp.MCPServers) == 0 {
		return nil
	}

	mcpServers := make(map[string]MCPServerConfig, len(mcp.MCPServers))
	for name, server := range mcp.MCPServers {
		if server == nil || server.Disabled {
			continue
		}
		mcpServers[name] = MCPServerConfig{
			Type:    server.Type,
			Command: server.Command,
			Args:    server.Args,
			Env:     server.Env,
			URL:     server.URL,
			Headers: server.Headers,
		}
	}
	return mcpServers
}
