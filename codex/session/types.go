package session

import (
	"encoding/json"
	"time"

	"github.com/randalmurphal/llmkit/codexcontract"
)

// SessionStatus represents the current state of a session.
type SessionStatus string

// Session status constants.
const (
	StatusCreating SessionStatus = "creating"
	StatusActive   SessionStatus = "active"
	StatusClosing  SessionStatus = "closing"
	StatusClosed   SessionStatus = "closed"
	StatusError    SessionStatus = "error"
)

// JSON-RPC 2.0 method constants for the Codex app-server protocol.
const (
	MethodThreadStart  = "thread/start"
	MethodThreadResume = "thread/resume"
	MethodTurnStart    = "turn/start"
	MethodTurnSteer    = "turn/steer"
	MethodShutdown     = "shutdown"
)

// JSONRPCVersion is the JSON-RPC protocol version.
const JSONRPCVersion = "2.0"

// JSONRPCRequest is a JSON-RPC 2.0 request sent to the app-server.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  any         `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response from the app-server.
// It contains either a result or an error, but not both.
type JSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *int64           `json:"id"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// JSONRPCNotification is a JSON-RPC 2.0 notification from the app-server.
// Notifications have no ID field and do not expect a response.
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCError represents an error in a JSON-RPC 2.0 response.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *JSONRPCError) Error() string {
	return e.Message
}

// ThreadStartParams are the parameters for thread/start.
type ThreadStartParams struct{}

// ThreadResumeParams are the parameters for thread/resume.
type ThreadResumeParams struct {
	ThreadID string `json:"threadId"`
}

// ThreadStartResult is the result of a thread/start or thread/resume call.
type ThreadStartResult struct {
	ThreadID string `json:"threadId"`
}

// InputItem represents a content item in a turn/start or turn/steer request.
type InputItem struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// TurnStartParams are the parameters for turn/start.
type TurnStartParams struct {
	ThreadID string      `json:"threadId"`
	Input    []InputItem `json:"input"`
}

// TurnSteerParams are the parameters for turn/steer.
// This injects input into an actively running turn.
type TurnSteerParams struct {
	ThreadID       string      `json:"threadId"`
	Input          []InputItem `json:"input"`
	ExpectedTurnID string      `json:"expectedTurnId,omitempty"`
}

// OutputMessage represents a notification received from the Codex app-server.
// Use the Is*() methods to determine the notification type, and GetText() to
// extract text content.
type OutputMessage struct {
	// Type is the notification event type (e.g., "thread.started", "item.updated").
	Type string `json:"type"`

	// ThreadID is the thread this notification belongs to.
	ThreadID string `json:"threadId,omitempty"`

	// TurnID is the turn this notification belongs to.
	TurnID string `json:"turnId,omitempty"`

	// ItemID is the item this notification belongs to (for item.* events).
	ItemID string `json:"itemId,omitempty"`

	// ItemType is the kind of item (e.g., "agent_message", "reasoning").
	ItemType string `json:"itemType,omitempty"`

	// Content holds text content for item updates.
	Content string `json:"content,omitempty"`

	// Done is true when the turn or item has completed.
	Done bool `json:"done,omitempty"`

	// Error holds error information when a turn fails.
	Error string `json:"error,omitempty"`

	// Raw holds the original JSON for advanced parsing.
	Raw []byte `json:"-"`
}

// IsTurnStarted returns true if this is a turn.started notification.
func (m *OutputMessage) IsTurnStarted() bool {
	return m.Type == codexcontract.EventTurnStarted
}

// IsTurnComplete returns true if this is a turn.completed notification.
func (m *OutputMessage) IsTurnComplete() bool {
	return m.Type == codexcontract.EventTurnCompleted
}

// IsTurnFailed returns true if this is a turn.failed notification.
func (m *OutputMessage) IsTurnFailed() bool {
	return m.Type == codexcontract.EventTurnFailed
}

// IsThreadStarted returns true if this is a thread.started notification.
func (m *OutputMessage) IsThreadStarted() bool {
	return m.Type == codexcontract.EventThreadStarted
}

// IsItemStarted returns true if this is an item.started notification.
func (m *OutputMessage) IsItemStarted() bool {
	return m.Type == codexcontract.EventItemStarted
}

// IsItemUpdate returns true if this is an item.updated notification.
func (m *OutputMessage) IsItemUpdate() bool {
	return m.Type == codexcontract.EventItemUpdated
}

// IsItemComplete returns true if this is an item.completed notification.
func (m *OutputMessage) IsItemComplete() bool {
	return m.Type == codexcontract.EventItemCompleted
}

// IsError returns true if this is an error notification or a failed turn.
func (m *OutputMessage) IsError() bool {
	return m.Type == codexcontract.EventError || m.Type == codexcontract.EventTurnFailed
}

// IsAgentMessage returns true if this is an agent_message item.
func (m *OutputMessage) IsAgentMessage() bool {
	return m.ItemType == codexcontract.ItemAgentMessage
}

// IsReasoning returns true if this is a reasoning item.
func (m *OutputMessage) IsReasoning() bool {
	return m.ItemType == codexcontract.ItemReasoning
}

// GetText returns the text content of the message.
// For item updates, returns the content field.
// For errors, returns the error field.
func (m *OutputMessage) GetText() string {
	if m.Content != "" {
		return m.Content
	}
	if m.Error != "" {
		return m.Error
	}
	return ""
}

// SessionInfo contains metadata about a session.
type SessionInfo struct {
	ID           string        `json:"id"`
	ThreadID     string        `json:"thread_id"`
	Status       SessionStatus `json:"status"`
	Model        string        `json:"model"`
	CWD          string        `json:"cwd"`
	CreatedAt    time.Time     `json:"created_at"`
	LastActivity time.Time     `json:"last_activity"`
	TurnCount    int           `json:"turn_count"`
}

// UserMessage is a message to send to Codex via the turn/start method.
type UserMessage struct {
	Content string
}

// NewUserMessage creates a new user message for sending to Codex.
func NewUserMessage(content string) UserMessage {
	return UserMessage{Content: content}
}

// ParseOutputMessage parses a JSON-RPC notification from the app-server into
// an OutputMessage. The incoming data should be a JSON-RPC notification with a
// method field corresponding to a Codex event type and a params object containing
// event-specific fields.
func ParseOutputMessage(data []byte) (*OutputMessage, error) {
	// First try to parse as a JSON-RPC notification envelope.
	var notification JSONRPCNotification
	if err := json.Unmarshal(data, &notification); err != nil {
		return nil, err
	}

	// If it has a method field, it's a JSON-RPC notification - extract the params.
	if notification.Method != "" {
		msg := &OutputMessage{
			Type: notification.Method,
			Raw:  data,
		}
		// Parse params into the message fields if present.
		if len(notification.Params) > 0 {
			if err := json.Unmarshal(notification.Params, msg); err != nil {
				return nil, err
			}
			// Restore the type from the method (params may not have a "type" field).
			msg.Type = notification.Method
		}
		msg.Raw = data
		return msg, nil
	}

	// Fallback: parse as a bare event object (for forward compatibility).
	var msg OutputMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	msg.Raw = data

	// If the type contains a slash or dot, it's a known event format.
	// If it's empty, try to infer from presence of error field.
	if msg.Type == "" && msg.Error != "" {
		msg.Type = codexcontract.EventError
	}

	return &msg, nil
}

// parseJSONRPCLine classifies a line from stdout as either a JSON-RPC response
// (has an "id" field) or a notification (has a "method" field). Returns the
// response if it's a response, or nil if it's a notification.
func parseJSONRPCLine(data []byte) (*JSONRPCResponse, bool) {
	// Quick check: does it look like a response? Responses have "id" and "result"/"error".
	// Notifications have "method" but no "id".
	// We use a partial parse to avoid double-unmarshaling.
	var probe struct {
		ID     *int64          `json:"id"`
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  *JSONRPCError   `json:"error"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, false
	}

	// If it has an ID, it's a response.
	if probe.ID != nil {
		resp := &JSONRPCResponse{
			JSONRPC: JSONRPCVersion,
			ID:      probe.ID,
			Result:  probe.Result,
			Error:   probe.Error,
		}
		return resp, true
	}

	// No ID means it's a notification (or malformed). Not a response.
	return nil, false
}

// isTerminalEvent returns true if this notification type signals the end of a turn.
func isTerminalEvent(eventType string) bool {
	return eventType == codexcontract.EventTurnCompleted ||
		eventType == codexcontract.EventTurnFailed ||
		eventType == codexcontract.EventError
}
