package codex

import (
	"context"

	"github.com/randalmurphal/llmkit/provider"
)

func init() {
	provider.Register("codex", newFromProviderConfig)
}

// newFromProviderConfig creates a CodexCLI from a provider.Config.
// This is the factory function registered with the provider registry.
func newFromProviderConfig(cfg provider.Config) (provider.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := make([]CodexOption, 0, 24)

	// Map common config fields
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

	// Map Codex-specific options from Options map
	if cfg.Options != nil {
		// Sandbox mode
		if sm := cfg.GetStringOption("sandbox", ""); sm != "" {
			opts = append(opts, WithSandboxMode(SandboxMode(sm)))
		}

		// Approval mode
		if am := cfg.GetStringOption("ask_for_approval", ""); am != "" {
			opts = append(opts, WithApprovalMode(ApprovalMode(am)))
		}

		// Full auto mode
		if cfg.GetBoolOption("full_auto", false) {
			opts = append(opts, WithFullAuto())
		}
		if cfg.GetBoolOption("yolo", false) || cfg.GetBoolOption("dangerously_bypass_approvals_and_sandbox", false) {
			opts = append(opts, WithDangerouslyBypassApprovalsAndSandbox())
		}

		// Session ID
		if sid := cfg.GetStringOption("session_id", ""); sid != "" {
			opts = append(opts, WithSessionID(sid))
		}
		if cfg.GetBoolOption("resume_all", false) {
			opts = append(opts, WithResumeAll())
		}

		// Search
		if webSearch := cfg.GetStringOption("web_search", ""); webSearch != "" {
			opts = append(opts, WithWebSearchMode(WebSearchMode(webSearch)))
		} else if webSearch := cfg.GetStringOption("web_search_mode", ""); webSearch != "" {
			opts = append(opts, WithWebSearchMode(WebSearchMode(webSearch)))
		} else if cfg.GetBoolOption("search", false) {
			opts = append(opts, WithSearch())
		}
		if cfg.GetBoolOption("enable_search", false) {
			opts = append(opts, WithSearch())
		}

		// Profile and config overrides
		if profile := cfg.GetStringOption("profile", ""); profile != "" {
			opts = append(opts, WithProfile(profile))
		}
		if localProvider := cfg.GetStringOption("local_provider", ""); localProvider != "" {
			opts = append(opts, WithLocalProvider(localProvider))
		}
		if cfg.GetBoolOption("skip_git_repo_check", false) {
			opts = append(opts, WithSkipGitRepoCheck())
		}
		if outputSchema := cfg.GetStringOption("output_schema", ""); outputSchema != "" {
			opts = append(opts, WithOutputSchema(outputSchema))
		}
		if outputLastMessage := cfg.GetStringOption("output_last_message", ""); outputLastMessage != "" {
			opts = append(opts, WithOutputLastMessage(outputLastMessage))
		}
		if effort := cfg.GetStringOption("model_reasoning_effort", ""); effort != "" {
			opts = append(opts, WithReasoningEffort(effort))
		}
		if cfg.GetBoolOption("hide_agent_reasoning", false) {
			opts = append(opts, WithHideAgentReasoning())
		}
		if cfg.GetBoolOption("oss", false) || cfg.GetBoolOption("use_oss", false) {
			opts = append(opts, WithOSS())
		}
		if color := cfg.GetStringOption("color", ""); color != "" {
			opts = append(opts, WithColorMode(color))
		} else if color := cfg.GetStringOption("color_mode", ""); color != "" {
			opts = append(opts, WithColorMode(color))
		}
		if features := cfg.GetStringSliceOption("enable_features"); len(features) > 0 {
			opts = append(opts, WithEnabledFeatures(features))
		}
		if features := cfg.GetStringSliceOption("disable_features"); len(features) > 0 {
			opts = append(opts, WithDisabledFeatures(features))
		}

		// Codex binary path
		if cp := cfg.GetStringOption("codex_path", ""); cp != "" {
			opts = append(opts, WithCodexPath(cp))
		}

		// Additional directories
		if addDirs, ok := cfg.Options["add_dirs"].([]string); ok && len(addDirs) > 0 {
			opts = append(opts, WithAddDirs(addDirs))
		} else if addDirs, ok := cfg.Options["add_dirs"].([]any); ok && len(addDirs) > 0 {
			dirs := make([]string, 0, len(addDirs))
			for _, d := range addDirs {
				if s, ok := d.(string); ok {
					dirs = append(dirs, s)
				}
			}
			if len(dirs) > 0 {
				opts = append(opts, WithAddDirs(dirs))
			}
		}

		// Images
		if images, ok := cfg.Options["images"].([]string); ok && len(images) > 0 {
			opts = append(opts, WithImages(images))
		} else if images, ok := cfg.Options["images"].([]any); ok && len(images) > 0 {
			imgs := make([]string, 0, len(images))
			for _, i := range images {
				if s, ok := i.(string); ok {
					imgs = append(imgs, s)
				}
			}
			if len(imgs) > 0 {
				opts = append(opts, WithImages(imgs))
			}
		}

		// Config overrides map
		if overrides, ok := cfg.Options["config_overrides"].(map[string]any); ok && len(overrides) > 0 {
			opts = append(opts, WithConfigOverrides(overrides))
		} else if raw, ok := cfg.Options["config_overrides"].(map[string]string); ok && len(raw) > 0 {
			overrides := make(map[string]any, len(raw))
			for k, v := range raw {
				overrides[k] = v
			}
			opts = append(opts, WithConfigOverrides(overrides))
		}
	}

	return &codexProviderAdapter{
		cli: NewCodexCLI(opts...),
	}, nil
}

