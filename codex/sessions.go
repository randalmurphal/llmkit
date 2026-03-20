package codex

import (
	"github.com/randalmurphal/llmkit/codex/session"
)

// Session-related type aliases for convenience.
// These allow using session types without importing the session subpackage.
type (
	// CodexSession manages a long-running Codex app-server process with JSON-RPC I/O.
	CodexSession = session.Session

	// CodexSessionManager manages multiple Codex sessions.
	CodexSessionManager = session.SessionManager

	// CodexSessionOption configures session creation.
	CodexSessionOption = session.SessionOption

	// CodexSessionInfo contains metadata about a session.
	CodexSessionInfo = session.SessionInfo

	// CodexSessionStatus represents the current state of a session.
	CodexSessionStatus = session.SessionStatus

	// CodexOutputMessage represents a notification from the app-server.
	CodexOutputMessage = session.OutputMessage

	// CodexUserMessage is a message to send to Codex via turn/start.
	CodexUserMessage = session.UserMessage

	// CodexManagerOption configures session manager creation.
	CodexManagerOption = session.ManagerOption
)

// Session status constants.
const (
	CodexSessionStatusCreating = session.StatusCreating
	CodexSessionStatusActive   = session.StatusActive
	CodexSessionStatusClosing  = session.StatusClosing
	CodexSessionStatusClosed   = session.StatusClosed
	CodexSessionStatusError    = session.StatusError
)

// NewCodexSessionManager creates a new Codex session manager.
var NewCodexSessionManager = session.NewManager

// NewCodexUserMessage creates a new user message for sending to Codex.
var NewCodexUserMessage = session.NewUserMessage

// Session options re-exported for convenience.
var (
	CodexSessionWithModel       = session.WithModel
	CodexSessionWithWorkdir     = session.WithWorkdir
	CodexSessionWithThreadID    = session.WithThreadID
	CodexSessionWithResume      = session.WithResume
	CodexSessionWithFullAuto    = session.WithFullAuto
	CodexSessionWithSandboxMode = session.WithSandboxMode
	CodexSessionWithEnv         = session.WithEnv
)

// Manager options re-exported for convenience.
var (
	CodexWithMaxSessions          = session.WithMaxSessions
	CodexWithSessionTTL           = session.WithSessionTTL
	CodexWithCleanupInterval      = session.WithCleanupInterval
	CodexWithDefaultSessionOptions = session.WithDefaultSessionOptions
)
