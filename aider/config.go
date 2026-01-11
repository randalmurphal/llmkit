package aider

import (
	"fmt"
	"time"
)

// Config holds Aider CLI configuration.
type Config struct {
	// Path is the path to the aider binary.
	// Default: "aider"
	Path string `json:"path" yaml:"path"`

	// Model is the model to use.
	// For Ollama, use "ollama_chat/<model>" prefix.
	// Example: "ollama_chat/llama3.2:latest"
	Model string `json:"model" yaml:"model"`

	// WorkDir is the working directory for the CLI.
	WorkDir string `json:"work_dir" yaml:"work_dir"`

	// Timeout is the API request timeout.
	// Default: 5 minutes.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// EditableFiles are files that Aider can modify.
	// These are passed with --file flags.
	EditableFiles []string `json:"editable_files" yaml:"editable_files"`

	// ReadOnlyFiles are files for context only (not editable).
	// These are passed with --read flags.
	ReadOnlyFiles []string `json:"read_only_files" yaml:"read_only_files"`

	// NoGit disables git integration entirely.
	NoGit bool `json:"no_git" yaml:"no_git"`

	// NoAutoCommits disables automatic commits after edits.
	NoAutoCommits bool `json:"no_auto_commits" yaml:"no_auto_commits"`

	// NoStream disables streaming responses.
	NoStream bool `json:"no_stream" yaml:"no_stream"`

	// DryRun previews changes without modifying files.
	DryRun bool `json:"dry_run" yaml:"dry_run"`

	// YesAlways automatically confirms all prompts.
	// Required for non-interactive automation.
	YesAlways bool `json:"yes_always" yaml:"yes_always"`

	// EditFormat specifies the edit format (diff, whole, etc.).
	EditFormat string `json:"edit_format" yaml:"edit_format"`

	// Env provides additional environment variables.
	Env map[string]string `json:"env" yaml:"env"`

	// OllamaAPIBase is the Ollama API base URL.
	// Can also be set via OLLAMA_API_BASE env var.
	// Default: http://127.0.0.1:11434
	OllamaAPIBase string `json:"ollama_api_base" yaml:"ollama_api_base"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Path:    "aider",
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

// Option configures an AiderCLI.
type Option func(*AiderCLI)

// WithPath sets the path to the aider binary.
func WithPath(path string) Option {
	return func(c *AiderCLI) { c.path = path }
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(c *AiderCLI) { c.model = model }
}

// WithWorkdir sets the working directory.
func WithWorkdir(dir string) Option {
	return func(c *AiderCLI) { c.workdir = dir }
}

// WithTimeout sets the API timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *AiderCLI) { c.timeout = d }
}

// WithEditableFiles sets files that can be modified.
func WithEditableFiles(files []string) Option {
	return func(c *AiderCLI) { c.editableFiles = files }
}

// WithReadOnlyFiles sets read-only context files.
func WithReadOnlyFiles(files []string) Option {
	return func(c *AiderCLI) { c.readOnlyFiles = files }
}

// WithNoGit disables git integration.
func WithNoGit() Option {
	return func(c *AiderCLI) { c.noGit = true }
}

// WithNoAutoCommits disables automatic commits.
func WithNoAutoCommits() Option {
	return func(c *AiderCLI) { c.noAutoCommits = true }
}

// WithNoStream disables streaming responses.
func WithNoStream() Option {
	return func(c *AiderCLI) { c.noStream = true }
}

// WithDryRun enables dry-run mode (preview without modifying).
func WithDryRun() Option {
	return func(c *AiderCLI) { c.dryRun = true }
}

// WithYesAlways enables automatic confirmation of all prompts.
func WithYesAlways() Option {
	return func(c *AiderCLI) { c.yesAlways = true }
}

// WithEditFormat sets the edit format.
func WithEditFormat(format string) Option {
	return func(c *AiderCLI) { c.editFormat = format }
}

// WithEnv adds environment variables.
func WithEnv(env map[string]string) Option {
	return func(c *AiderCLI) {
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
	return func(c *AiderCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		c.extraEnv[key] = value
	}
}

// WithOllamaAPIBase sets the Ollama API base URL.
func WithOllamaAPIBase(url string) Option {
	return func(c *AiderCLI) { c.ollamaAPIBase = url }
}
