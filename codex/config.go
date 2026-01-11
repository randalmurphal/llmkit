package codex

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for a Codex client.
// Zero values use sensible defaults where noted.
type Config struct {
	// --- Model Selection ---

	// Model is the primary model to use.
	// Default depends on OpenAI account configuration.
	Model string `json:"model" yaml:"model" mapstructure:"model"`

	// --- Execution Limits ---

	// Timeout is the maximum duration for a completion request.
	// 0 uses the default (5 minutes).
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`

	// --- Working Directory ---

	// WorkDir is the working directory for file operations.
	// Default: current directory.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`

	// --- Sandbox Mode ---

	// SandboxMode controls file system access.
	// Options: "read-only", "workspace-write", "danger-full-access"
	// Default: "workspace-write"
	SandboxMode SandboxMode `json:"sandbox_mode" yaml:"sandbox_mode" mapstructure:"sandbox_mode"`

	// --- Approval Mode ---

	// ApprovalMode controls when to ask for user approval.
	// Options: "untrusted", "on-failure", "on-request", "never"
	ApprovalMode ApprovalMode `json:"approval_mode" yaml:"approval_mode" mapstructure:"approval_mode"`

	// FullAuto enables automatic approvals for all operations.
	// Equivalent to --full-auto flag.
	FullAuto bool `json:"full_auto" yaml:"full_auto" mapstructure:"full_auto"`

	// --- Session Management ---

	// SessionID is the session ID for resuming sessions.
	SessionID string `json:"session_id" yaml:"session_id" mapstructure:"session_id"`

	// --- Features ---

	// EnableSearch enables web search capabilities.
	EnableSearch bool `json:"enable_search" yaml:"enable_search" mapstructure:"enable_search"`

	// --- Directories ---

	// AddDirs adds directories to Codex's file access scope.
	AddDirs []string `json:"add_dirs" yaml:"add_dirs" mapstructure:"add_dirs"`

	// --- Images ---

	// Images lists image paths to attach to requests.
	Images []string `json:"images" yaml:"images" mapstructure:"images"`

	// --- Environment ---

	// Env provides additional environment variables.
	Env map[string]string `json:"env" yaml:"env" mapstructure:"env"`

	// --- Advanced ---

	// CodexPath is the path to the codex CLI binary.
	// Default: "codex" (found via PATH).
	CodexPath string `json:"codex_path" yaml:"codex_path" mapstructure:"codex_path"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout:     5 * time.Minute,
		SandboxMode: SandboxWorkspaceWrite,
	}
}

// LoadFromEnv populates config fields from environment variables.
// Environment variables use CODEX_ prefix and take precedence over existing values.
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("CODEX_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("CODEX_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("CODEX_WORK_DIR"); v != "" {
		c.WorkDir = v
	}
	if v := os.Getenv("CODEX_SANDBOX_MODE"); v != "" {
		c.SandboxMode = SandboxMode(v)
	}
	if v := os.Getenv("CODEX_APPROVAL_MODE"); v != "" {
		c.ApprovalMode = ApprovalMode(v)
	}
	if v := os.Getenv("CODEX_FULL_AUTO"); v == "true" || v == "1" {
		c.FullAuto = true
	}
	if v := os.Getenv("CODEX_SESSION_ID"); v != "" {
		c.SessionID = v
	}
	if v := os.Getenv("CODEX_SEARCH"); v == "true" || v == "1" {
		c.EnableSearch = true
	}
	if v := os.Getenv("CODEX_PATH"); v != "" {
		c.CodexPath = v
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
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0, got %v", c.Timeout)
	}

	// Validate sandbox mode if set
	if c.SandboxMode != "" {
		switch c.SandboxMode {
		case SandboxReadOnly, SandboxWorkspaceWrite, SandboxDangerFullAccess:
			// Valid
		default:
			return fmt.Errorf("invalid sandbox_mode: %q (must be read-only, workspace-write, or danger-full-access)", c.SandboxMode)
		}
	}

	// Validate approval mode if set
	if c.ApprovalMode != "" {
		switch c.ApprovalMode {
		case ApprovalUntrusted, ApprovalOnFailure, ApprovalOnRequest, ApprovalNever:
			// Valid
		default:
			return fmt.Errorf("invalid approval_mode: %q (must be untrusted, on-failure, on-request, or never)", c.ApprovalMode)
		}
	}

	return nil
}

// ToOptions converts the config to functional options.
func (c *Config) ToOptions() []CodexOption {
	opts := make([]CodexOption, 0, 12)

	if c.Model != "" {
		opts = append(opts, WithModel(c.Model))
	}
	if c.Timeout > 0 {
		opts = append(opts, WithTimeout(c.Timeout))
	}
	if c.WorkDir != "" {
		opts = append(opts, WithWorkdir(c.WorkDir))
	}
	if c.SandboxMode != "" {
		opts = append(opts, WithSandboxMode(c.SandboxMode))
	}
	if c.ApprovalMode != "" {
		opts = append(opts, WithApprovalMode(c.ApprovalMode))
	}
	if c.FullAuto {
		opts = append(opts, WithFullAuto())
	}
	if c.SessionID != "" {
		opts = append(opts, WithSessionID(c.SessionID))
	}
	if c.EnableSearch {
		opts = append(opts, WithSearch())
	}
	if len(c.AddDirs) > 0 {
		opts = append(opts, WithAddDirs(c.AddDirs))
	}
	if len(c.Images) > 0 {
		opts = append(opts, WithImages(c.Images))
	}
	if len(c.Env) > 0 {
		opts = append(opts, WithEnv(c.Env))
	}
	if c.CodexPath != "" {
		opts = append(opts, WithCodexPath(c.CodexPath))
	}

	return opts
}

// GetStringOption retrieves a string from environment with fallback.
func GetStringOption(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// GetBoolOption retrieves a bool from environment with fallback.
func GetBoolOption(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
