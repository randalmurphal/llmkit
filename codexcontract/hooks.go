package codexcontract

import (
	"encoding/json"
	"slices"
)

// HookEvent represents a Codex CLI lifecycle event that can trigger hooks.
// Codex hooks are configured in hooks.json and require the codex_hooks feature flag.
type HookEvent string

const (
	// HookSessionStart fires when a session begins, resumes, or clears.
	HookSessionStart HookEvent = "SessionStart"

	// HookPreToolUse fires before a tool is executed.
	HookPreToolUse HookEvent = "PreToolUse"

	// HookPostToolUse fires after a tool returns.
	HookPostToolUse HookEvent = "PostToolUse"

	// HookUserPromptSubmit fires when the user submits a prompt.
	HookUserPromptSubmit HookEvent = "UserPromptSubmit"

	// HookStop fires when the agent finishes a turn or is about to stop.
	HookStop HookEvent = "Stop"
)

// ValidHookEvents returns all valid Codex hook events.
func ValidHookEvents() []HookEvent {
	return []HookEvent{
		HookSessionStart,
		HookPreToolUse,
		HookPostToolUse,
		HookUserPromptSubmit,
		HookStop,
	}
}

// IsValid returns true if the hook event is a recognized Codex event.
func (h HookEvent) IsValid() bool {
	return slices.Contains(ValidHookEvents(), h)
}

// String returns the string value of the hook event.
func (h HookEvent) String() string {
	return string(h)
}

// HookDecision represents the decision a hook returns to control agent behavior.
type HookDecision string

const (
	// HookDecisionBlock prevents the operation (stop or prompt submission).
	HookDecisionBlock HookDecision = "block"
)

// SessionStartSource indicates how a session started.
type SessionStartSource string

const (
	SessionStartSourceStartup SessionStartSource = "startup"
	SessionStartSourceResume  SessionStartSource = "resume"
	SessionStartSourceClear   SessionStartSource = "clear"
)

// HookConfig represents the top-level hooks.json configuration file.
type HookConfig struct {
	Hooks map[string][]HookMatcher `json:"hooks"`
}

// HookMatcher represents a matcher group within an event's hook array.
type HookMatcher struct {
	Matcher string      `json:"matcher,omitempty"` // Regex pattern (only meaningful for SessionStart)
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry represents a single hook action in hooks.json.
type HookEntry struct {
	Type          string `json:"type"`                    // "command" (only type currently implemented)
	Command       string `json:"command,omitempty"`       // Shell command to execute
	Timeout       int    `json:"timeout,omitempty"`       // Timeout in seconds (default 600, min 1)
	StatusMessage string `json:"statusMessage,omitempty"` // Status displayed while hook runs
}

// SessionStartInput is the JSON passed to SessionStart hooks via stdin.
type SessionStartInput struct {
	HookContext
	Source SessionStartSource `json:"source"` // "startup", "resume", or "clear"
}

// UserPromptSubmitInput is the JSON passed to UserPromptSubmit hooks via stdin.
type UserPromptSubmitInput struct {
	HookContext
	Prompt string `json:"prompt"`
}

// HookContext is the shared hook stdin envelope for Codex hooks.
type HookContext struct {
	SessionID      string `json:"session_id,omitempty"`
	TurnID         string `json:"turn_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	CWD            string `json:"cwd,omitempty"`
	HookEventName  string `json:"hook_event_name,omitempty"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
}

// ToolHookInput is the JSON passed to PreToolUse and PostToolUse hooks via stdin.
type ToolHookInput struct {
	HookContext
	ToolName   string          `json:"tool_name,omitempty"`
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`
	ToolOutput json.RawMessage `json:"tool_output,omitempty"`
}

// StopInput is the JSON passed to Stop hooks via stdin.
type StopInput struct {
	HookContext
	StopHookActive       bool   `json:"stop_hook_active"`
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
}

// HookOutput is the JSON a hook command writes to stdout.
// Note: Continue is a pointer so that the zero value (nil) is distinguishable
// from an explicit false. A nil Continue is treated as "continue" (safe default),
// whereas *Continue == false means "abort session".
type HookOutput struct {
	Continue       *bool           `json:"continue,omitempty"`           // nil/true = continue, false = abort session
	StopReason     string          `json:"stopReason,omitempty"`         // Shown when continue=false
	SuppressOutput bool            `json:"suppressOutput,omitempty"`     // Reserved
	SystemMessage  string          `json:"systemMessage,omitempty"`      // Displayed as warning
	Decision       HookDecision    `json:"decision,omitempty"`           // "block" for Stop/UserPromptSubmit
	Reason         string          `json:"reason,omitempty"`             // Required when decision=block
	Specific       json.RawMessage `json:"hookSpecificOutput,omitempty"` // Event-specific output
}

// ContinueOutput returns a HookOutput that signals the session should continue.
func ContinueOutput() HookOutput {
	t := true
	return HookOutput{Continue: &t}
}

// AbortOutput returns a HookOutput that signals the session should abort
// with the given reason.
func AbortOutput(reason string) HookOutput {
	f := false
	return HookOutput{Continue: &f, StopReason: reason}
}

// ShouldContinue returns true if the hook output indicates the session
// should continue. A nil Continue pointer is treated as true (safe default).
func (h *HookOutput) ShouldContinue() bool {
	return h.Continue == nil || *h.Continue
}

// SessionStartSpecific is the hookSpecificOutput for SessionStart hooks.
type SessionStartSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"` // Injected as model context
}

// UserPromptSubmitSpecific is the hookSpecificOutput for UserPromptSubmit hooks.
type UserPromptSubmitSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"` // Injected as model context
}

// FeatureCodexHooks is the feature flag key to enable hooks in config.toml.
const FeatureCodexHooks = "codex_hooks"
