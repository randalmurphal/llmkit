package continuedev

import (
	"fmt"
	"time"
)

// Config holds Continue CLI configuration.
type Config struct {
	// Path is the path to the cn binary.
	// Default: "cn"
	Path string `json:"path" yaml:"path"`

	// ConfigPath is the path to config.yaml or a hub reference.
	// Example: "~/.continue/config.yaml" or "continuedev/default-cli-config"
	ConfigPath string `json:"config_path" yaml:"config_path"`

	// Model is the model name to use (must be configured in config.yaml).
	Model string `json:"model" yaml:"model"`

	// WorkDir is the working directory for the CLI.
	WorkDir string `json:"work_dir" yaml:"work_dir"`

	// Timeout is the request timeout.
	// Default: 5 minutes.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// AllowedTools are tool patterns to allow without prompting.
	// Example: []string{"Write()", "Edit()"}
	AllowedTools []string `json:"allowed_tools" yaml:"allowed_tools"`

	// AskTools are tool patterns that require approval.
	// Example: []string{"Bash(curl*)"}
	AskTools []string `json:"ask_tools" yaml:"ask_tools"`

	// ExcludedTools are tools to disable entirely.
	// Example: []string{"Fetch"}
	ExcludedTools []string `json:"excluded_tools" yaml:"excluded_tools"`

	// Env provides additional environment variables.
	Env map[string]string `json:"env" yaml:"env"`

	// APIKey is the Continue API key for CI/headless environments.
	// Can also be set via CONTINUE_API_KEY env var.
	APIKey string `json:"api_key" yaml:"api_key"`

	// Verbose enables detailed logging to ~/.continue/logs/cn.log.
	Verbose bool `json:"verbose" yaml:"verbose"`

	// Resume continues the previous conversation.
	Resume bool `json:"resume" yaml:"resume"`

	// Rule applies a specific rule from Mission Control.
	// Example: "nate/spanish"
	Rule string `json:"rule" yaml:"rule"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Path:    "cn",
		Timeout: 5 * time.Minute,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0")
	}
	return nil
}

// WithDefaults returns a copy of the config with defaults applied.
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.Path == "" {
		c.Path = defaults.Path
	}
	if c.Timeout == 0 {
		c.Timeout = defaults.Timeout
	}
	return c
}

// Option configures a ContinueCLI.
type Option func(*ContinueCLI)

// WithPath sets the path to the cn binary.
func WithPath(path string) Option {
	return func(c *ContinueCLI) { c.path = path }
}

// WithConfigPath sets the path to config.yaml.
func WithConfigPath(configPath string) Option {
	return func(c *ContinueCLI) { c.configPath = configPath }
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(c *ContinueCLI) { c.model = model }
}

// WithWorkdir sets the working directory.
func WithWorkdir(dir string) Option {
	return func(c *ContinueCLI) { c.workdir = dir }
}

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *ContinueCLI) { c.timeout = d }
}

// WithAllowedTools sets tools to allow without prompting.
func WithAllowedTools(tools []string) Option {
	return func(c *ContinueCLI) { c.allowedTools = tools }
}

// WithAskTools sets tools that require approval.
func WithAskTools(tools []string) Option {
	return func(c *ContinueCLI) { c.askTools = tools }
}

// WithExcludedTools sets tools to disable.
func WithExcludedTools(tools []string) Option {
	return func(c *ContinueCLI) { c.excludedTools = tools }
}

// WithEnv adds environment variables.
func WithEnv(env map[string]string) Option {
	return func(c *ContinueCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		for k, v := range env {
			c.extraEnv[k] = v
		}
	}
}

// WithEnvVar adds a single environment variable.
func WithEnvVar(key, value string) Option {
	return func(c *ContinueCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		c.extraEnv[key] = value
	}
}

// WithAPIKey sets the Continue API key.
func WithAPIKey(key string) Option {
	return func(c *ContinueCLI) { c.apiKey = key }
}

// WithVerbose enables verbose logging.
func WithVerbose() Option {
	return func(c *ContinueCLI) { c.verbose = true }
}

// WithResume enables session resumption.
func WithResume() Option {
	return func(c *ContinueCLI) { c.resume = true }
}

// WithRule sets a rule to apply.
func WithRule(rule string) Option {
	return func(c *ContinueCLI) { c.rule = rule }
}
