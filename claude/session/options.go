package session

import "time"

// SessionOption configures a Session.
type SessionOption func(*sessionConfig)

// sessionConfig holds session configuration.
type sessionConfig struct {
	// CLI path
	claudePath string

	// Model configuration
	model         string
	fallbackModel string

	// Working directory
	workdir string

	// Session behavior
	sessionID            string
	resume               bool
	noSessionPersistence bool

	// Tool control
	allowedTools    []string
	disallowedTools []string
	tools           []string

	// Permissions
	dangerouslySkipPermissions bool
	permissionMode             string
	settingSources             []string

	// Context
	addDirs            []string
	systemPrompt       string
	appendSystemPrompt string

	// Budget and limits
	maxBudgetUSD float64
	maxTurns     int

	// Timeouts
	startupTimeout time.Duration
	idleTimeout    time.Duration

	// Environment
	homeDir   string
	configDir string
	extraEnv  map[string]string

	// Output filtering
	includeHookOutput bool
}

// defaultConfig returns the default session configuration.
func defaultConfig() sessionConfig {
	return sessionConfig{
		claudePath:                 "claude",
		dangerouslySkipPermissions: true, // Required for non-interactive use
		startupTimeout:             30 * time.Second,
		idleTimeout:                10 * time.Minute,
		includeHookOutput:          false,
	}
}

// WithClaudePath sets the path to the claude binary.
func WithClaudePath(path string) SessionOption {
	return func(c *sessionConfig) { c.claudePath = path }
}

// WithModel sets the model to use.
func WithModel(model string) SessionOption {
	return func(c *sessionConfig) { c.model = model }
}

// WithFallbackModel sets a fallback model if primary is overloaded.
func WithFallbackModel(model string) SessionOption {
	return func(c *sessionConfig) { c.fallbackModel = model }
}

// WithWorkdir sets the working directory for the session.
func WithWorkdir(dir string) SessionOption {
	return func(c *sessionConfig) { c.workdir = dir }
}

// WithSessionID sets a specific session ID.
// If not set, Claude CLI generates one automatically.
func WithSessionID(id string) SessionOption {
	return func(c *sessionConfig) { c.sessionID = id }
}

// WithResume enables resuming the specified session ID.
// The session must have been previously persisted.
func WithResume(sessionID string) SessionOption {
	return func(c *sessionConfig) {
		c.sessionID = sessionID
		c.resume = true
	}
}

// WithNoSessionPersistence disables saving session data.
func WithNoSessionPersistence() SessionOption {
	return func(c *sessionConfig) { c.noSessionPersistence = true }
}

// WithAllowedTools sets the allowed tools (whitelist).
func WithAllowedTools(tools []string) SessionOption {
	return func(c *sessionConfig) { c.allowedTools = tools }
}

// WithDisallowedTools sets the disallowed tools (blacklist).
func WithDisallowedTools(tools []string) SessionOption {
	return func(c *sessionConfig) { c.disallowedTools = tools }
}

// WithTools sets the exact list of available tools.
func WithTools(tools []string) SessionOption {
	return func(c *sessionConfig) { c.tools = tools }
}

// WithPermissions configures permission handling.
// If skip is true, all permission prompts are bypassed (default for sessions).
func WithPermissions(skip bool) SessionOption {
	return func(c *sessionConfig) { c.dangerouslySkipPermissions = skip }
}

// WithPermissionMode sets the permission mode.
// Valid values: "", "acceptEdits", "bypassPermissions"
func WithPermissionMode(mode string) SessionOption {
	return func(c *sessionConfig) { c.permissionMode = mode }
}

// WithSettingSources specifies which setting sources to use.
// Valid values: "project", "local", "user"
func WithSettingSources(sources []string) SessionOption {
	return func(c *sessionConfig) { c.settingSources = sources }
}

// WithAddDirs adds directories to Claude's file access scope.
func WithAddDirs(dirs []string) SessionOption {
	return func(c *sessionConfig) { c.addDirs = dirs }
}

// WithSystemPrompt sets a custom system prompt.
func WithSystemPrompt(prompt string) SessionOption {
	return func(c *sessionConfig) { c.systemPrompt = prompt }
}

// WithAppendSystemPrompt appends to the system prompt.
func WithAppendSystemPrompt(prompt string) SessionOption {
	return func(c *sessionConfig) { c.appendSystemPrompt = prompt }
}

// WithMaxBudgetUSD sets a maximum spending limit.
func WithMaxBudgetUSD(amount float64) SessionOption {
	return func(c *sessionConfig) { c.maxBudgetUSD = amount }
}

// WithMaxTurns limits the number of agentic turns.
func WithMaxTurns(n int) SessionOption {
	return func(c *sessionConfig) { c.maxTurns = n }
}

// WithStartupTimeout sets how long to wait for session initialization.
func WithStartupTimeout(d time.Duration) SessionOption {
	return func(c *sessionConfig) { c.startupTimeout = d }
}

// WithIdleTimeout sets how long a session can be idle before auto-closing.
func WithIdleTimeout(d time.Duration) SessionOption {
	return func(c *sessionConfig) { c.idleTimeout = d }
}

// WithHomeDir sets the HOME environment variable for credential discovery.
func WithHomeDir(dir string) SessionOption {
	return func(c *sessionConfig) { c.homeDir = dir }
}

// WithConfigDir sets the Claude config directory path.
func WithConfigDir(dir string) SessionOption {
	return func(c *sessionConfig) { c.configDir = dir }
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

// WithIncludeHookOutput includes hook responses in the output channel.
// By default, hook output is filtered out.
func WithIncludeHookOutput(include bool) SessionOption {
	return func(c *sessionConfig) { c.includeHookOutput = include }
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
