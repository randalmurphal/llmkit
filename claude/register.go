package claude

import (
	"context"
	"fmt"
	"strings"

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

		// JSON schema for structured output
		if js := cfg.GetStringOption("json_schema", ""); js != "" {
			opts = append(opts, WithJSONSchema(js))
		}

		// Exact tool set (different from allowed tools whitelist)
		if tools := cfg.GetStringSliceOption("tools"); len(tools) > 0 {
			opts = append(opts, WithTools(tools))
		}

		// Setting sources (project, local, user)
		if sources := cfg.GetStringSliceOption("setting_sources"); len(sources) > 0 {
			opts = append(opts, WithSettingSources(sources))
		}

		// Additional directories for file access
		if addDirs := cfg.GetStringSliceOption("add_dirs"); len(addDirs) > 0 {
			opts = append(opts, WithAddDirs(addDirs))
		}

		// Append to system prompt (without replacing)
		if asp := cfg.GetStringOption("append_system_prompt", ""); asp != "" {
			opts = append(opts, WithAppendSystemPrompt(asp))
		}

		// MCP config file paths (in addition to inline servers)
		if mcpConfigs := cfg.GetStringSliceOption("mcp_config_paths"); len(mcpConfigs) > 0 {
			for _, pathOrJSON := range mcpConfigs {
				opts = append(opts, WithMCPConfig(pathOrJSON))
			}
		} else if mcpConfig := cfg.GetStringOption("mcp_config", ""); mcpConfig != "" {
			opts = append(opts, WithMCPConfig(mcpConfig))
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

	// Convert messages (including multimodal content)
	claudeReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		claudeReq.Messages[i] = convertMessageToClaude(m)
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

	// Convert messages (including multimodal content)
	claudeReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		claudeReq.Messages[i] = convertMessageToClaude(m)
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

	// Call underlying StreamJSON implementation
	events, result, err := a.cli.StreamJSON(ctx, claudeReq)
	if err != nil {
		return nil, err
	}

	// Convert stream events to provider chunks
	providerStream := make(chan provider.StreamChunk)
	go func() {
		defer close(providerStream)

		var totalUsage provider.TokenUsage

		for event := range events {
			if event.Error != nil {
				providerStream <- provider.StreamChunk{Error: event.Error}
				return
			}

			if event.Type == StreamEventAssistant && event.Assistant != nil {
				// Emit text content
				if event.Assistant.Text != "" {
					providerStream <- provider.StreamChunk{Content: event.Assistant.Text}
				}

				// Convert tool calls from content blocks
				var toolCalls []provider.ToolCall
				for _, block := range event.Assistant.Content {
					if block.Type == "tool_use" {
						toolCalls = append(toolCalls, provider.ToolCall{
							ID:        block.ID,
							Name:      block.Name,
							Arguments: block.Input,
						})
					}
				}
				if len(toolCalls) > 0 {
					providerStream <- provider.StreamChunk{ToolCalls: toolCalls}
				}

				// Accumulate usage
				totalUsage.InputTokens += event.Assistant.Usage.InputTokens
				totalUsage.OutputTokens += event.Assistant.Usage.OutputTokens
				totalUsage.CacheCreationInputTokens += event.Assistant.Usage.CacheCreationInputTokens
				totalUsage.CacheReadInputTokens += event.Assistant.Usage.CacheReadInputTokens
			}
		}

		// Wait for final result and emit done chunk with usage
		final, err := result.Wait(ctx)
		if err != nil {
			providerStream <- provider.StreamChunk{Error: err, Done: true}
			return
		}

		totalUsage.TotalTokens = totalUsage.InputTokens + totalUsage.OutputTokens
		providerStream <- provider.StreamChunk{
			Done:  true,
			Usage: &totalUsage,
		}

		// Handle error result
		if final.IsError {
			providerStream <- provider.StreamChunk{
				Error: fmt.Errorf("streaming failed: %s", final.Result),
			}
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
			InputTokens:              resp.Usage.InputTokens,
			OutputTokens:             resp.Usage.OutputTokens,
			TotalTokens:              resp.Usage.TotalTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
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

// convertMessageToClaude converts a provider.Message to claude.Message.
// Handles multimodal content by including image file paths in the content.
func convertMessageToClaude(m provider.Message) Message {
	msg := Message{
		Role: Role(m.Role),
		Name: m.Name,
	}

	// Handle multimodal content
	if m.IsMultimodal() {
		// Claude CLI can read images via file paths
		// For multimodal messages, we need to format the content appropriately
		var parts []string
		var imagePaths []string

		for _, part := range m.ContentParts {
			switch part.Type {
			case "text":
				if part.Text != "" {
					parts = append(parts, part.Text)
				}
			case "image":
				// Claude CLI supports images via file paths
				if part.FilePath != "" {
					imagePaths = append(imagePaths, part.FilePath)
				}
				// Note: ImageURL and ImageBase64 would need to be downloaded/saved
				// to temp files for Claude CLI to read them. For now, we skip them
				// and let the caller handle temp file creation if needed.
			case "file":
				if part.FilePath != "" {
					// For file references, include as context
					parts = append(parts, "[File: "+part.FilePath+"]")
				}
			}
		}

		// Combine text content
		if len(parts) > 0 {
			msg.Content = strings.Join(parts, "\n")
		}

		// If we have image paths, add them to Options for the CLI to handle
		// This allows the CLI wrapper to pass them via appropriate flags
		if len(imagePaths) > 0 && msg.Content != "" {
			// For now, include image references in the content
			// The Claude CLI reads images from the filesystem via the Read tool
			for _, path := range imagePaths {
				msg.Content += "\n[Image: " + path + "]"
			}
		}
	} else {
		msg.Content = m.Content
	}

	return msg
}
