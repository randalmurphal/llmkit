package local

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestRequest_Marshal(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		Method:  "complete",
		Params:  map[string]string{"model": "test"},
		ID:      1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify JSON structure
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want %v", parsed["jsonrpc"], "2.0")
	}
	if parsed["method"] != "complete" {
		t.Errorf("method = %v, want %v", parsed["method"], "complete")
	}
	if int(parsed["id"].(float64)) != 1 {
		t.Errorf("id = %v, want %v", parsed["id"], 1)
	}
}

func TestResponse_Unmarshal(t *testing.T) {
	data := `{"jsonrpc":"2.0","result":{"content":"hello"},"id":1}`

	var resp Response
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want %v", resp.JSONRPC, "2.0")
	}
	if resp.ID != 1 {
		t.Errorf("ID = %v, want %v", resp.ID, 1)
	}
	if resp.Error != nil {
		t.Errorf("Error = %v, want nil", resp.Error)
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Unmarshal result error = %v", err)
	}
	if result["content"] != "hello" {
		t.Errorf("result.content = %v, want %v", result["content"], "hello")
	}
}

func TestResponse_UnmarshalError(t *testing.T) {
	data := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":1}`

	var resp Response
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Error = nil, want error")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Error.Code = %v, want %v", resp.Error.Code, -32600)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("Error.Message = %v, want %v", resp.Error.Message, "Invalid Request")
	}
}

