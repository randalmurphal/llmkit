package claude

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for a Claude client.
// Zero values use sensible defaults where noted.
type Config struct {
	// --- Model Selection ---

	// Model is the primary model to use.
	// Default: "claude-sonnet-4-20250514"
	Model string `json:"model" yaml:"model" mapstructure:"model"`

	// FallbackModel is used when primary model is unavailable.
	// Optional.
	FallbackModel string `json:"fallback_model" yaml:"fallback_model" mapstructure:"fallback_model"`

	// --- Prompts ---

	// SystemPrompt is the system message prepended to all requests.
	// Optional.
	SystemPrompt string `json:"system_prompt" yaml:"system_prompt" mapstructure:"system_prompt"`

	// AppendSystemPrompt is appended to the existing system prompt.
	// Use when you want to add to defaults rather than replace.
	AppendSystemPrompt string `json:"append_system_prompt" yaml:"append_system_prompt" mapstructure:"append_system_prompt"`

	// --- Execution Limits ---

	// MaxTurns limits conversation turns (tool calls + responses).
	// 0 means no limit. Default: 10.
	MaxTurns int `json:"max_turns" yaml:"max_turns" mapstructure:"max_turns"`

	// Timeout is the maximum duration for a completion request.
	// 0 uses the default (5 minutes).
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`

	// MaxBudgetUSD limits spending per request.
	// 0 means no limit.
	MaxBudgetUSD float64 `json:"max_budget_usd" yaml:"max_budget_usd" mapstructure:"max_budget_usd"`

	// --- Working Directory ---

	// WorkDir is the working directory for file operations.
	// Default: current directory.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`

	// --- Tool Control ---

	// AllowedTools limits which tools Claude can use.
	// Empty means all tools allowed.
	AllowedTools []string `json:"allowed_tools" yaml:"allowed_tools" mapstructure:"allowed_tools"`

	// DisallowedTools explicitly blocks certain tools.
	// Takes precedence over AllowedTools.
	DisallowedTools []string `json:"disallowed_tools" yaml:"disallowed_tools" mapstructure:"disallowed_tools"`

	// Tools specifies the exact list of available tools.
	// Different from AllowedTools which is a whitelist filter.
	Tools []string `json:"tools" yaml:"tools" mapstructure:"tools"`

	// DangerouslySkipPermissions bypasses permission prompts.
	// Use with extreme caution, only in trusted environments.
	DangerouslySkipPermissions bool `json:"dangerously_skip_permissions" yaml:"dangerously_skip_permissions" mapstructure:"dangerously_skip_permissions"`

	// PermissionMode sets the permission handling mode.
	// Valid values: "", "acceptEdits", "bypassPermissions"
	PermissionMode PermissionMode `json:"permission_mode" yaml:"permission_mode" mapstructure:"permission_mode"`

	// --- Session Management ---

	// SessionID enables session persistence with this ID.
	// Optional.
	SessionID string `json:"session_id" yaml:"session_id" mapstructure:"session_id"`

	// Continue resumes the last session.
	Continue bool `json:"continue" yaml:"continue" mapstructure:"continue"`

	// Resume resumes a specific session by ID.
	Resume string `json:"resume" yaml:"resume" mapstructure:"resume"`

	// NoSessionPersistence disables session saving.
	NoSessionPersistence bool `json:"no_session_persistence" yaml:"no_session_persistence" mapstructure:"no_session_persistence"`

	// --- Container Environment ---

	// HomeDir overrides the home directory (for containers).
	HomeDir string `json:"home_dir" yaml:"home_dir" mapstructure:"home_dir"`

	// ConfigDir overrides the .claude config directory.
	ConfigDir string `json:"config_dir" yaml:"config_dir" mapstructure:"config_dir"`

	// Env provides additional environment variables.
	Env map[string]string `json:"env" yaml:"env" mapstructure:"env"`

	// --- Output Control ---

	// OutputFormat controls CLI output format.
	// Default: "json".
	OutputFormat OutputFormat `json:"output_format" yaml:"output_format" mapstructure:"output_format"`

	// JSONSchema forces structured output matching the given schema.
	JSONSchema string `json:"json_schema" yaml:"json_schema" mapstructure:"json_schema"`

	// --- Context ---

	// AddDirs adds directories to Claude's file access scope.
	AddDirs []string `json:"add_dirs" yaml:"add_dirs" mapstructure:"add_dirs"`

	// SettingSources specifies which setting sources to use.
	// Valid values: "project", "local", "user"
	SettingSources []string `json:"setting_sources" yaml:"setting_sources" mapstructure:"setting_sources"`

	// --- MCP Configuration ---

	// MCPConfigPath is the path to an MCP configuration JSON file.
	// Maps to the --mcp-config CLI flag.
	MCPConfigPath string `json:"mcp_config_path" yaml:"mcp_config_path" mapstructure:"mcp_config_path"`

	// MCPServers defines MCP servers inline (alternative to config file).
	// Keys are server names, values are server configurations.
	MCPServers map[string]MCPServerConfig `json:"mcp_servers" yaml:"mcp_servers" mapstructure:"mcp_servers"`

	// StrictMCPConfig ignores all MCP configurations except those specified.
	// Maps to the --strict-mcp-config CLI flag.
	StrictMCPConfig bool `json:"strict_mcp_config" yaml:"strict_mcp_config" mapstructure:"strict_mcp_config"`

	// --- Advanced ---

	// ClaudePath is the path to the claude CLI binary.
	// Default: "claude" (found via PATH).
	ClaudePath string `json:"claude_path" yaml:"claude_path" mapstructure:"claude_path"`
}

