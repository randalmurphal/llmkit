// Package opencode provides a client for the OpenCode CLI.
// OpenCode is an AI coding assistant that supports 75+ LLM providers
// with built-in agents for development (build) and analysis (plan).
package opencode

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for an OpenCode client.
// Zero values use sensible defaults where noted.
type Config struct {
	// --- Execution Settings ---

	// OpenCodePath is the path to the opencode CLI binary.
	// Default: "opencode" (found via PATH).
	OpenCodePath string `json:"opencode_path" yaml:"opencode_path" mapstructure:"opencode_path"`

	// WorkDir is the working directory for file operations.
	// Default: current directory.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`

	// Timeout is the maximum duration for a completion request.
	// 0 uses the default (5 minutes).
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`

	// --- Agent Selection ---

	// Agent specifies which OpenCode agent to use.
	// Valid values: "build" (default, full access), "plan" (read-only).
	Agent Agent `json:"agent" yaml:"agent" mapstructure:"agent"`

	// --- Output Control ---

	// OutputFormat controls CLI output format.
	// Default: "json".
	OutputFormat OutputFormat `json:"output_format" yaml:"output_format" mapstructure:"output_format"`

	// Quiet enables quiet mode for cleaner output.
	// Default: true.
	Quiet bool `json:"quiet" yaml:"quiet" mapstructure:"quiet"`

	// Debug enables debug mode for verbose output.
	// Default: false.
	Debug bool `json:"debug" yaml:"debug" mapstructure:"debug"`

	// --- Prompts ---

	// SystemPrompt is the system message prepended to all requests.
	// Optional.
	SystemPrompt string `json:"system_prompt" yaml:"system_prompt" mapstructure:"system_prompt"`

	// --- Execution Limits ---

	// MaxTurns limits conversation turns (tool calls + responses).
	// 0 means no limit.
	MaxTurns int `json:"max_turns" yaml:"max_turns" mapstructure:"max_turns"`

	// --- Tool Control ---

	// AllowedTools limits which tools OpenCode can use.
	// Empty means all tools allowed.
	AllowedTools []string `json:"allowed_tools" yaml:"allowed_tools" mapstructure:"allowed_tools"`

	// DisallowedTools explicitly blocks certain tools.
	// Takes precedence over AllowedTools.
	DisallowedTools []string `json:"disallowed_tools" yaml:"disallowed_tools" mapstructure:"disallowed_tools"`

	// --- Environment ---

	// Env provides additional environment variables.
	Env map[string]string `json:"env" yaml:"env" mapstructure:"env"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		OpenCodePath: "opencode",
		Timeout:      5 * time.Minute,
		Agent:        AgentBuild,
		OutputFormat: OutputFormatJSON,
		Quiet:        true,
	}
}

// LoadFromEnv populates config fields from environment variables.
// Environment variables use OPENCODE_ prefix and take precedence over existing values.
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("OPENCODE_PATH"); v != "" {
		c.OpenCodePath = v
	}
	if v := os.Getenv("OPENCODE_WORK_DIR"); v != "" {
		c.WorkDir = v
	}
	if v := os.Getenv("OPENCODE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("OPENCODE_AGENT"); v != "" {
		c.Agent = Agent(v)
	}
	if v := os.Getenv("OPENCODE_OUTPUT_FORMAT"); v != "" {
		c.OutputFormat = OutputFormat(v)
	}
	if v := os.Getenv("OPENCODE_QUIET"); v == "true" || v == "1" {
		c.Quiet = true
	} else if v == "false" || v == "0" {
		c.Quiet = false
	}
	if v := os.Getenv("OPENCODE_DEBUG"); v == "true" || v == "1" {
		c.Debug = true
	}
	if v := os.Getenv("OPENCODE_SYSTEM_PROMPT"); v != "" {
		c.SystemPrompt = v
	}
	if v := os.Getenv("OPENCODE_MAX_TURNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxTurns = n
		}
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
	// Timeout must be non-negative
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0, got %v", c.Timeout)
	}
	// MaxTurns must be non-negative
	if c.MaxTurns < 0 {
		return fmt.Errorf("max_turns must be >= 0, got %d", c.MaxTurns)
	}
	// Agent must be valid if set
	if c.Agent != "" && c.Agent != AgentBuild && c.Agent != AgentPlan {
		return fmt.Errorf("invalid agent %q, must be 'build' or 'plan'", c.Agent)
	}
	// OutputFormat must be valid if set
	if c.OutputFormat != "" && c.OutputFormat != OutputFormatText && c.OutputFormat != OutputFormatJSON {
		return fmt.Errorf("invalid output_format %q, must be 'text' or 'json'", c.OutputFormat)
	}
	return nil
}

// ToOptions converts the config to functional options.
// This enables mixing Config with additional options.
func (c *Config) ToOptions() []OpenCodeOption {
	opts := make([]OpenCodeOption, 0, 12)

	if c.OpenCodePath != "" {
		opts = append(opts, WithOpenCodePath(c.OpenCodePath))
	}
	if c.WorkDir != "" {
		opts = append(opts, WithWorkdir(c.WorkDir))
	}
	if c.Timeout > 0 {
		opts = append(opts, WithTimeout(c.Timeout))
	}
	if c.Agent != "" {
		opts = append(opts, WithAgent(c.Agent))
	}
	if c.OutputFormat != "" {
		opts = append(opts, WithOutputFormat(c.OutputFormat))
	}
	// Always apply Quiet since it defaults to true
	opts = append(opts, WithQuiet(c.Quiet))
	if c.Debug {
		opts = append(opts, WithDebug(c.Debug))
	}
	if c.SystemPrompt != "" {
		opts = append(opts, WithSystemPrompt(c.SystemPrompt))
	}
	if c.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(c.MaxTurns))
	}
	if len(c.AllowedTools) > 0 {
		opts = append(opts, WithAllowedTools(c.AllowedTools))
	}
	if len(c.DisallowedTools) > 0 {
		opts = append(opts, WithDisallowedTools(c.DisallowedTools))
	}
	if len(c.Env) > 0 {
		opts = append(opts, WithEnv(c.Env))
	}

	return opts
}

// NewFromConfig creates an OpenCodeCLI from a Config struct.
// Additional options can be passed to override config values.
func NewFromConfig(cfg Config, opts ...OpenCodeOption) *OpenCodeCLI {
	allOpts := cfg.ToOptions()
	allOpts = append(allOpts, opts...)
	return NewOpenCodeCLI(allOpts...)
}

// NewFromEnv creates an OpenCodeCLI from environment variables.
// Additional options can be passed to override config values.
func NewFromEnv(opts ...OpenCodeOption) *OpenCodeCLI {
	cfg := FromEnv()
	return NewFromConfig(cfg, opts...)
}
