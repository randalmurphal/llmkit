package provider

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/randalmurphal/llmkit/claudeconfig"
)

// Config holds configuration for creating an LLM provider client.
// Common fields apply to all providers; use Options for provider-specific settings.
type Config struct {
	// --- Provider Selection ---

	// Provider is the name of the provider to use.
	// Required. Values: "claude", "gemini", "codex", "opencode", "local"
	Provider string `json:"provider" yaml:"provider" mapstructure:"provider"`

	// --- Model Selection ---

	// Model is the model to use (provider-specific name).
	// Examples: "claude-sonnet-4-20250514", "gemini-2.5-pro", "gpt-5-codex"
	Model string `json:"model" yaml:"model" mapstructure:"model"`

	// FallbackModel is used when the primary model is unavailable.
	// Optional.
	FallbackModel string `json:"fallback_model" yaml:"fallback_model" mapstructure:"fallback_model"`

	// --- Prompts ---

	// SystemPrompt is the system message prepended to all requests.
	// Optional.
	SystemPrompt string `json:"system_prompt" yaml:"system_prompt" mapstructure:"system_prompt"`

	// --- Execution Limits ---

	// MaxTurns limits conversation turns (tool calls + responses).
	// 0 means no limit. Default varies by provider.
	MaxTurns int `json:"max_turns" yaml:"max_turns" mapstructure:"max_turns"`

	// Timeout is the maximum duration for a completion request.
	// 0 uses the provider default.
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`

	// MaxBudgetUSD limits spending per request.
	// 0 means no limit. Not all providers support this.
	MaxBudgetUSD float64 `json:"max_budget_usd" yaml:"max_budget_usd" mapstructure:"max_budget_usd"`

	// --- Working Directory ---

	// WorkDir is the working directory for file operations.
	// Default: current directory.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`

	// --- Tool Control ---

	// AllowedTools limits which tools the model can use.
	// Empty means all tools allowed. Tool names are provider-specific.
	AllowedTools []string `json:"allowed_tools" yaml:"allowed_tools" mapstructure:"allowed_tools"`

	// DisallowedTools explicitly blocks certain tools.
	// Takes precedence over AllowedTools.
	DisallowedTools []string `json:"disallowed_tools" yaml:"disallowed_tools" mapstructure:"disallowed_tools"`

	// --- MCP Configuration ---

	// MCP configures MCP servers to enable.
	// MCP is the universal tool extension mechanism supported by all providers.
	MCP *claudeconfig.MCPConfig `json:"mcp" yaml:"mcp" mapstructure:"mcp"`

	// --- Environment ---

	// Env provides additional environment variables for CLI execution.
	Env map[string]string `json:"env" yaml:"env" mapstructure:"env"`

	// --- Provider-Specific Options ---

	// Options holds provider-specific configuration.
	// See each provider's documentation for available options.
	//
	// Common options by provider:
	//
	// Claude:
	//   - "permission_mode": "acceptEdits" | "bypassPermissions"
	//   - "skip_permissions": bool (dangerously skip all permission prompts)
	//   - "session_id": string
	//   - "home_dir": string (for containers)
	//   - "config_dir": string
	//   - "output_format": "json" | "stream-json" | "text"
	//   - "claude_path": string (path to claude binary)
	//   - "continue": bool (continue most recent session)
	//   - "resume": string (resume specific session ID)
	//   - "no_session_persistence": bool
	//   - "json_schema": string (JSON schema for structured output)
	//   - "tools": []string (exact tool set, different from allowed_tools)
	//   - "setting_sources": []string (e.g., ["project", "local", "user"])
	//   - "add_dirs": []string (additional directories for file access)
	//   - "append_system_prompt": string (append to system prompt)
	//   - "mcp_config": string (MCP config file path or JSON)
	//   - "mcp_config_paths": []string (multiple MCP config paths)
	//   - "strict_mcp_config": bool
	//
	// Gemini:
	//   - "include_directories": []string
	//   - "yolo": bool (auto-approve all actions)
	//   - "sandbox": string (docker/podman/custom, via GEMINI_SANDBOX env)
	//
	// Codex:
	//   - "sandbox": "read-only" | "workspace-write" | "danger-full-access"
	//   - "ask_for_approval": "untrusted" | "on-failure" | "on-request" | "never"
	//   - "search": bool
	//   - "full_auto": bool
	//
	// OpenCode:
	//   - "quiet": bool
	//   - "agent": "build" | "plan"
	//   - "debug": bool
	//
	// Local:
	//   - "backend": "ollama" | "llama.cpp" | "vllm"
	//   - "sidecar_path": string
	Options map[string]any `json:"options" yaml:"options" mapstructure:"options"`
}

