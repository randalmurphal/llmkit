package session

import (
	"testing"
)

func TestParseOutputMessage_Init(t *testing.T) {
	input := []byte(`{"type":"system","subtype":"init","cwd":"/home/user","session_id":"abc-123","model":"claude-opus-4-5-20251101","tools":["Read","Write"],"permissionMode":"bypassPermissions","claude_code_version":"2.0.76","apiKeySource":"none"}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != "system" {
		t.Errorf("expected type 'system', got %q", msg.Type)
	}
	if msg.Subtype != "init" {
		t.Errorf("expected subtype 'init', got %q", msg.Subtype)
	}
	if msg.SessionID != "abc-123" {
		t.Errorf("expected session_id 'abc-123', got %q", msg.SessionID)
	}
	if !msg.IsInit() {
		t.Error("expected IsInit() to return true")
	}
	if msg.Init == nil {
		t.Fatal("expected Init to be populated")
	}
	if msg.Init.Model != "claude-opus-4-5-20251101" {
		t.Errorf("expected model 'claude-opus-4-5-20251101', got %q", msg.Init.Model)
	}
	if msg.Init.CWD != "/home/user" {
		t.Errorf("expected cwd '/home/user', got %q", msg.Init.CWD)
	}
	if len(msg.Init.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(msg.Init.Tools))
	}
}

func TestParseOutputMessage_Assistant(t *testing.T) {
	input := []byte(`{"type":"assistant","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-opus-4-5-20251101","content":[{"type":"text","text":"Hello, world!"}],"stop_reason":null,"usage":{"input_tokens":100,"output_tokens":10}},"session_id":"abc-123"}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != "assistant" {
		t.Errorf("expected type 'assistant', got %q", msg.Type)
	}
	if !msg.IsAssistant() {
		t.Error("expected IsAssistant() to return true")
	}
	if msg.Assistant == nil {
		t.Fatal("expected Assistant to be populated")
	}
	if msg.Assistant.Message.ID != "msg_123" {
		t.Errorf("expected message id 'msg_123', got %q", msg.Assistant.Message.ID)
	}
	if len(msg.Assistant.Message.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(msg.Assistant.Message.Content))
	}
	if msg.Assistant.Message.Content[0].Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %q", msg.Assistant.Message.Content[0].Text)
	}

	text := msg.GetText()
	if text != "Hello, world!" {
		t.Errorf("expected GetText() to return 'Hello, world!', got %q", text)
	}
}

func TestParseOutputMessage_Result(t *testing.T) {
	input := []byte(`{"type":"result","subtype":"success","is_error":false,"result":"Done","session_id":"abc-123","duration_ms":1000,"duration_api_ms":500,"num_turns":1,"total_cost_usd":0.05,"usage":{"input_tokens":100,"output_tokens":50}}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if msg.Type != "result" {
		t.Errorf("expected type 'result', got %q", msg.Type)
	}
	if !msg.IsResult() {
		t.Error("expected IsResult() to return true")
	}
	if !msg.IsSuccess() {
		t.Error("expected IsSuccess() to return true")
	}
	if msg.IsError() {
		t.Error("expected IsError() to return false")
	}
	if msg.Result == nil {
		t.Fatal("expected Result to be populated")
	}
	if msg.Result.TotalCostUSD != 0.05 {
		t.Errorf("expected cost 0.05, got %f", msg.Result.TotalCostUSD)
	}
	if msg.Result.NumTurns != 1 {
		t.Errorf("expected 1 turn, got %d", msg.Result.NumTurns)
	}

	text := msg.GetText()
	if text != "Done" {
		t.Errorf("expected GetText() to return 'Done', got %q", text)
	}
}

func TestParseOutputMessage_Hook(t *testing.T) {
	input := []byte(`{"type":"system","subtype":"hook_response","session_id":"abc-123","hook_name":"SessionStart:startup","hook_event":"SessionStart","stdout":"Hook output","stderr":"","exit_code":0}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if !msg.IsHook() {
		t.Error("expected IsHook() to return true")
	}
	if msg.Hook == nil {
		t.Fatal("expected Hook to be populated")
	}
	if msg.Hook.HookName != "SessionStart:startup" {
		t.Errorf("expected hook_name 'SessionStart:startup', got %q", msg.Hook.HookName)
	}
	if msg.Hook.Stdout != "Hook output" {
		t.Errorf("expected stdout 'Hook output', got %q", msg.Hook.Stdout)
	}
}

func TestParseOutputMessage_Error(t *testing.T) {
	input := []byte(`{"type":"result","subtype":"error","is_error":true,"result":"Something failed","session_id":"abc-123"}`)

	msg, err := ParseOutputMessage(input)
	if err != nil {
		t.Fatalf("ParseOutputMessage failed: %v", err)
	}

	if !msg.IsError() {
		t.Error("expected IsError() to return true")
	}
	if msg.IsSuccess() {
		t.Error("expected IsSuccess() to return false")
	}
}

func TestParseOutputMessage_InvalidJSON(t *testing.T) {
	input := []byte(`not valid json`)

	_, err := ParseOutputMessage(input)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestNewUserMessage(t *testing.T) {
	msg := NewUserMessage("Hello, Claude!")

	if msg.Type != "user" {
		t.Errorf("expected type 'user', got %q", msg.Type)
	}
	if msg.Message.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Message.Role)
	}
	if msg.Message.Content != "Hello, Claude!" {
		t.Errorf("expected content 'Hello, Claude!', got %q", msg.Message.Content)
	}
}

