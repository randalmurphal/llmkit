package gemini

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for a Gemini client.
// Zero values use sensible defaults where noted.
type Config struct {
	// --- Model Selection ---

	// Model is the primary model to use.
	// Default: "gemini-2.5-pro"
	Model string `json:"model" yaml:"model" mapstructure:"model"`

	// --- Prompts ---

	// SystemPrompt is the system message prepended to all requests.
	// Optional.
	SystemPrompt string `json:"system_prompt" yaml:"system_prompt" mapstructure:"system_prompt"`

	// --- Execution Limits ---

	// MaxTurns limits conversation turns (tool calls + responses).
	// 0 means no limit.
	MaxTurns int `json:"max_turns" yaml:"max_turns" mapstructure:"max_turns"`

	// Timeout is the maximum duration for a completion request.
	// 0 uses the default (5 minutes).
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`

	// --- Working Directory ---

	// WorkDir is the working directory for file operations.
	// Default: current directory.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`

	// --- Tool Control ---

	// AllowedTools limits which tools Gemini can use (comma-separated in CLI).
	// Tools in this list bypass the confirmation dialog.
	// Empty means normal approval flow applies.
	AllowedTools []string `json:"allowed_tools" yaml:"allowed_tools" mapstructure:"allowed_tools"`

	// Note: Gemini CLI does not support a --disallowed-tools flag.
	// Use approval modes or /policies command for tool restrictions.

	// Yolo enables auto-approval of all actions (no prompts).
	// Use with extreme caution, only in trusted environments.
	Yolo bool `json:"yolo" yaml:"yolo" mapstructure:"yolo"`

	// --- Environment ---

	// Env provides additional environment variables.
	Env map[string]string `json:"env" yaml:"env" mapstructure:"env"`

	// --- Output Control ---

	// OutputFormat controls CLI output format.
	// Default: "json".
	OutputFormat OutputFormat `json:"output_format" yaml:"output_format" mapstructure:"output_format"`

	// --- Context ---

	// IncludeDirs adds directories to Gemini's file access scope.
	IncludeDirs []string `json:"include_dirs" yaml:"include_dirs" mapstructure:"include_dirs"`

	// --- MCP Configuration ---

	// MCPConfigPath is the path to an MCP configuration JSON file.
	// Maps to the --mcp CLI flag.
	MCPConfigPath string `json:"mcp_config_path" yaml:"mcp_config_path" mapstructure:"mcp_config_path"`

	// MCPServers defines MCP servers inline (alternative to config file).
	// Keys are server names, values are server configurations.
	MCPServers map[string]MCPServerConfig `json:"mcp_servers" yaml:"mcp_servers" mapstructure:"mcp_servers"`

	// --- Sandbox ---

	// Sandbox specifies the execution sandbox mode.
	// Valid values: "host" (default), "docker", "remote-execution"
	Sandbox string `json:"sandbox" yaml:"sandbox" mapstructure:"sandbox"`

	// --- Advanced ---

	// GeminiPath is the path to the gemini CLI binary.
	// Default: "gemini" (found via PATH).
	GeminiPath string `json:"gemini_path" yaml:"gemini_path" mapstructure:"gemini_path"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Model:        "gemini-2.5-pro",
		Timeout:      5 * time.Minute,
		OutputFormat: OutputFormatJSON,
	}
}

// LoadFromEnv populates config fields from environment variables.
// Environment variables use GEMINI_ prefix and take precedence over existing values.
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("GEMINI_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("GEMINI_SYSTEM_PROMPT"); v != "" {
		c.SystemPrompt = v
	}
	if v := os.Getenv("GEMINI_MAX_TURNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxTurns = n
		}
	}
	if v := os.Getenv("GEMINI_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("GEMINI_WORK_DIR"); v != "" {
		c.WorkDir = v
	}
	if v := os.Getenv("GEMINI_PATH"); v != "" {
		c.GeminiPath = v
	}
	if v := os.Getenv("GEMINI_OUTPUT_FORMAT"); v != "" {
		c.OutputFormat = OutputFormat(v)
	}
	if v := os.Getenv("GEMINI_YOLO"); v == "true" || v == "1" {
		c.Yolo = true
	}
	if v := os.Getenv("GEMINI_SANDBOX"); v != "" {
		c.Sandbox = v
	}
	if v := os.Getenv("GEMINI_MCP_CONFIG"); v != "" {
		c.MCPConfigPath = v
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
	// Timeout must be non-negative
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0, got %v", c.Timeout)
	}
	// Validate sandbox mode if specified
	if c.Sandbox != "" && c.Sandbox != "host" && c.Sandbox != "docker" && c.Sandbox != "remote-execution" {
		return fmt.Errorf("invalid sandbox mode: %s (valid: host, docker, remote-execution)", c.Sandbox)
	}
	return nil
}

// ToOptions converts the config to functional options.
// This enables mixing Config with additional options.
func (c *Config) ToOptions() []GeminiOption {
	opts := make([]GeminiOption, 0, 16)

	if c.Model != "" {
		opts = append(opts, WithModel(c.Model))
	}
	if c.SystemPrompt != "" {
		opts = append(opts, WithSystemPrompt(c.SystemPrompt))
	}
	if c.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(c.MaxTurns))
	}
	if c.Timeout > 0 {
		opts = append(opts, WithTimeout(c.Timeout))
	}
	if c.WorkDir != "" {
		opts = append(opts, WithWorkdir(c.WorkDir))
	}
	if len(c.AllowedTools) > 0 {
		opts = append(opts, WithAllowedTools(c.AllowedTools))
	}
	// Note: Gemini CLI does not support --disallowed-tools flag
	// Tool restrictions should be managed via approval modes or policies
	if c.Yolo {
		opts = append(opts, WithYolo())
	}
	if len(c.Env) > 0 {
		opts = append(opts, WithEnv(c.Env))
	}
	if c.OutputFormat != "" {
		opts = append(opts, WithOutputFormat(c.OutputFormat))
	}
	if len(c.IncludeDirs) > 0 {
		opts = append(opts, WithIncludeDirs(c.IncludeDirs))
	}
	if c.MCPConfigPath != "" {
		opts = append(opts, WithMCPConfig(c.MCPConfigPath))
	}
	if len(c.MCPServers) > 0 {
		opts = append(opts, WithMCPServers(c.MCPServers))
	}
	if c.Sandbox != "" {
		opts = append(opts, WithSandbox(c.Sandbox))
	}
	if c.GeminiPath != "" {
		opts = append(opts, WithGeminiPath(c.GeminiPath))
	}

	return opts
}