func TestRPCError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *RPCError
		wantMsg string
	}{
		{
			name:    "without data",
			err:     &RPCError{Code: -32600, Message: "Invalid Request"},
			wantMsg: "RPC error -32600: Invalid Request",
		},
		{
			name:    "with data",
			err:     &RPCError{Code: -32000, Message: "Backend error", Data: json.RawMessage(`"details"`)},
			wantMsg: `RPC error -32000: Backend error (data: "details")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestNotification_Marshal(t *testing.T) {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "stream.chunk",
		Params:  map[string]any{"content": "text", "done": false},
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Should not have an "id" field
	if strings.Contains(string(data), `"id"`) {
		t.Error("Notification should not have id field")
	}
}

func TestProtocol_Call(t *testing.T) {
	// Create a pipe for testing
	response := `{"jsonrpc":"2.0","result":{"ready":true},"id":1}` + "\n"
	reader := strings.NewReader(response)
	var writer bytes.Buffer

	proto := NewProtocol(reader, &writer)

	var result InitResult
	err := proto.Call("init", InitParams{Backend: "ollama", Model: "test"}, &result)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if !result.Ready {
		t.Error("result.Ready = false, want true")
	}

	// Verify request was written
	written := writer.String()
	if !strings.Contains(written, `"method":"init"`) {
		t.Error("Request should contain method:init")
	}
	if !strings.Contains(written, `"jsonrpc":"2.0"`) {
		t.Error("Request should contain jsonrpc:2.0")
	}
}

func TestProtocol_Call_Error(t *testing.T) {
	response := `{"jsonrpc":"2.0","error":{"code":-32001,"message":"Model not found"},"id":1}` + "\n"
	reader := strings.NewReader(response)
	var writer bytes.Buffer

	proto := NewProtocol(reader, &writer)

	var result CompleteResult
	err := proto.Call("complete", CompleteParams{Model: "unknown"}, &result)
	if err == nil {
		t.Fatal("Call() should return error")
	}

	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error type = %T, want *RPCError", err)
	}
	if rpcErr.Code != -32001 {
		t.Errorf("Error.Code = %d, want %d", rpcErr.Code, -32001)
	}
}

func TestProtocol_Call_SkipsNotifications(t *testing.T) {
	// Response with notification before actual response
	response := `{"jsonrpc":"2.0","method":"stream.chunk","params":{"content":"test"}}` + "\n" +
		`{"jsonrpc":"2.0","result":{"ready":true},"id":1}` + "\n"
	reader := strings.NewReader(response)
	var writer bytes.Buffer

	proto := NewProtocol(reader, &writer)

	var result InitResult
	err := proto.Call("init", nil, &result)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if !result.Ready {
		t.Error("result.Ready = false, want true")
	}
}

func TestProtocol_Notify(t *testing.T) {
	var writer bytes.Buffer
	proto := NewProtocol(strings.NewReader(""), &writer)

	err := proto.Notify("stream.start", CompleteParams{Model: "test", Stream: true})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	written := writer.String()
	if !strings.Contains(written, `"method":"stream.start"`) {
		t.Error("Notification should contain method")
	}
	// Should not have id
	if strings.Contains(written, `"id":`) {
		t.Error("Notification should not have id")
	}
}

func TestProtocol_ReadMessage(t *testing.T) {
	msg := `{"jsonrpc":"2.0","method":"test"}` + "\n"
	proto := NewProtocol(strings.NewReader(msg), io.Discard)

	data, err := proto.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}

	if !strings.Contains(string(data), `"method":"test"`) {
		t.Error("ReadMessage() should return the message")
	}
}

func TestParseNotification(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantNil bool
		method  string
	}{
		{
			name:    "valid notification",
			data:    `{"jsonrpc":"2.0","method":"stream.chunk","params":{"content":"test"}}`,
			wantNil: false,
			method:  "stream.chunk",
		},
		{
			name:    "response (has id)",
			data:    `{"jsonrpc":"2.0","result":{},"id":1}`,
			wantNil: true,
		},
		{
			name:    "stream.done notification",
			data:    `{"jsonrpc":"2.0","method":"stream.done","params":{"usage":{"input_tokens":10}}}`,
			wantNil: false,
			method:  "stream.done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif, err := ParseNotification([]byte(tt.data))
			if err != nil {
				t.Fatalf("ParseNotification() error = %v", err)
			}

			if (notif == nil) != tt.wantNil {
				t.Errorf("ParseNotification() = %v, wantNil = %v", notif, tt.wantNil)
			}

			if notif != nil && notif.Method != tt.method {
				t.Errorf("Method = %v, want %v", notif.Method, tt.method)
			}
		})
	}
}

func TestParseStreamChunk(t *testing.T) {
	data := json.RawMessage(`{"content":"hello","done":false}`)

	chunk, err := ParseStreamChunk(data)
	if err != nil {
		t.Fatalf("ParseStreamChunk() error = %v", err)
	}

	if chunk.Content != "hello" {
		t.Errorf("Content = %v, want %v", chunk.Content, "hello")
	}
	if chunk.Done {
		t.Error("Done = true, want false")
	}
}

func TestParseStreamDone(t *testing.T) {
	data := json.RawMessage(`{"usage":{"input_tokens":10,"output_tokens":20,"total_tokens":30},"finish_reason":"stop"}`)

	done, err := ParseStreamDone(data)
	if err != nil {
		t.Fatalf("ParseStreamDone() error = %v", err)
	}

	if done.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %v, want %v", done.Usage.InputTokens, 10)
	}
	if done.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens = %v, want %v", done.Usage.OutputTokens, 20)
	}
	if done.FinishReason != "stop" {
		t.Errorf("FinishReason = %v, want %v", done.FinishReason, "stop")
	}
}

func TestIsNotification(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "notification",
			data: `{"jsonrpc":"2.0","method":"test"}`,
			want: true,
		},
		{
			name: "response",
			data: `{"jsonrpc":"2.0","result":{},"id":1}`,
			want: false,
		},
		{
			name: "request",
			data: `{"jsonrpc":"2.0","method":"test","id":1}`,
			want: false,
		},
		{
			name: "invalid json",
			data: `not json`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotification([]byte(tt.data)); got != tt.want {
				t.Errorf("IsNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompleteParams_Marshal(t *testing.T) {
	params := CompleteParams{
		Messages: []MessageParam{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		Model:        "llama3.2:latest",
		SystemPrompt: "You are helpful.",
		MaxTokens:    100,
		Temperature:  0.7,
		Stream:       false,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify structure
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if parsed["model"] != "llama3.2:latest" {
		t.Errorf("model = %v, want llama3.2:latest", parsed["model"])
	}

	messages := parsed["messages"].([]any)
	if len(messages) != 2 {
		t.Errorf("len(messages) = %d, want 2", len(messages))
	}
}

func TestCompleteResult_Unmarshal(t *testing.T) {
	data := `{
		"content": "Hello, world!",
		"model": "llama3.2:latest",
		"finish_reason": "stop",
		"usage": {
			"input_tokens": 10,
			"output_tokens": 5,
			"total_tokens": 15
		}
	}`

	var result CompleteResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result.Content != "Hello, world!" {
		t.Errorf("Content = %v, want 'Hello, world!'", result.Content)
	}
	if result.Model != "llama3.2:latest" {
		t.Errorf("Model = %v, want llama3.2:latest", result.Model)
	}
	if result.FinishReason != "stop" {
		t.Errorf("FinishReason = %v, want stop", result.FinishReason)
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %v, want 10", result.Usage.InputTokens)
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %v, want 15", result.Usage.TotalTokens)
	}
}

func TestInitParams_Marshal(t *testing.T) {
	params := InitParams{
		Backend: "ollama",
		Model:   "llama3.2:latest",
		Host:    "localhost:11434",
		MCPServers: map[string]MCPServerConfig{
			"test": {Type: "stdio", Command: "cmd", Args: []string{"arg1"}},
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify MCP servers included
	if !strings.Contains(string(data), `"mcp_servers"`) {
		t.Error("Should contain mcp_servers")
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify error code constants
	if CodeParseError != -32700 {
		t.Errorf("CodeParseError = %d, want -32700", CodeParseError)
	}
	if CodeInvalidRequest != -32600 {
		t.Errorf("CodeInvalidRequest = %d, want -32600", CodeInvalidRequest)
	}
	if CodeMethodNotFound != -32601 {
		t.Errorf("CodeMethodNotFound = %d, want -32601", CodeMethodNotFound)
	}
	if CodeInvalidParams != -32602 {
		t.Errorf("CodeInvalidParams = %d, want -32602", CodeInvalidParams)
	}
	if CodeInternalError != -32603 {
		t.Errorf("CodeInternalError = %d, want -32603", CodeInternalError)
	}

	// Application-specific codes
	if CodeBackendError != -32000 {
		t.Errorf("CodeBackendError = %d, want -32000", CodeBackendError)
	}
	if CodeModelNotFound != -32001 {
		t.Errorf("CodeModelNotFound = %d, want -32001", CodeModelNotFound)
	}
}