// DefaultConfig returns a Config with sensible defaults.
// Provider must still be set before use.
func DefaultConfig() Config {
	return Config{
		MaxTurns: 10,
		Timeout:  5 * time.Minute,
	}
}

// LoadFromEnv populates config fields from environment variables.
// Environment variables use LLMKIT_ prefix and take precedence over existing values.
//
// Supported variables:
//   - LLMKIT_PROVIDER: Provider name
//   - LLMKIT_MODEL: Model name
//   - LLMKIT_FALLBACK_MODEL: Fallback model name
//   - LLMKIT_SYSTEM_PROMPT: System prompt
//   - LLMKIT_MAX_TURNS: Maximum turns
//   - LLMKIT_TIMEOUT: Timeout duration (e.g., "5m")
//   - LLMKIT_MAX_BUDGET_USD: Maximum budget
//   - LLMKIT_WORK_DIR: Working directory
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("LLMKIT_PROVIDER"); v != "" {
		c.Provider = v
	}
	if v := os.Getenv("LLMKIT_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("LLMKIT_FALLBACK_MODEL"); v != "" {
		c.FallbackModel = v
	}
	if v := os.Getenv("LLMKIT_SYSTEM_PROMPT"); v != "" {
		c.SystemPrompt = v
	}
	if v := os.Getenv("LLMKIT_MAX_TURNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxTurns = n
		}
	}
	if v := os.Getenv("LLMKIT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("LLMKIT_MAX_BUDGET_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MaxBudgetUSD = f
		}
	}
	if v := os.Getenv("LLMKIT_WORK_DIR"); v != "" {
		c.WorkDir = v
	}
}

// FromEnv creates a Config from environment variables with defaults.
func FromEnv() Config {
	cfg := DefaultConfig()
	cfg.LoadFromEnv()
	return cfg
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if c.MaxTurns < 0 {
		return fmt.Errorf("max_turns must be >= 0, got %d", c.MaxTurns)
	}
	if c.MaxBudgetUSD < 0 {
		return fmt.Errorf("max_budget_usd must be >= 0, got %f", c.MaxBudgetUSD)
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0, got %v", c.Timeout)
	}
	return nil
}

// WithProvider returns a copy of the config with the specified provider.
func (c Config) WithProvider(provider string) Config {
	c.Provider = provider
	return c
}

// WithModel returns a copy of the config with the specified model.
func (c Config) WithModel(model string) Config {
	c.Model = model
	return c
}

// WithWorkDir returns a copy of the config with the specified working directory.
func (c Config) WithWorkDir(dir string) Config {
	c.WorkDir = dir
	return c
}

// WithMCP returns a copy of the config with the specified MCP configuration.
func (c Config) WithMCP(mcp *claudeconfig.MCPConfig) Config {
	c.MCP = mcp
	return c
}

// WithOption returns a copy of the config with the specified option set.
func (c Config) WithOption(key string, value any) Config {
	if c.Options == nil {
		c.Options = make(map[string]any)
	} else {
		// Copy to avoid modifying original
		newOpts := make(map[string]any, len(c.Options)+1)
		for k, v := range c.Options {
			newOpts[k] = v
		}
		c.Options = newOpts
	}
	c.Options[key] = value
	return c
}

// GetOption retrieves a provider-specific option by key.
// Returns the zero value if the option is not set or has the wrong type.
func (c Config) GetOption(key string) any {
	if c.Options == nil {
		return nil
	}
	return c.Options[key]
}

// GetStringOption retrieves a string option, returning defaultVal if not set.
func (c Config) GetStringOption(key, defaultVal string) string {
	if c.Options == nil {
		return defaultVal
	}
	if v, ok := c.Options[key].(string); ok {
		return v
	}
	return defaultVal
}

// GetBoolOption retrieves a bool option, returning defaultVal if not set.
func (c Config) GetBoolOption(key string, defaultVal bool) bool {
	if c.Options == nil {
		return defaultVal
	}
	if v, ok := c.Options[key].(bool); ok {
		return v
	}
	return defaultVal
}

// GetIntOption retrieves an int option, returning defaultVal if not set.
func (c Config) GetIntOption(key string, defaultVal int) int {
	if c.Options == nil {
		return defaultVal
	}
	switch v := c.Options[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultVal
}

// GetStringSliceOption retrieves a string slice option, returning nil if not set.
// Handles both []string and []any (from JSON unmarshaling).
func (c Config) GetStringSliceOption(key string) []string {
	if c.Options == nil {
		return nil
	}
	switch v := c.Options[key].(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
