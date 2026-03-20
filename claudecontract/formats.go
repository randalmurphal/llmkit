package claudecontract

import "slices"

// Output formats for CLI output.
const (
	// FormatText is plain text output (default).
	FormatText = "text"

	// FormatJSON is structured JSON output.
	FormatJSON = "json"

	// FormatStreamJSON is newline-delimited JSON for streaming.
	FormatStreamJSON = "stream-json"
)

// Transport types for MCP servers.
const (
	// TransportStdio is a stdio-based MCP server.
	TransportStdio = "stdio"

	// TransportHTTP is an HTTP-based MCP server.
	TransportHTTP = "http"

	// TransportSSE is a Server-Sent Events MCP server.
	TransportSSE = "sse"

	// TransportSDK is an in-process SDK MCP server.
	TransportSDK = "sdk"
)

// HookEvent represents a lifecycle event that can trigger hooks.
// From official SDK documentation.
type HookEvent string

const (
	// HookSessionStart fires when a session begins or resumes.
	HookSessionStart HookEvent = "SessionStart"

	// HookUserPromptSubmit fires when the user submits a prompt.
	HookUserPromptSubmit HookEvent = "UserPromptSubmit"

	// HookPreToolUse fires before a tool is executed.
	HookPreToolUse HookEvent = "PreToolUse"

	// HookPermissionRequest fires when a permission dialog appears.
	HookPermissionRequest HookEvent = "PermissionRequest"

	// HookPostToolUse fires after a tool succeeds.
	HookPostToolUse HookEvent = "PostToolUse"

	// HookPostToolUseFailure fires after a tool fails.
	HookPostToolUseFailure HookEvent = "PostToolUseFailure"

	// HookSubagentStart fires when a subagent is spawned.
	HookSubagentStart HookEvent = "SubagentStart"

	// HookSubagentStop fires when a subagent finishes.
	HookSubagentStop HookEvent = "SubagentStop"

	// HookStop fires when Claude finishes responding.
	HookStop HookEvent = "Stop"

	// HookStopFailure fires when a turn ends due to an API error.
	HookStopFailure HookEvent = "StopFailure"

	// HookPreCompact fires before context compaction.
	HookPreCompact HookEvent = "PreCompact"

	// HookPostCompact fires after context compaction completes.
	HookPostCompact HookEvent = "PostCompact"

	// HookSessionEnd fires when a session terminates.
	HookSessionEnd HookEvent = "SessionEnd"

	// HookNotification fires when Claude Code sends notifications.
	HookNotification HookEvent = "Notification"

	// HookTeammateIdle fires when an agent team teammate is about to go idle.
	HookTeammateIdle HookEvent = "TeammateIdle"

	// HookTaskCompleted fires when a task is marked as completed.
	HookTaskCompleted HookEvent = "TaskCompleted"

	// HookInstructionsLoaded fires when CLAUDE.md or .claude/rules/*.md files are loaded.
	HookInstructionsLoaded HookEvent = "InstructionsLoaded"

	// HookConfigChange fires when a configuration file changes during a session.
	HookConfigChange HookEvent = "ConfigChange"

	// HookWorktreeCreate fires when a worktree is being created.
	HookWorktreeCreate HookEvent = "WorktreeCreate"

	// HookWorktreeRemove fires when a worktree is being removed.
	HookWorktreeRemove HookEvent = "WorktreeRemove"

	// HookElicitation fires when an MCP server requests user input.
	HookElicitation HookEvent = "Elicitation"

	// HookElicitationResult fires when the user responds to an MCP elicitation.
	HookElicitationResult HookEvent = "ElicitationResult"
)

// ValidHookEvents returns all valid hook events.
func ValidHookEvents() []HookEvent {
	return []HookEvent{
		HookSessionStart,
		HookUserPromptSubmit,
		HookPreToolUse,
		HookPermissionRequest,
		HookPostToolUse,
		HookPostToolUseFailure,
		HookSubagentStart,
		HookSubagentStop,
		HookStop,
		HookStopFailure,
		HookPreCompact,
		HookPostCompact,
		HookSessionEnd,
		HookNotification,
		HookTeammateIdle,
		HookTaskCompleted,
		HookInstructionsLoaded,
		HookConfigChange,
		HookWorktreeCreate,
		HookWorktreeRemove,
		HookElicitation,
		HookElicitationResult,
	}
}

// IsValid returns true if the hook event is valid.
func (h HookEvent) IsValid() bool {
	return slices.Contains(ValidHookEvents(), h)
}

// String returns the string value of the hook event.
func (h HookEvent) String() string {
	return string(h)
}

// CompactTrigger represents what triggered a compaction.
type CompactTrigger string

const (
	// CompactTriggerManual indicates manual compaction.
	CompactTriggerManual CompactTrigger = "manual"

	// CompactTriggerAuto indicates automatic compaction.
	CompactTriggerAuto CompactTrigger = "auto"
)

// SessionStartSource indicates how a session started.
type SessionStartSource string

const (
	// SessionStartSourceStartup indicates a new session started.
	SessionStartSourceStartup SessionStartSource = "startup"

	// SessionStartSourceResume indicates a resumed session.
	SessionStartSourceResume SessionStartSource = "resume"

	// SessionStartSourceClear indicates a cleared session.
	SessionStartSourceClear SessionStartSource = "clear"

	// SessionStartSourceCompact indicates a compacted session.
	SessionStartSourceCompact SessionStartSource = "compact"
)
