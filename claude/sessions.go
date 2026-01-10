package claude

import (
	"github.com/randalmurphal/llmkit/claude/session"
)

// Session-related type aliases for convenience.
// These allow using session types without importing the session subpackage.
type (
	// Session manages a long-running Claude CLI process with stream-json I/O.
	Session = session.Session

	// SessionManager manages multiple Claude CLI sessions.
	SessionManager = session.SessionManager

	// SessionOption configures session creation.
	SessionOption = session.SessionOption

	// SessionInfo contains metadata about a session.
	SessionInfo = session.SessionInfo

	// SessionStatus represents the current state of a session.
	SessionStatus = session.SessionStatus

	// OutputMessage represents any message received from Claude CLI stdout.
	OutputMessage = session.OutputMessage

	// InitMessage contains session initialization data.
	InitMessage = session.InitMessage

	// AssistantMessage contains Claude's response.
	AssistantMessage = session.AssistantMessage

	// ResultMessage contains the final result of a session turn.
	ResultMessage = session.ResultMessage

	// UserMessage is a message to send to Claude via stream-json input.
	UserMessage = session.UserMessage

	// ManagerOption configures session manager creation.
	ManagerOption = session.ManagerOption
)

// Session status constants.
const (
	SessionStatusCreating    = session.StatusCreating
	SessionStatusActive      = session.StatusActive
	SessionStatusClosing     = session.StatusClosing
	SessionStatusClosed      = session.StatusClosed
	SessionStatusError       = session.StatusError
	SessionStatusTerminating = session.StatusTerminating
)

// NewSessionManager creates a new session manager.
// Use this to manage multiple concurrent sessions with automatic cleanup.
//
// Example:
//
//	mgr := claude.NewSessionManager(
//	    claude.WithMaxSessions(10),
//	    claude.WithSessionTTL(30*time.Minute),
//	)
//	defer mgr.CloseAll()
var NewSessionManager = session.NewManager

// NewUserMessage creates a new user message for sending to Claude.
var NewUserMessage = session.NewUserMessage

// Session options re-exported for convenience.
// Note: These are prefixed with "Session" to avoid conflicts with ClaudeCLI options.
var (
	// SessionWithID sets a specific session ID.
	SessionWithID = session.WithSessionID

	// SessionWithResume enables resuming a persisted session.
	SessionWithResume = session.WithResume

	// SessionWithModel sets the model for the session.
	SessionWithModel = session.WithModel

	// SessionWithWorkdir sets the working directory for the session.
	SessionWithWorkdir = session.WithWorkdir

	// SessionWithSystemPrompt sets a custom system prompt.
	SessionWithSystemPrompt = session.WithSystemPrompt

	// SessionWithAppendSystemPrompt appends to the system prompt.
	SessionWithAppendSystemPrompt = session.WithAppendSystemPrompt

	// SessionWithAllowedTools sets the allowed tools whitelist.
	SessionWithAllowedTools = session.WithAllowedTools

	// SessionWithDisallowedTools sets the disallowed tools blacklist.
	SessionWithDisallowedTools = session.WithDisallowedTools

	// SessionWithPermissions configures permission handling.
	// If skip is true, all permission prompts are bypassed.
	SessionWithPermissions = session.WithPermissions

	// SessionWithMaxBudgetUSD sets a maximum spending limit.
	SessionWithMaxBudgetUSD = session.WithMaxBudgetUSD

	// SessionWithMaxTurns limits the number of turns.
	SessionWithMaxTurns = session.WithMaxTurns

	// SessionWithNoPersistence disables session persistence.
	SessionWithNoPersistence = session.WithNoSessionPersistence

	// SessionWithIncludeHookOutput includes hook output in the output channel.
	SessionWithIncludeHookOutput = session.WithIncludeHookOutput
)

// Manager options re-exported for convenience.
var (
	// WithMaxSessions sets the maximum number of concurrent sessions.
	WithMaxSessions = session.WithMaxSessions

	// WithSessionTTL sets the time-to-live for idle sessions.
	WithSessionTTL = session.WithSessionTTL

	// WithCleanupInterval sets how often to check for expired sessions.
	WithCleanupInterval = session.WithCleanupInterval

	// WithDefaultSessionOptions sets default options for all sessions.
	WithDefaultSessionOptions = session.WithDefaultSessionOptions
)
