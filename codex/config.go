package codex

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	SandboxMode SandboxMode `json:"sandbox_mode" yaml:"sandbox_mode" mapstructure:"sandbox_mode"`

	// --- Approval Mode ---

	// ApprovalMode controls when to ask for user approval.
	// Options: "untrusted", "on-failure", "on-request", "never"
	ApprovalMode ApprovalMode `json:"approval_mode" yaml:"approval_mode" mapstructure:"approval_mode"`

	// FullAuto enables automatic approvals for all operations.
	// Equivalent to --full-auto flag.
	FullAuto bool `json:"full_auto" yaml:"full_auto" mapstructure:"full_auto"`

	// DangerouslyBypassApprovalsAndSandbox bypasses both approvals and sandboxing.
	// Equivalent to --dangerously-bypass-approvals-and-sandbox / --yolo.
	DangerouslyBypassApprovalsAndSandbox bool `json:"dangerously_bypass_approvals_and_sandbox" yaml:"dangerously_bypass_approvals_and_sandbox" mapstructure:"dangerously_bypass_approvals_and_sandbox"`

	// --- Session Management ---

	// SessionID is the session ID for resuming sessions.
	SessionID string `json:"session_id" yaml:"session_id" mapstructure:"session_id"`

	// ResumeAll includes all previous turns when resuming `last`.
	ResumeAll bool `json:"resume_all" yaml:"resume_all" mapstructure:"resume_all"`

	// --- Features ---

	// EnableSearch enables web search capabilities.
	// Deprecated alias for WebSearchMode=live.
	EnableSearch bool `json:"enable_search" yaml:"enable_search" mapstructure:"enable_search"`

	// WebSearchMode controls web search strategy.
	// Options: "cached", "live", "disabled"
	WebSearchMode WebSearchMode `json:"web_search_mode" yaml:"web_search_mode" mapstructure:"web_search_mode"`

	// ReasoningEffort controls model reasoning effort for models that support it.
	// Common values: "minimal", "low", "medium", "high".
	ReasoningEffort string `json:"reasoning_effort" yaml:"reasoning_effort" mapstructure:"reasoning_effort"`

	// HideAgentReasoning hides internal agent reasoning in output.
	// Applied via -c hide_agent_reasoning=true.
	HideAgentReasoning bool `json:"hide_agent_reasoning" yaml:"hide_agent_reasoning" mapstructure:"hide_agent_reasoning"`

	// UseOSS enables local OSS backend mode.
	UseOSS bool `json:"use_oss" yaml:"use_oss" mapstructure:"use_oss"`

	// EnabledFeatures enables experimental codex features (--enable).
	EnabledFeatures []string `json:"enabled_features" yaml:"enabled_features" mapstructure:"enabled_features"`

	// DisabledFeatures disables codex features (--disable).
	DisabledFeatures []string `json:"disabled_features" yaml:"disabled_features" mapstructure:"disabled_features"`

	// ColorMode sets output color mode: "auto", "always", "never".
	ColorMode string `json:"color_mode" yaml:"color_mode" mapstructure:"color_mode"`

	// --- Profiles and Config Overrides ---

	// Profile selects a codex profile from config.toml.
	Profile string `json:"profile" yaml:"profile" mapstructure:"profile"`

	// LocalProvider selects OSS local provider backend (e.g., "lmstudio", "ollama").
	LocalProvider string `json:"local_provider" yaml:"local_provider" mapstructure:"local_provider"`

	// ConfigOverrides are applied as repeated -c key=value flags.
	ConfigOverrides map[string]any `json:"config_overrides" yaml:"config_overrides" mapstructure:"config_overrides"`

	// --- Workspace and Git ---

	// SkipGitRepoCheck allows running outside a git repository.
	SkipGitRepoCheck bool `json:"skip_git_repo_check" yaml:"skip_git_repo_check" mapstructure:"skip_git_repo_check"`

	// --- Output Controls ---

	// OutputSchemaPath sets --output-schema to enforce final output JSON schema.
	OutputSchemaPath string `json:"output_schema_path" yaml:"output_schema_path" mapstructure:"output_schema_path"`

	// OutputLastMessagePath sets --output-last-message file output path.
	OutputLastMessagePath string `json:"output_last_message_path" yaml:"output_last_message_path" mapstructure:"output_last_message_path"`

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
	if v := os.Getenv("CODEX_YOLO"); v == "true" || v == "1" {
		c.DangerouslyBypassApprovalsAndSandbox = true
	}
	if v := os.Getenv("CODEX_SESSION_ID"); v != "" {
		c.SessionID = v
	}
	if v := os.Getenv("CODEX_RESUME_ALL"); v == "true" || v == "1" {
		c.ResumeAll = true
	}
	if v := os.Getenv("CODEX_SEARCH"); v == "true" || v == "1" {
		c.EnableSearch = true
	}
	if v := os.Getenv("CODEX_WEB_SEARCH"); v != "" {
		c.WebSearchMode = WebSearchMode(v)
	}
	if v := os.Getenv("CODEX_PROFILE"); v != "" {
		c.Profile = v
	}
	if v := os.Getenv("CODEX_LOCAL_PROVIDER"); v != "" {
		c.LocalProvider = v
	}
	if v := os.Getenv("CODEX_SKIP_GIT_REPO_CHECK"); v == "true" || v == "1" {
		c.SkipGitRepoCheck = true
	}
	if v := os.Getenv("CODEX_OUTPUT_SCHEMA"); v != "" {
		c.OutputSchemaPath = v
	}
	if v := os.Getenv("CODEX_OUTPUT_LAST_MESSAGE"); v != "" {
		c.OutputLastMessagePath = v
	}
	if v := os.Getenv("CODEX_REASONING_EFFORT"); v != "" {
		c.ReasoningEffort = v
	}
	if v := os.Getenv("CODEX_HIDE_AGENT_REASONING"); v == "true" || v == "1" {
		c.HideAgentReasoning = true
	}
	if v := os.Getenv("CODEX_OSS"); v == "true" || v == "1" {
		c.UseOSS = true
	}
	if v := os.Getenv("CODEX_ENABLE_FEATURES"); v != "" {
		c.EnabledFeatures = splitCommaList(v)
	}
	if v := os.Getenv("CODEX_DISABLE_FEATURES"); v != "" {
		c.DisabledFeatures = splitCommaList(v)
	}
	if v := os.Getenv("CODEX_COLOR"); v != "" {
		c.ColorMode = v
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

	if c.SandboxMode != "" {
		switch c.SandboxMode {
		case SandboxReadOnly, SandboxWorkspaceWrite, SandboxDangerFullAccess:
		default:
			return fmt.Errorf("invalid sandbox_mode: %q (must be read-only, workspace-write, or danger-full-access)", c.SandboxMode)
		}
	}

	if c.ApprovalMode != "" {
		switch c.ApprovalMode {
		case ApprovalUntrusted, ApprovalOnFailure, ApprovalOnRequest, ApprovalNever:
		default:
			return fmt.Errorf("invalid approval_mode: %q (must be untrusted, on-failure, on-request, or never)", c.ApprovalMode)
		}
	}

	if c.WebSearchMode != "" {
		switch c.WebSearchMode {
		case WebSearchCached, WebSearchLive, WebSearchDisabled:
		default:
			return fmt.Errorf("invalid web_search_mode: %q (must be cached, live, or disabled)", c.WebSearchMode)
		}
	}

	if c.ReasoningEffort != "" {
		re := strings.ToLower(c.ReasoningEffort)
		switch re {
		case "minimal", "low", "medium", "high", "xhigh":
		default:
			return fmt.Errorf("invalid reasoning_effort: %q (must be minimal, low, medium, high, or xhigh)", c.ReasoningEffort)
		}
	}
	if c.ColorMode != "" {
		switch strings.ToLower(c.ColorMode) {
		case "auto", "always", "never":
		default:
			return fmt.Errorf("invalid color_mode: %q (must be auto, always, or never)", c.ColorMode)
		}
	}

	return nil
}

