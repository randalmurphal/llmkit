package claudecontract

// Stream event types from CLI stream-json output.
// Based on official TypeScript/Python SDK documentation.
const (
	// EventTypeSystem is used for init, hook_response, and compact_boundary events.
	EventTypeSystem = "system"

	// EventTypeAssistant is for assistant messages (model responses).
	EventTypeAssistant = "assistant"

	// EventTypeUser is for user messages (including tool results).
	EventTypeUser = "user"

	// EventTypeResult is the final result message with stats.
	EventTypeResult = "result"

	// EventTypeStreamEvent is for partial message streaming (when include_partial_messages is true).
	EventTypeStreamEvent = "stream_event"
)

// System event subtypes.
const (
	// SubtypeInit is the initialization event at session start.
	SubtypeInit = "init"

	// SubtypeHookResponse is for hook execution results.
	SubtypeHookResponse = "hook_response"

	// SubtypeCompactBoundary indicates a conversation compaction boundary.
	SubtypeCompactBoundary = "compact_boundary"
)

// Result subtypes indicating how the session ended.
// From official SDK documentation.
const (
	// ResultSubtypeSuccess indicates successful completion.
	ResultSubtypeSuccess = "success"

	// ResultSubtypeErrorMaxTurns indicates max turns limit reached.
	ResultSubtypeErrorMaxTurns = "error_max_turns"

	// ResultSubtypeErrorDuringExecution indicates an error occurred during execution.
	ResultSubtypeErrorDuringExecution = "error_during_execution"

	// ResultSubtypeErrorMaxBudgetUSD indicates budget limit reached.
	ResultSubtypeErrorMaxBudgetUSD = "error_max_budget_usd"

	// ResultSubtypeErrorMaxStructuredOutputRetries indicates structured output retries exceeded.
	ResultSubtypeErrorMaxStructuredOutputRetries = "error_max_structured_output_retries"
)

// Content block types within messages.
const (
	// ContentTypeText is a text content block.
	ContentTypeText = "text"

	// ContentTypeToolUse is a tool invocation request.
	ContentTypeToolUse = "tool_use"

	// ContentTypeToolResult is the result of a tool invocation.
	ContentTypeToolResult = "tool_result"

	// ContentTypeThinking is a thinking block (for models with thinking capability).
	ContentTypeThinking = "thinking"
)

// Message roles.
const (
	// RoleUser is the user role in messages.
	RoleUser = "user"

	// RoleAssistant is the assistant role in messages.
	RoleAssistant = "assistant"
)

// API key sources.
const (
	// APIKeySourceUser indicates the API key came from user settings.
	APIKeySourceUser = "user"

	// APIKeySourceProject indicates the API key came from project settings.
	APIKeySourceProject = "project"

	// APIKeySourceOrg indicates the API key came from organization settings.
	APIKeySourceOrg = "org"

	// APIKeySourceTemporary indicates a temporary API key.
	APIKeySourceTemporary = "temporary"
)

// MCP server status values.
const (
	// MCPStatusConnected indicates the MCP server is connected.
	MCPStatusConnected = "connected"

	// MCPStatusFailed indicates the MCP server connection failed.
	MCPStatusFailed = "failed"

	// MCPStatusNeedsAuth indicates the MCP server needs authentication.
	MCPStatusNeedsAuth = "needs-auth"

	// MCPStatusPending indicates the MCP server connection is pending.
	MCPStatusPending = "pending"
)

// Stop reasons for assistant messages.
const (
	// StopReasonEndTurn indicates the assistant ended its turn naturally.
	StopReasonEndTurn = "end_turn"

	// StopReasonMaxTokens indicates the max tokens limit was reached.
	StopReasonMaxTokens = "max_tokens"

	// StopReasonStopSequence indicates a stop sequence was encountered.
	StopReasonStopSequence = "stop_sequence"

	// StopReasonToolUse indicates the assistant is waiting for tool results.
	StopReasonToolUse = "tool_use"
)
