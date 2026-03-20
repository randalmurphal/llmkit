package session

import "time"

// SessionOption configures a Session.
type SessionOption func(*sessionConfig)

// sessionConfig holds session configuration.
type sessionConfig struct {
	// CLI path
	codexPath string

	// Model configuration
	model string

	// Working directory
	workdir string

	// Sandbox and approval modes
	sandboxMode  string // "read-only", "workspace-write", "full-access"
	approvalMode string // "untrusted", "on-failure", "on-request", "never"
	fullAuto     bool

	// Thread management
	threadID string
	resume   bool

	// System prompt
	systemPrompt string

	// Feature flags
	enabledFeatures  []string
	disabledFeatures []string

	// Timeouts
	startupTimeout time.Duration
	idleTimeout    time.Duration

	// Environment
	extraEnv map[string]string
}

// defaultConfig returns the default session configuration.
func defaultConfig() sessionConfig {
	return sessionConfig{
		codexPath:      "codex",
		startupTimeout: 30 * time.Second,
		idleTimeout:    10 * time.Minute,
	}
}

// WithCodexPath sets the path to the codex binary.
func WithCodexPath(path string) SessionOption {
	return func(c *sessionConfig) { c.codexPath = path }
}

// WithModel sets the model to use.
func WithModel(model string) SessionOption {
	return func(c *sessionConfig) { c.model = model }
}

// WithWorkdir sets the working directory for the session.
func WithWorkdir(dir string) SessionOption {
	return func(c *sessionConfig) { c.workdir = dir }
}

// WithSandboxMode sets the sandbox mode for file system access.
// Valid values: "read-only", "workspace-write", "full-access".
func WithSandboxMode(mode string) SessionOption {
	return func(c *sessionConfig) { c.sandboxMode = mode }
}

// WithApprovalMode sets the approval mode for tool execution.
// Valid values: "untrusted", "on-failure", "on-request", "never".
func WithApprovalMode(mode string) SessionOption {
	return func(c *sessionConfig) { c.approvalMode = mode }
}

// WithFullAuto enables full-auto mode, which sets approval to "never"
// and sandbox to "full-access".
func WithFullAuto() SessionOption {
	return func(c *sessionConfig) { c.fullAuto = true }
}

// WithThreadID sets a specific thread ID for resuming.
func WithThreadID(id string) SessionOption {
	return func(c *sessionConfig) { c.threadID = id }
}

// WithResume enables resuming the specified thread ID.
// The thread must have been previously created.
func WithResume(threadID string) SessionOption {
	return func(c *sessionConfig) {
		c.threadID = threadID
		c.resume = true
	}
}

// WithSystemPrompt sets a custom system prompt.
func WithSystemPrompt(prompt string) SessionOption {
	return func(c *sessionConfig) { c.systemPrompt = prompt }
}

// WithEnabledFeatures sets feature flags to enable (e.g., "codex_hooks").
func WithEnabledFeatures(features []string) SessionOption {
	return func(c *sessionConfig) { c.enabledFeatures = features }
}

// WithDisabledFeatures sets feature flags to disable.
func WithDisabledFeatures(features []string) SessionOption {
	return func(c *sessionConfig) { c.disabledFeatures = features }
}

// WithStartupTimeout sets the timeout for session startup operations,
// including the thread/start or thread/resume handshake.
func WithStartupTimeout(d time.Duration) SessionOption {
	return func(c *sessionConfig) { c.startupTimeout = d }
}

// WithIdleTimeout sets how long a session can be idle before auto-closing.
func WithIdleTimeout(d time.Duration) SessionOption {
	return func(c *sessionConfig) { c.idleTimeout = d }
}

// WithEnv adds environment variables to the CLI process.
func WithEnv(env map[string]string) SessionOption {
	return func(c *sessionConfig) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		for k, v := range env {
			c.extraEnv[k] = v
		}
	}
}

// ManagerOption configures a SessionManager.
type ManagerOption func(*managerConfig)

// managerConfig holds manager configuration.
type managerConfig struct {
	// Maximum concurrent sessions
	maxSessions int

	// Default session options applied to all sessions
	defaultOpts []SessionOption

	// TTL for idle sessions (0 = no auto-cleanup)
	sessionTTL time.Duration

	// Cleanup interval for expired sessions
	cleanupInterval time.Duration
}

// defaultManagerConfig returns the default manager configuration.
func defaultManagerConfig() managerConfig {
	return managerConfig{
		maxSessions:     100,
		sessionTTL:      30 * time.Minute,
		cleanupInterval: 5 * time.Minute,
	}
}

// WithMaxSessions sets the maximum number of concurrent sessions.
func WithMaxSessions(n int) ManagerOption {
	return func(c *managerConfig) { c.maxSessions = n }
}

// WithDefaultSessionOptions sets options applied to all new sessions.
func WithDefaultSessionOptions(opts ...SessionOption) ManagerOption {
	return func(c *managerConfig) { c.defaultOpts = opts }
}

// WithSessionTTL sets the TTL for idle sessions.
// Sessions idle longer than this are automatically closed.
func WithSessionTTL(d time.Duration) ManagerOption {
	return func(c *managerConfig) { c.sessionTTL = d }
}

// WithCleanupInterval sets how often to check for expired sessions.
func WithCleanupInterval(d time.Duration) ManagerOption {
	return func(c *managerConfig) { c.cleanupInterval = d }
}
