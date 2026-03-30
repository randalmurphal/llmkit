package session

import (
	"encoding/json"
	"testing"

	"github.com/randalmurphal/llmkit/v2/codexcontract"
)

// =============================================================================
// ParseOutputMessage Tests
// =============================================================================

func TestParseOutputMessage_JSONRPCNotification(t *testing.T) {
	// JSON-RPC notification with method and params.
	input := []byte(`{"jsonrpc":"2.0","method":"item.updated","params":{"threadId":"t-1","turnId":"turn-1","itemId":"item-1","itemType":"agent_message","content":"Hello!"}}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventItemUpdated {
		t.Errorf("expected type %q, got %q", codexcontract.EventItemUpdated, msg.Type)
	}
	if msg.ThreadID != "t-1" {
		t.Errorf("expected threadId 't-1', got %q", msg.ThreadID)
	}
	if msg.TurnID != "turn-1" {
		t.Errorf("expected turnId 'turn-1', got %q", msg.TurnID)
	}
	if msg.ItemID != "item-1" {
		t.Errorf("expected itemId 'item-1', got %q", msg.ItemID)
	}
	if msg.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %q", msg.Content)
	}
	if msg.Raw == nil {
		t.Error("expected Raw to be populated")
	}
}

func TestParseOutputMessage_BareEventObject(t *testing.T) {
	// Bare event object without JSON-RPC envelope.
	input := []byte(`{"type":"turn.completed","threadId":"t-2","turnId":"turn-5","done":true}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventTurnCompleted {
		t.Errorf("expected type %q, got %q", codexcontract.EventTurnCompleted, msg.Type)
	}
	if msg.ThreadID != "t-2" {
		t.Errorf("expected threadId 't-2', got %q", msg.ThreadID)
	}
	if !msg.Done {
		t.Error("expected Done to be true")
	}
}

func TestParseOutputMessage_ErrorNotification(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","method":"turn.failed","params":{"threadId":"t-1","error":"out of tokens"}}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventTurnFailed {
		t.Errorf("expected type %q, got %q", codexcontract.EventTurnFailed, msg.Type)
	}
	if msg.Error != "out of tokens" {
		t.Errorf("expected error 'out of tokens', got %q", msg.Error)
	}
}