// MCPServerConfig defines an MCP server for the Claude CLI.
// Supports stdio, http, and sse transport types.
type MCPServerConfig struct {
	// Type specifies the transport type: "stdio", "http", or "sse".
	// If empty, defaults to "stdio" for servers with Command set.
	Type string `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type"`

	// Command is the command to run the MCP server (for stdio transport).
	Command string `json:"command,omitempty" yaml:"command,omitempty" mapstructure:"command"`

	// Args are the arguments to pass to the command (for stdio transport).
	Args []string `json:"args,omitempty" yaml:"args,omitempty" mapstructure:"args"`

	// Env provides environment variables for the server process.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty" mapstructure:"env"`

	// URL is the server endpoint (for http/sse transport).
	URL string `json:"url,omitempty" yaml:"url,omitempty" mapstructure:"url"`

	// Headers are HTTP headers (for http/sse transport).
	Headers []string `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Model:        "claude-sonnet-4-20250514",
		MaxTurns:     10,
		Timeout:      5 * time.Minute,
		OutputFormat: OutputFormatJSON,
	}
}

// LoadFromEnv populates config fields from environment variables.
// Environment variables use CLAUDE_ prefix and take precedence over existing values.
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("CLAUDE_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("CLAUDE_FALLBACK_MODEL"); v != "" {
		c.FallbackModel = v
	}
	if v := os.Getenv("CLAUDE_SYSTEM_PROMPT"); v != "" {
		c.SystemPrompt = v
	}
	if v := os.Getenv("CLAUDE_APPEND_SYSTEM_PROMPT"); v != "" {
		c.AppendSystemPrompt = v
	}
	if v := os.Getenv("CLAUDE_MAX_TURNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxTurns = n
		}
	}
	if v := os.Getenv("CLAUDE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("CLAUDE_MAX_BUDGET_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MaxBudgetUSD = f
		}
	}
	if v := os.Getenv("CLAUDE_WORK_DIR"); v != "" {
		c.WorkDir = v
	}
	if v := os.Getenv("CLAUDE_PATH"); v != "" {
		c.ClaudePath = v
	}
	if v := os.Getenv("CLAUDE_HOME_DIR"); v != "" {
		c.HomeDir = v
	}
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		c.ConfigDir = v
	}
	if v := os.Getenv("CLAUDE_OUTPUT_FORMAT"); v != "" {
		c.OutputFormat = OutputFormat(v)
	}
	if v := os.Getenv("CLAUDE_SKIP_PERMISSIONS"); v == "true" || v == "1" {
		c.DangerouslySkipPermissions = true
	}
	if v := os.Getenv("CLAUDE_PERMISSION_MODE"); v != "" {
		c.PermissionMode = PermissionMode(v)
	}
	if v := os.Getenv("CLAUDE_SESSION_ID"); v != "" {
		c.SessionID = v
	}
	if v := os.Getenv("CLAUDE_NO_SESSION_PERSISTENCE"); v == "true" || v == "1" {
		c.NoSessionPersistence = true
	}
	if v := os.Getenv("CLAUDE_MCP_CONFIG"); v != "" {
		c.MCPConfigPath = v
	}
	if v := os.Getenv("CLAUDE_STRICT_MCP_CONFIG"); v == "true" || v == "1" {
		c.StrictMCPConfig = true
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
	// Model is the only truly required field
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	// MaxTurns must be non-negative
	if c.MaxTurns < 0 {
		return fmt.Errorf("max_turns must be >= 0, got %d", c.MaxTurns)
	}
	// MaxBudgetUSD must be non-negative
	if c.MaxBudgetUSD < 0 {
		return fmt.Errorf("max_budget_usd must be >= 0, got %f", c.MaxBudgetUSD)
	}
	// Timeout must be non-negative
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0, got %v", c.Timeout)
	}
	return nil
}

// ToOptions converts the config to functional options.
// This enables mixing Config with additional options.
func (c *Config) ToOptions() []ClaudeOption {
	opts := make([]ClaudeOption, 0, 24)

	if c.Model != "" {
		opts = append(opts, WithModel(c.Model))
	}
	if c.FallbackModel != "" {
		opts = append(opts, WithFallbackModel(c.FallbackModel))
	}
	if c.SystemPrompt != "" {
		opts = append(opts, WithSystemPrompt(c.SystemPrompt))
	}
	if c.AppendSystemPrompt != "" {
		opts = append(opts, WithAppendSystemPrompt(c.AppendSystemPrompt))
	}
	if c.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(c.MaxTurns))
	}
	if c.Timeout > 0 {
		opts = append(opts, WithTimeout(c.Timeout))
	}
	if c.MaxBudgetUSD > 0 {
		opts = append(opts, WithMaxBudgetUSD(c.MaxBudgetUSD))
	}
	if c.WorkDir != "" {
		opts = append(opts, WithWorkdir(c.WorkDir))
	}
	if len(c.AllowedTools) > 0 {
		opts = append(opts, WithAllowedTools(c.AllowedTools))
	}
	if len(c.DisallowedTools) > 0 {
		opts = append(opts, WithDisallowedTools(c.DisallowedTools))
	}
	if len(c.Tools) > 0 {
		opts = append(opts, WithTools(c.Tools))
	}
	if c.DangerouslySkipPermissions {
		opts = append(opts, WithDangerouslySkipPermissions())
	}
	if c.PermissionMode != "" {
		opts = append(opts, WithPermissionMode(c.PermissionMode))
	}
	if c.SessionID != "" {
		opts = append(opts, WithSessionID(c.SessionID))
	}
	if c.Continue {
		opts = append(opts, WithContinue())
	}
	if c.Resume != "" {
		opts = append(opts, WithResume(c.Resume))
	}
	if c.NoSessionPersistence {
		opts = append(opts, WithNoSessionPersistence())
	}
	if c.HomeDir != "" {
		opts = append(opts, WithHomeDir(c.HomeDir))
	}
	if c.ConfigDir != "" {
		opts = append(opts, WithConfigDir(c.ConfigDir))
	}
	if len(c.Env) > 0 {
		opts = append(opts, WithEnv(c.Env))
	}
	if c.OutputFormat != "" {
		opts = append(opts, WithOutputFormat(c.OutputFormat))
	}
	if c.JSONSchema != "" {
		opts = append(opts, WithJSONSchema(c.JSONSchema))
	}
	if len(c.AddDirs) > 0 {
		opts = append(opts, WithAddDirs(c.AddDirs))
	}
	if len(c.SettingSources) > 0 {
		opts = append(opts, WithSettingSources(c.SettingSources))
	}
	if c.MCPConfigPath != "" {
		opts = append(opts, WithMCPConfig(c.MCPConfigPath))
	}
	if len(c.MCPServers) > 0 {
		opts = append(opts, WithMCPServers(c.MCPServers))
	}
	if c.StrictMCPConfig {
		opts = append(opts, WithStrictMCPConfig())
	}
	if c.ClaudePath != "" {
		opts = append(opts, WithClaudePath(c.ClaudePath))
	}

	return opts
}
