package claude

// NewFromConfig creates a Client from a Config struct.
// Additional options can be provided to override config values.
//
// Example:
//
//	cfg := claude.Config{
//	    Model:        "claude-opus-4-5-20251101",
//	    SystemPrompt: "You are a code reviewer.",
//	    MaxTurns:     5,
//	}
//
//	client := claude.NewFromConfig(cfg)
func NewFromConfig(cfg Config, opts ...ClaudeOption) Client {
	// Convert config to options
	configOpts := cfg.ToOptions()

	// Combine: config options first, then explicit overrides
	allOpts := make([]ClaudeOption, 0, len(configOpts)+len(opts))
	allOpts = append(allOpts, configOpts...)
	allOpts = append(allOpts, opts...)

	return NewClaudeCLI(allOpts...)
}

// NewFromEnv creates a Client configured from environment variables.
// Additional options can be provided to override env values.
//
// Environment variables checked (CLAUDE_ prefix):
//   - CLAUDE_MODEL: Model name
//   - CLAUDE_FALLBACK_MODEL: Fallback model
//   - CLAUDE_SYSTEM_PROMPT: System prompt
//   - CLAUDE_MAX_TURNS: Max conversation turns
//   - CLAUDE_TIMEOUT: Request timeout (e.g., "5m")
//   - CLAUDE_MAX_BUDGET_USD: Budget limit
//   - CLAUDE_WORK_DIR: Working directory
//   - CLAUDE_PATH: Path to claude binary
//   - CLAUDE_HOME_DIR: Override HOME (for containers)
//   - CLAUDE_CONFIG_DIR: Override .claude directory
//   - CLAUDE_OUTPUT_FORMAT: Output format (json, text, stream-json)
//   - CLAUDE_SKIP_PERMISSIONS: Skip permission prompts (true/1)
//   - CLAUDE_PERMISSION_MODE: Permission mode
//   - CLAUDE_SESSION_ID: Session ID
//   - CLAUDE_NO_SESSION_PERSISTENCE: Disable session saving (true/1)
//
// Example:
//
//	// Set environment:
//	// CLAUDE_MODEL=claude-opus-4-5-20251101
//	// CLAUDE_MAX_TURNS=20
//
//	client := claude.NewFromEnv()
func NewFromEnv(opts ...ClaudeOption) Client {
	cfg := FromEnv()
	return NewFromConfig(cfg, opts...)
}