// codexProviderAdapter wraps CodexCLI to implement provider.Client.
type codexProviderAdapter struct {
	cli *CodexCLI
}

// Complete implements provider.Client.
func (a *codexProviderAdapter) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	// Convert provider.Request to codex.CompletionRequest
	codexReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}
	applyRequestOverrides(&codexReq)

	// Convert messages
	codexReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		codexReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	// Convert tools
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

	// Call underlying implementation
	resp, err := a.cli.Complete(ctx, codexReq)
	if err != nil {
		return nil, err
	}

	return a.convertResponse(resp), nil
}

// Stream implements provider.Client.
func (a *codexProviderAdapter) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	// Convert provider.Request to codex.CompletionRequest
	codexReq := CompletionRequest{
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Options:      req.Options,
	}
	applyRequestOverrides(&codexReq)

	// Convert messages
	codexReq.Messages = make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		codexReq.Messages[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
			Name:    m.Name,
		}
	}

	// Convert tools
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

	// Call underlying implementation
	codexStream, err := a.cli.Stream(ctx, codexReq)
	if err != nil {
		return nil, err
	}

	// Convert stream chunks
	providerStream := make(chan provider.StreamChunk)
	go func() {
		defer close(providerStream)
		for chunk := range codexStream {
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
func (a *codexProviderAdapter) Provider() string {
	return "codex"
}

// Capabilities implements provider.Client.
func (a *codexProviderAdapter) Capabilities() provider.Capabilities {
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
func (a *codexProviderAdapter) Close() error {
	return a.cli.Close()
}

// convertResponse converts codex.CompletionResponse to provider.Response.
func (a *codexProviderAdapter) convertResponse(resp *CompletionResponse) *provider.Response {
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

func applyRequestOverrides(req *CompletionRequest) {
	if req == nil || req.Options == nil {
		return
	}

	if webSearch, ok := req.Options["web_search"].(string); ok && webSearch != "" {
		req.WebSearchMode = WebSearchMode(webSearch)
	} else if webSearch, ok := req.Options["web_search_mode"].(string); ok && webSearch != "" {
		req.WebSearchMode = WebSearchMode(webSearch)
	}
	if outputSchema, ok := req.Options["output_schema"].(string); ok && outputSchema != "" {
		req.OutputSchemaPath = outputSchema
	}
	if outputLast, ok := req.Options["output_last_message"].(string); ok && outputLast != "" {
		req.OutputLastMessagePath = outputLast
	}

	if overrides, ok := req.Options["config_overrides"].(map[string]any); ok && len(overrides) > 0 {
		req.ConfigOverrides = overrides
		return
	}
	if raw, ok := req.Options["config_overrides"].(map[string]string); ok && len(raw) > 0 {
		req.ConfigOverrides = make(map[string]any, len(raw))
		for k, v := range raw {
			req.ConfigOverrides[k] = v
		}
	}
}