// =============================================================================
// JSONL Types Tests
// =============================================================================

func TestParseJSONLMessage_User(t *testing.T) {
	input := []byte(`{"type":"user","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"sess-123","uuid":"msg-1","message":{"role":"user","content":[{"type":"text","text":"Hello"}]}}`)

	msg, err := ParseJSONLMessage(input)
	if err != nil {
		t.Fatalf("ParseJSONLMessage failed: %v", err)
	}

	if msg.Type != "user" {
		t.Errorf("expected type 'user', got %q", msg.Type)
	}
	if !msg.IsUser() {
		t.Error("expected IsUser() to return true")
	}
	if msg.IsAssistant() {
		t.Error("expected IsAssistant() to return false")
	}
	if msg.UUID != "msg-1" {
		t.Errorf("expected UUID 'msg-1', got %q", msg.UUID)
	}
	if msg.SessionID != "sess-123" {
		t.Errorf("expected sessionId 'sess-123', got %q", msg.SessionID)
	}
}

func TestParseJSONLMessage_Assistant(t *testing.T) {
	input := []byte(`{"type":"assistant","timestamp":"2024-01-19T10:00:01.000Z","sessionId":"sess-123","uuid":"msg-2","parentUuid":"msg-1","message":{"role":"assistant","content":[{"type":"text","text":"Hello there!"}],"model":"claude-opus-4-5-20251101","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":1000,"cache_read_input_tokens":500}}}`)

	msg, err := ParseJSONLMessage(input)
	if err != nil {
		t.Fatalf("ParseJSONLMessage failed: %v", err)
	}

	if msg.Type != "assistant" {
		t.Errorf("expected type 'assistant', got %q", msg.Type)
	}
	if !msg.IsAssistant() {
		t.Error("expected IsAssistant() to return true")
	}
	if msg.IsUser() {
		t.Error("expected IsUser() to return false")
	}
	if msg.ParentUUID == nil || *msg.ParentUUID != "msg-1" {
		t.Errorf("expected parentUuid 'msg-1', got %v", msg.ParentUUID)
	}

	// Test GetModel
	if model := msg.GetModel(); model != "claude-opus-4-5-20251101" {
		t.Errorf("expected model 'claude-opus-4-5-20251101', got %q", model)
	}

	// Test GetUsage
	usage := msg.GetUsage()
	if usage == nil {
		t.Fatal("expected usage to be populated")
	}
	if usage.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", usage.OutputTokens)
	}
	if usage.CacheCreationInputTokens != 1000 {
		t.Errorf("expected 1000 cache creation tokens, got %d", usage.CacheCreationInputTokens)
	}
	if usage.CacheReadInputTokens != 500 {
		t.Errorf("expected 500 cache read tokens, got %d", usage.CacheReadInputTokens)
	}

	// Test GetText
	text := msg.GetText()
	if text != "Hello there!" {
		t.Errorf("expected text 'Hello there!', got %q", text)
	}
}