// ToOptions converts the config to functional options.
func (c *Config) ToOptions() []CodexOption {
	opts := make([]CodexOption, 0, 24)

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
	if c.DangerouslyBypassApprovalsAndSandbox {
		opts = append(opts, WithDangerouslyBypassApprovalsAndSandbox())
	}
	if c.SessionID != "" {
		opts = append(opts, WithSessionID(c.SessionID))
	}
	if c.ResumeAll {
		opts = append(opts, WithResumeAll())
	}
	if c.WebSearchMode != "" {
		opts = append(opts, WithWebSearchMode(c.WebSearchMode))
	} else if c.EnableSearch {
		opts = append(opts, WithSearch())
	}
	if c.Profile != "" {
		opts = append(opts, WithProfile(c.Profile))
	}
	if c.LocalProvider != "" {
		opts = append(opts, WithLocalProvider(c.LocalProvider))
	}
	if len(c.ConfigOverrides) > 0 {
		opts = append(opts, WithConfigOverrides(c.ConfigOverrides))
	}
	if c.SkipGitRepoCheck {
		opts = append(opts, WithSkipGitRepoCheck())
	}
	if c.OutputSchemaPath != "" {
		opts = append(opts, WithOutputSchema(c.OutputSchemaPath))
	}
	if c.OutputLastMessagePath != "" {
		opts = append(opts, WithOutputLastMessage(c.OutputLastMessagePath))
	}
	if c.ReasoningEffort != "" {
		opts = append(opts, WithReasoningEffort(c.ReasoningEffort))
	}
	if c.HideAgentReasoning {
		opts = append(opts, WithHideAgentReasoning())
	}
	if c.UseOSS {
		opts = append(opts, WithOSS())
	}
	if len(c.EnabledFeatures) > 0 {
		opts = append(opts, WithEnabledFeatures(c.EnabledFeatures))
	}
	if len(c.DisabledFeatures) > 0 {
		opts = append(opts, WithDisabledFeatures(c.DisabledFeatures))
	}
	if c.ColorMode != "" {
		opts = append(opts, WithColorMode(c.ColorMode))
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

func splitCommaList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
