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

// App-server streaming event types. These use slash-delimited method names
// that get normalized to dot-delimited by ParseOutputMessage.
const (
	EventAgentMessageDelta = "item.agentMessage.delta"
	EventReasoningDelta    = "item.reasoning.delta"
)

// Item types we can interpret in headless mode.
// The app-server uses camelCase ("agentMessage"), while codex exec --json
// uses snake_case ("agent_message"). Both are accepted.
const (
	ItemAgentMessage      = "agent_message"
	ItemAgentMessageCamel = "agentMessage"
	ItemReasoning         = "reasoning"
)