func TestParseOutputMessage_InfersErrorFromErrorField(t *testing.T) {
	// Bare object with no type but an error field.
	input := []byte(`{"error":"something went wrong"}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventError {
		t.Errorf("expected type %q, got %q", codexcontract.EventError, msg.Type)
	}
}

func TestParseOutputMessage_InvalidJSON(t *testing.T) {
	_, err := ParseOutputMessage([]byte(`not valid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseOutputMessage_EmptyParams(t *testing.T) {
	// Notification with method but empty/missing params.
	input := []byte(`{"jsonrpc":"2.0","method":"thread.started"}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventThreadStarted {
		t.Errorf("expected type %q, got %q", codexcontract.EventThreadStarted, msg.Type)
	}
}

// =============================================================================
// Slash-to-dot normalization Tests
// =============================================================================

func TestParseOutputMessage_SlashDelimitedMethod(t *testing.T) {
	// App-server sends slash-delimited methods; they should be normalized to dots.
	input := []byte(`{"jsonrpc":"2.0","method":"item/updated","params":{"threadId":"t-1","turnId":"turn-1","itemId":"item-1","itemType":"agent_message","content":"Hello!"}}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventItemUpdated {
		t.Errorf("expected type %q, got %q", codexcontract.EventItemUpdated, msg.Type)
	}
	if msg.ThreadID != "t-1" {
		t.Errorf("expected threadId 't-1', got %q", msg.ThreadID)
	}
	if msg.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %q", msg.Content)
	}
}

func TestParseOutputMessage_TurnStartedWithNestedTurnID(t *testing.T) {
	// turn/started carries the turn ID nested inside params.turn.id, not params.turnId.
	input := []byte(`{"jsonrpc":"2.0","method":"turn/started","params":{"threadId":"t-1","turn":{"id":"turn-abc","items":[],"status":"running"}}}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventTurnStarted {
		t.Errorf("expected type %q, got %q", codexcontract.EventTurnStarted, msg.Type)
	}
	if msg.TurnID != "turn-abc" {
		t.Errorf("expected turnId 'turn-abc', got %q", msg.TurnID)
	}
	if msg.ThreadID != "t-1" {
		t.Errorf("expected threadId 't-1', got %q", msg.ThreadID)
	}
}

func TestParseOutputMessage_TurnCompletedWithNestedTurnID(t *testing.T) {
	// turn/completed also carries a nested Turn object.
	input := []byte(`{"jsonrpc":"2.0","method":"turn/completed","params":{"threadId":"t-2","turn":{"id":"turn-xyz","items":[],"status":"completed"}}}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != codexcontract.EventTurnCompleted {
		t.Errorf("expected type %q, got %q", codexcontract.EventTurnCompleted, msg.Type)
	}
	if msg.TurnID != "turn-xyz" {
		t.Errorf("expected turnId 'turn-xyz', got %q", msg.TurnID)
	}
}

func TestParseOutputMessage_FlatTurnIDTakesPriority(t *testing.T) {
	// If both flat turnId and nested turn.id exist, flat takes priority (it's parsed first).
	input := []byte(`{"jsonrpc":"2.0","method":"turn/started","params":{"threadId":"t-1","turnId":"flat-id","turn":{"id":"nested-id","items":[],"status":"running"}}}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.TurnID != "flat-id" {
		t.Errorf("expected flat turnId to take priority, got %q", msg.TurnID)
	}
}

func TestNormalizeEventType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"turn/started", "turn.started"},
		{"turn/completed", "turn.completed"},
		{"item/updated", "item.updated"},
		{"thread/started", "thread.started"},
		{"turn/diff/updated", "turn.diff.updated"},
		{"turn.started", "turn.started"}, // Already dotted
		{"error", "error"},               // No separator
		{"", ""},                         // Empty
	}

	for _, tt := range tests {
		got := normalizeEventType(tt.input)
		if got != tt.want {
			t.Errorf("normalizeEventType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// =============================================================================
// parseJSONRPCLine Tests
// =============================================================================

func TestParseJSONRPCLine_Response(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":1,"result":{"thread":{"id":"t-1"},"model":"gpt-5","cwd":"/tmp"}}`)

	resp, isResp := parseJSONRPCLine(input)
	if !isResp {
		t.Fatal("expected line to be classified as a response")
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID == nil || *resp.ID != 1 {
		t.Errorf("expected ID 1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}

	var result ThreadStartResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if result.Thread.ID != "t-1" {
		t.Errorf("expected thread.id 't-1', got %q", result.Thread.ID)
	}
}

func TestParseJSONRPCLine_ResponseWithError(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"invalid request"}}`)

	resp, isResp := parseJSONRPCLine(input)
	if !isResp {
		t.Fatal("expected line to be classified as a response")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("expected error code -32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "invalid request" {
		t.Errorf("expected error message 'invalid request', got %q", resp.Error.Message)
	}
}

func TestParseJSONRPCLine_Notification(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","method":"item.updated","params":{"content":"text"}}`)

	resp, isResp := parseJSONRPCLine(input)
	if isResp {
		t.Error("notification should not be classified as a response")
	}
	if resp != nil {
		t.Error("expected nil response for notification")
	}
}

func TestParseJSONRPCLine_InvalidJSON(t *testing.T) {
	resp, isResp := parseJSONRPCLine([]byte(`not json`))
	if isResp {
		t.Error("invalid JSON should not be classified as a response")
	}
	if resp != nil {
		t.Error("expected nil response for invalid JSON")
	}
}

// =============================================================================
// OutputMessage Is*() Helper Tests
// =============================================================================

func TestOutputMessage_IsTurnStarted(t *testing.T) {
	msg := &OutputMessage{Type: codexcontract.EventTurnStarted}
	if !msg.IsTurnStarted() {
		t.Error("expected IsTurnStarted() to return true")
	}
	msg.Type = codexcontract.EventTurnCompleted
	if msg.IsTurnStarted() {
		t.Error("expected IsTurnStarted() to return false")
	}
}

func TestOutputMessage_IsTurnComplete(t *testing.T) {
	msg := &OutputMessage{Type: codexcontract.EventTurnCompleted}
	if !msg.IsTurnComplete() {
		t.Error("expected IsTurnComplete() to return true")
	}
}

func TestOutputMessage_IsTurnFailed(t *testing.T) {
	msg := &OutputMessage{Type: codexcontract.EventTurnFailed}
	if !msg.IsTurnFailed() {
		t.Error("expected IsTurnFailed() to return true")
	}
}

func TestOutputMessage_IsThreadStarted(t *testing.T) {
	msg := &OutputMessage{Type: codexcontract.EventThreadStarted}
	if !msg.IsThreadStarted() {
		t.Error("expected IsThreadStarted() to return true")
	}
}

func TestOutputMessage_IsItemStarted(t *testing.T) {
	msg := &OutputMessage{Type: codexcontract.EventItemStarted}
	if !msg.IsItemStarted() {
		t.Error("expected IsItemStarted() to return true")
	}
}

func TestOutputMessage_IsItemUpdate(t *testing.T) {
	msg := &OutputMessage{Type: codexcontract.EventItemUpdated}
	if !msg.IsItemUpdate() {
		t.Error("expected IsItemUpdate() to return true")
	}
}

func TestOutputMessage_IsItemComplete(t *testing.T) {
	msg := &OutputMessage{Type: codexcontract.EventItemCompleted}
	if !msg.IsItemComplete() {
		t.Error("expected IsItemComplete() to return true")
	}
}

func TestOutputMessage_IsError(t *testing.T) {
	// "error" type
	msg := &OutputMessage{Type: codexcontract.EventError}
	if !msg.IsError() {
		t.Error("expected IsError() to return true for error type")
	}

	// "turn.failed" also counts as error
	msg.Type = codexcontract.EventTurnFailed
	if !msg.IsError() {
		t.Error("expected IsError() to return true for turn.failed type")
	}

	// Other types should not be error
	msg.Type = codexcontract.EventTurnCompleted
	if msg.IsError() {
		t.Error("expected IsError() to return false for turn.completed type")
	}
}

func TestOutputMessage_IsAgentMessage(t *testing.T) {
	msg := &OutputMessage{ItemType: codexcontract.ItemAgentMessage}
	if !msg.IsAgentMessage() {
		t.Error("expected IsAgentMessage() to return true")
	}

	msg.ItemType = codexcontract.ItemReasoning
	if msg.IsAgentMessage() {
		t.Error("expected IsAgentMessage() to return false for reasoning")
	}
}

func TestOutputMessage_IsReasoning(t *testing.T) {
	msg := &OutputMessage{ItemType: codexcontract.ItemReasoning}
	if !msg.IsReasoning() {
		t.Error("expected IsReasoning() to return true")
	}
}

// =============================================================================
// GetText() Tests
// =============================================================================

func TestOutputMessage_GetText_Content(t *testing.T) {
	msg := &OutputMessage{Content: "hello"}
	if got := msg.GetText(); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestOutputMessage_GetText_Error(t *testing.T) {
	msg := &OutputMessage{Error: "something failed"}
	if got := msg.GetText(); got != "something failed" {
		t.Errorf("expected 'something failed', got %q", got)
	}
}

func TestOutputMessage_GetText_ContentTakesPriority(t *testing.T) {
	msg := &OutputMessage{Content: "content", Error: "error"}
	if got := msg.GetText(); got != "content" {
		t.Errorf("expected 'content' (priority over error), got %q", got)
	}
}

func TestOutputMessage_GetText_Empty(t *testing.T) {
	msg := &OutputMessage{}
	if got := msg.GetText(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// =============================================================================
// NewUserMessage Tests
// =============================================================================

func TestNewUserMessage(t *testing.T) {
	msg := NewUserMessage("Hello, Codex!")
	if msg.Content != "Hello, Codex!" {
		t.Errorf("expected content 'Hello, Codex!', got %q", msg.Content)
	}
}

func TestNewUserMessage_Empty(t *testing.T) {
	msg := NewUserMessage("")
	if msg.Content != "" {
		t.Errorf("expected empty content, got %q", msg.Content)
	}
}

// =============================================================================
// JSONRPCError Tests
// =============================================================================

func TestJSONRPCError_Error(t *testing.T) {
	err := &JSONRPCError{Code: -32600, Message: "invalid request"}
	if err.Error() != "invalid request" {
		t.Errorf("expected 'invalid request', got %q", err.Error())
	}
}

// =============================================================================
// isTerminalEvent Tests
// =============================================================================

func TestIsTerminalEvent(t *testing.T) {
	tests := []struct {
		eventType string
		want      bool
	}{
		{codexcontract.EventTurnCompleted, true},
		{codexcontract.EventTurnFailed, true},
		{codexcontract.EventError, true},
		{"error.something", false},
		{codexcontract.EventTurnStarted, false},
		{codexcontract.EventItemUpdated, false},
		{codexcontract.EventThreadStarted, false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if got := isTerminalEvent(tt.eventType); got != tt.want {
				t.Errorf("isTerminalEvent(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Session Status Constants Tests
// =============================================================================

func TestSessionStatusConstants(t *testing.T) {
	if StatusCreating != "creating" {
		t.Errorf("StatusCreating = %q, want 'creating'", StatusCreating)
	}
	if StatusActive != "active" {
		t.Errorf("StatusActive = %q, want 'active'", StatusActive)
	}
	if StatusClosing != "closing" {
		t.Errorf("StatusClosing = %q, want 'closing'", StatusClosing)
	}
	if StatusClosed != "closed" {
		t.Errorf("StatusClosed = %q, want 'closed'", StatusClosed)
	}
	if StatusError != "error" {
		t.Errorf("StatusError = %q, want 'error'", StatusError)
	}
}
