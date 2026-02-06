package codexcontract

// Canonical JSONL event types from codex exec --json.
const (
	EventThreadStarted = "thread.started"
	EventTurnStarted   = "turn.started"
	EventTurnCompleted = "turn.completed"
	EventTurnFailed    = "turn.failed"
	EventItemStarted   = "item.started"
	EventItemUpdated   = "item.updated"
	EventItemCompleted = "item.completed"
	EventError         = "error"

	// Legacy/compat event types seen in older wrappers or test fakes.
	EventContent   = "content"
	EventText      = "text"
	EventAssistant = "assistant"
	EventMessage   = "message"
	EventToolCall  = "tool_call"
	EventDone      = "done"
	EventComplete  = "complete"
	EventEnd       = "end"
	EventUsage     = "usage"
	EventSession   = "session"
	EventResult    = "result"
)

// Item types we can interpret in headless mode.
const (
	ItemAgentMessage = "agent_message"
	ItemReasoning    = "reasoning"
)
