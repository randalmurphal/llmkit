package claude

import (
	"context"

	"github.com/randalmurphal/llmkit/claudeconfig"
	"github.com/randalmurphal/llmkit/provider"
)

func init() {
	provider.Register("claude", newFromProviderConfig)
}

// newFromProviderConfig creates a ClaudeCLI from a provider.Config.
// This is the factory function registered with the provider registry.
func newFromProviderConfig(cfg provider.Config) (provider.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := make([]ClaudeOption, 0, 16)

	// Map common config fields
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

	// Map Claude-specific options from Options map
	if cfg.Options != nil {
		// Permission mode
		if pm := cfg.GetStringOption("permission_mode", ""); pm != "" {
			opts = append(opts, WithPermissionMode(PermissionMode(pm)))
		}

		// Skip permissions (boolean)
		if cfg.GetBoolOption("skip_permissions", false) {
			opts = append(opts, WithDangerouslySkipPermissions())
		}

		// Session ID
		if sid := cfg.GetStringOption("session_id", ""); sid != "" {
			opts = append(opts, WithSessionID(sid))
		}

		// Home directory (for containers)
		if hd := cfg.GetStringOption("home_dir", ""); hd != "" {
			opts = append(opts, WithHomeDir(hd))
		}

		// Config directory
		if cd := cfg.GetStringOption("config_dir", ""); cd != "" {
			opts = append(opts, WithConfigDir(cd))
		}

		// Output format
		if of := cfg.GetStringOption("output_format", ""); of != "" {
			opts = append(opts, WithOutputFormat(OutputFormat(of)))
		}

		// Claude binary path
		if cp := cfg.GetStringOption("claude_path", ""); cp != "" {
			opts = append(opts, WithClaudePath(cp))
		}

		// Continue session
		if cfg.GetBoolOption("continue", false) {
			opts = append(opts, WithContinue())
		}

		// Resume session
		if rs := cfg.GetStringOption("resume", ""); rs != "" {
			opts = append(opts, WithResume(rs))
		}

		// No session persistence
		if cfg.GetBoolOption("no_session_persistence", false) {
			opts = append(opts, WithNoSessionPersistence())
		}
	}

	// Handle MCP configuration
	if cfg.MCP != nil && len(cfg.MCP.MCPServers) > 0 {
		mcpServers := convertMCPConfig(cfg.MCP)
		if len(mcpServers) > 0 {
			opts = append(opts, WithMCPServers(mcpServers))
		}
	}

	// Handle StrictMCPConfig option
	if cfg.GetBoolOption("strict_mcp_config", false) {
		opts = append(opts, WithStrictMCPConfig())
	}

	return &claudeProviderAdapter{
		cli: NewClaudeCLI(opts...),
	}, nil
}

// claudeProviderAdapter wraps ClaudeCLI to implement provider.Client.
// This adapter converts between claude package types and provider package types.
type claudeProviderAdapter struct {
	cli *ClaudeCLI
}

// Complete implements provider.Client.
func (a *claudeProviderAdapter) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	// Convert provider.Request to claude.CompletionRequest
	claudeReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}

	// Convert messages
	claudeReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		claudeReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	// Convert tools
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

	// Call underlying implementation
	resp, err := a.cli.Complete(ctx, claudeReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	return a.convertResponse(resp), nil
}

// Stream implements provider.Client.
func (a *claudeProviderAdapter) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	// Convert provider.Request to claude.CompletionRequest
	claudeReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}

	// Convert messages
	claudeReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		claudeReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	// Convert tools
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

	// Call underlying implementation
	claudeStream, err := a.cli.Stream(ctx, claudeReq)
	if err != nil {
		return nil, err
	}

	// Convert stream chunks
	providerStream := make(chan provider.StreamChunk)
	go func() {
		defer close(providerStream)
		for chunk := range claudeStream {
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
					InputTokens:             chunk.Usage.InputTokens,
					OutputTokens:            chunk.Usage.OutputTokens,
					TotalTokens:             chunk.Usage.TotalTokens,
					CacheCreationInputTokens: chunk.Usage.CacheCreationInputTokens,
					CacheReadInputTokens:    chunk.Usage.CacheReadInputTokens,
				}
			}

			providerStream <- providerChunk
		}
	}()

	return providerStream, nil
}

// Provider implements provider.Client.
func (a *claudeProviderAdapter) Provider() string {
	return "claude"
}

// Capabilities implements provider.Client.
func (a *claudeProviderAdapter) Capabilities() provider.Capabilities {
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
func (a *claudeProviderAdapter) Close() error {
	return a.cli.Close()
}

// convertResponse converts claude.CompletionResponse to provider.Response.
func (a *claudeProviderAdapter) convertResponse(resp *CompletionResponse) *provider.Response {
	if resp == nil {
		return nil
	}

	providerResp := &provider.Response{
		Content:      resp.Content,
		Model:        resp.Model,
		FinishReason: resp.FinishReason,
		Duration:     resp.Duration,
		SessionID:    resp.SessionID,
		CostUSD:      resp.CostUSD,
		NumTurns:     resp.NumTurns,
		Usage: provider.TokenUsage{
			InputTokens:             resp.Usage.InputTokens,
			OutputTokens:            resp.Usage.OutputTokens,
			TotalTokens:             resp.Usage.TotalTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:    resp.Usage.CacheReadInputTokens,
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

// convertMCPConfig converts claudeconfig.MCPConfig to claude.MCPServerConfig map.
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
