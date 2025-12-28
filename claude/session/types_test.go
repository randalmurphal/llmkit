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
	if msg.Content != "Hello, Claude!" {
		t.Errorf("expected content 'Hello, Claude!', got %q", msg.Content)
	}
}