func TestJSONLMessage_GetContentBlocks(t *testing.T) {
	input := []byte(`{"type":"assistant","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"sess-123","uuid":"msg-1","message":{"role":"assistant","content":[{"type":"text","text":"Let me read that"},{"type":"tool_use","id":"tool-1","name":"Read","input":{"file_path":"/test.txt"}}]}}`)

	msg, err := ParseJSONLMessage(input)
	if err != nil {
		t.Fatalf("ParseJSONLMessage failed: %v", err)
	}

	blocks := msg.GetContentBlocks()
	if len(blocks) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(blocks))
	}

	// First block is text
	if blocks[0].Type != "text" {
		t.Errorf("expected first block type 'text', got %q", blocks[0].Type)
	}
	if blocks[0].Text != "Let me read that" {
		t.Errorf("expected text 'Let me read that', got %q", blocks[0].Text)
	}

	// Second block is tool_use
	if blocks[1].Type != "tool_use" {
		t.Errorf("expected second block type 'tool_use', got %q", blocks[1].Type)
	}
	if blocks[1].Name != "Read" {
		t.Errorf("expected tool name 'Read', got %q", blocks[1].Name)
	}
}

func TestJSONLMessage_GetToolCalls(t *testing.T) {
	input := []byte(`{"type":"assistant","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"sess-123","uuid":"msg-1","message":{"role":"assistant","content":[{"type":"text","text":"I will read two files"},{"type":"tool_use","id":"tool-1","name":"Read","input":{"file_path":"/a.txt"}},{"type":"tool_use","id":"tool-2","name":"Read","input":{"file_path":"/b.txt"}}]}}`)

	msg, err := ParseJSONLMessage(input)
	if err != nil {
		t.Fatalf("ParseJSONLMessage failed: %v", err)
	}

	tools := msg.GetToolCalls()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(tools))
	}

	if tools[0].ID != "tool-1" {
		t.Errorf("expected first tool ID 'tool-1', got %q", tools[0].ID)
	}
	if tools[1].ID != "tool-2" {
		t.Errorf("expected second tool ID 'tool-2', got %q", tools[1].ID)
	}
}

func TestJSONLMessage_HasTodoUpdate(t *testing.T) {
	// Message without todo update
	input1 := []byte(`{"type":"assistant","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"sess-123","uuid":"msg-1","message":{"role":"assistant","content":[]}}`)
	msg1, _ := ParseJSONLMessage(input1)
	if msg1.HasTodoUpdate() {
		t.Error("expected HasTodoUpdate() to return false for message without todos")
	}

	// Message with todo update
	input2 := []byte(`{"type":"assistant","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"sess-123","uuid":"msg-2","message":{"role":"assistant","content":[]},"toolUseResult":{"newTodos":[{"content":"Write code","status":"pending","activeForm":"Writing code"}]}}`)
	msg2, _ := ParseJSONLMessage(input2)
	if !msg2.HasTodoUpdate() {
		t.Error("expected HasTodoUpdate() to return true for message with todos")
	}

	// Test GetTodos
	todos := msg2.GetTodos()
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Content != "Write code" {
		t.Errorf("expected todo content 'Write code', got %q", todos[0].Content)
	}
	if todos[0].Status != "pending" {
		t.Errorf("expected todo status 'pending', got %q", todos[0].Status)
	}
	if todos[0].ActiveForm != "Writing code" {
		t.Errorf("expected todo activeForm 'Writing code', got %q", todos[0].ActiveForm)
	}
}

func TestJSONLMessage_NilReceiverSafety(t *testing.T) {
	var msg *JSONLMessage = nil

	// These should not panic
	if msg.IsUser() {
		t.Error("expected nil.IsUser() to return false")
	}
	if msg.IsAssistant() {
		t.Error("expected nil.IsAssistant() to return false")
	}
}

func TestJSONLMessage_NilFieldSafety(t *testing.T) {
	// Message with nil Message field
	msg := &JSONLMessage{Type: "assistant"}

	// These should return empty/nil without panic
	if model := msg.GetModel(); model != "" {
		t.Errorf("expected empty model, got %q", model)
	}
	if usage := msg.GetUsage(); usage != nil {
		t.Error("expected nil usage")
	}
	if blocks := msg.GetContentBlocks(); blocks != nil {
		t.Error("expected nil content blocks")
	}
	if text := msg.GetText(); text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
	if tools := msg.GetToolCalls(); tools != nil {
		t.Error("expected nil tool calls")
	}
	if todos := msg.GetTodos(); todos != nil {
		t.Error("expected nil todos")
	}
}

func TestParseJSONLMessage_InvalidJSON(t *testing.T) {
	input := []byte(`not valid json`)

	_, err := ParseJSONLMessage(input)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
