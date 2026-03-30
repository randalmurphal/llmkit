package llmkit

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds shared configuration for creating a provider client.
// Provider-specific behavior is configured through direct provider constructors.
type Config struct {
	Provider        string            `json:"provider" yaml:"provider" mapstructure:"provider"`
	Model           string            `json:"model" yaml:"model" mapstructure:"model"`
	FallbackModel   string            `json:"fallback_model" yaml:"fallback_model" mapstructure:"fallback_model"`
	SystemPrompt    string            `json:"system_prompt" yaml:"system_prompt" mapstructure:"system_prompt"`
	AppendSystemPrompt string         `json:"append_system_prompt" yaml:"append_system_prompt" mapstructure:"append_system_prompt"`
	MaxTurns        int               `json:"max_turns" yaml:"max_turns" mapstructure:"max_turns"`
	Timeout         time.Duration     `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	MaxBudgetUSD    float64           `json:"max_budget_usd" yaml:"max_budget_usd" mapstructure:"max_budget_usd"`
	WorkDir         string            `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`
	AllowedTools    []string          `json:"allowed_tools" yaml:"allowed_tools" mapstructure:"allowed_tools"`
	DisallowedTools []string          `json:"disallowed_tools" yaml:"disallowed_tools" mapstructure:"disallowed_tools"`
	Tools           []string          `json:"tools" yaml:"tools" mapstructure:"tools"`
	MCPServers      map[string]MCPServerConfig `json:"mcp_servers" yaml:"mcp_servers" mapstructure:"mcp_servers"`
	StrictMCPConfig bool              `json:"strict_mcp_config" yaml:"strict_mcp_config" mapstructure:"strict_mcp_config"`
	Env             map[string]string `json:"env" yaml:"env" mapstructure:"env"`
	AddDirs         []string          `json:"add_dirs" yaml:"add_dirs" mapstructure:"add_dirs"`
	Session         *SessionMetadata  `json:"session,omitempty" yaml:"session,omitempty" mapstructure:"session"`
	ReasoningEffort string            `json:"reasoning_effort" yaml:"reasoning_effort" mapstructure:"reasoning_effort"`
	WebSearchMode   string            `json:"web_search_mode" yaml:"web_search_mode" mapstructure:"web_search_mode"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxTurns: 10,
		Timeout:  5 * time.Minute,
	}
}

// LoadFromEnv populates config fields from environment variables.
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

func (c Config) WithProvider(provider string) Config {
	c.Provider = provider
	return c
}

func (c Config) WithModel(model string) Config {
	c.Model = model
	return c
}

func (c Config) WithWorkDir(dir string) Config {
	c.WorkDir = dir
	return c
}
