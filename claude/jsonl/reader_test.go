package jsonl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/claude/session"
)

// Sample JSONL content matching Claude Code's format
const sampleJSONL = `{"type":"user","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"test-session-123","uuid":"msg-1","message":{"role":"user","content":[{"type":"text","text":"Hello"}]}}
{"type":"assistant","timestamp":"2024-01-19T10:00:01.000Z","sessionId":"test-session-123","uuid":"msg-2","parentUuid":"msg-1","message":{"role":"assistant","content":[{"type":"text","text":"Hi there!"}],"model":"claude-opus-4-5-20251101","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":1000,"cache_read_input_tokens":500}}}
{"type":"assistant","timestamp":"2024-01-19T10:00:02.000Z","sessionId":"test-session-123","uuid":"msg-3","message":{"role":"assistant","content":[{"type":"tool_use","id":"tool-1","name":"Read","input":{"file_path":"/test.txt"}}],"model":"claude-opus-4-5-20251101","usage":{"input_tokens":150,"output_tokens":25}}}
`

func TestReader_ReadAll(t *testing.T) {
	// Create temp file with sample JSONL
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(sampleJSONL), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewReader(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	messages, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	// Check first message (user)
	if !messages[0].IsUser() {
		t.Error("first message should be user")
	}
	if messages[0].UUID != "msg-1" {
		t.Errorf("expected UUID msg-1, got %s", messages[0].UUID)
	}

	// Check second message (assistant)
	if !messages[1].IsAssistant() {
		t.Error("second message should be assistant")
	}
	if messages[1].GetModel() != "claude-opus-4-5-20251101" {
		t.Errorf("expected model claude-opus-4-5-20251101, got %s", messages[1].GetModel())
	}
	usage := messages[1].GetUsage()
	if usage == nil {
		t.Fatal("expected usage info")
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

	// Check third message has tool call
	tools := messages[2].GetToolCalls()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(tools))
	}
	if tools[0].Name != "Read" {
		t.Errorf("expected tool name Read, got %s", tools[0].Name)
	}
}

func TestReader_ReadFrom(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(sampleJSONL), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewReader(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Read first message, get offset
	messages, offset, err := r.ReadFrom(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
	if offset == 0 {
		t.Error("offset should be > 0 after reading")
	}

	// Reading from end should return nothing
	messages, newOffset, err := r.ReadFrom(offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages from end, got %d", len(messages))
	}
	if newOffset != offset {
		t.Error("offset should not change when nothing to read")
	}
}

func TestReader_Tail(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")

	// Start with empty file
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewReader(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch := r.Tail(ctx)

	// Wait a bit for tail to start
	time.Sleep(50 * time.Millisecond)

	// Append a message
	f, err := os.OpenFile(jsonlPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"test","uuid":"msg-1","message":{"role":"user","content":[{"type":"text","text":"Hello"}]}}` + "\n"
	if _, err := f.WriteString(line); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Should receive the message
	select {
	case msg := <-ch:
		if msg.UUID != "msg-1" {
			t.Errorf("expected UUID msg-1, got %s", msg.UUID)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for tailed message")
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(sampleJSONL), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadFile(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
}

func TestReadFile_NotFound(t *testing.T) {
	_, err := ReadFile("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSummarize(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(sampleJSONL), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := Summarize(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}

	if summary.SessionID != "test-session-123" {
		t.Errorf("expected session ID test-session-123, got %s", summary.SessionID)
	}
	if summary.MessageCount != 3 {
		t.Errorf("expected 3 messages, got %d", summary.MessageCount)
	}
	if summary.UserMessages != 1 {
		t.Errorf("expected 1 user message, got %d", summary.UserMessages)
	}
	if summary.AssistantMessages != 2 {
		t.Errorf("expected 2 assistant messages, got %d", summary.AssistantMessages)
	}
	if summary.TotalInputTokens != 250 {
		t.Errorf("expected 250 total input tokens, got %d", summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != 75 {
		t.Errorf("expected 75 total output tokens, got %d", summary.TotalOutputTokens)
	}
	if summary.TotalCacheCreationTokens != 1000 {
		t.Errorf("expected 1000 cache creation tokens, got %d", summary.TotalCacheCreationTokens)
	}
	if summary.TotalCacheReadTokens != 500 {
		t.Errorf("expected 500 cache read tokens, got %d", summary.TotalCacheReadTokens)
	}
	if summary.ToolCalls != 1 {
		t.Errorf("expected 1 tool call, got %d", summary.ToolCalls)
	}
	if count, ok := summary.Models["claude-opus-4-5-20251101"]; !ok || count != 2 {
		t.Errorf("expected 2 messages for claude-opus-4-5-20251101, got %v", summary.Models)
	}
}

func TestExtractToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(sampleJSONL), 0644); err != nil {
		t.Fatal(err)
	}

	tools, err := ExtractToolCalls(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(tools))
	}
	if tools[0].Name != "Read" {
		t.Errorf("expected tool name Read, got %s", tools[0].Name)
	}
}

func TestFilterByModel(t *testing.T) {
	messages := []session.JSONLMessage{
		{Type: "assistant", Message: &session.JSONLMessageBody{Model: "claude-opus-4-5-20251101"}},
		{Type: "assistant", Message: &session.JSONLMessageBody{Model: "claude-sonnet-4-20250514"}},
		{Type: "assistant", Message: &session.JSONLMessageBody{Model: "claude-opus-4-5-20251101"}},
	}

	filtered := FilterByModel(messages, "claude-opus-4-5-20251101")
	if len(filtered) != 2 {
		t.Errorf("expected 2 opus messages, got %d", len(filtered))
	}
}

const sampleJSONLWithTodos = `{"type":"assistant","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"test","uuid":"msg-1","message":{"role":"assistant","content":[]},"toolUseResult":{"oldTodos":[],"newTodos":[{"content":"Write code","status":"pending","activeForm":"Writing code"},{"content":"Run tests","status":"pending","activeForm":"Running tests"}]}}
{"type":"assistant","timestamp":"2024-01-19T10:00:01.000Z","sessionId":"test","uuid":"msg-2","message":{"role":"assistant","content":[]},"toolUseResult":{"oldTodos":[{"content":"Write code","status":"pending","activeForm":"Writing code"},{"content":"Run tests","status":"pending","activeForm":"Running tests"}],"newTodos":[{"content":"Write code","status":"completed","activeForm":"Writing code"},{"content":"Run tests","status":"in_progress","activeForm":"Running tests"}]}}
`

func TestExtractTodos(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(sampleJSONLWithTodos), 0644); err != nil {
		t.Fatal(err)
	}

	todos, err := ExtractTodos(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(todos) != 2 {
		t.Fatalf("expected 2 todo snapshots, got %d", len(todos))
	}

	// First snapshot: 2 pending
	if len(todos[0]) != 2 {
		t.Errorf("expected 2 todos in first snapshot, got %d", len(todos[0]))
	}
	if todos[0][0].Status != "pending" {
		t.Errorf("expected first todo pending, got %s", todos[0][0].Status)
	}

	// Second snapshot: 1 completed, 1 in_progress
	if len(todos[1]) != 2 {
		t.Errorf("expected 2 todos in second snapshot, got %d", len(todos[1]))
	}
	if todos[1][0].Status != "completed" {
		t.Errorf("expected first todo completed, got %s", todos[1][0].Status)
	}
	if todos[1][1].Status != "in_progress" {
		t.Errorf("expected second todo in_progress, got %s", todos[1][1].Status)
	}
}

func TestReadAll_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("ReadFile on empty file failed: %v", err)
	}
	// nil and empty slice are equivalent in Go - just check length
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestReadAll_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "malformed.jsonl")

	// Mix of valid and malformed JSON lines
	content := `{"type":"user","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"test","uuid":"msg-1","message":{"role":"user","content":[{"type":"text","text":"Hello"}]}}
not valid json at all
{"also invalid
{"type":"assistant","timestamp":"2024-01-19T10:00:01.000Z","sessionId":"test","uuid":"msg-2","message":{"role":"assistant","content":[{"type":"text","text":"Hi!"}]}}
`
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Should get only the 2 valid messages, malformed lines silently skipped
	if len(messages) != 2 {
		t.Errorf("expected 2 valid messages (malformed skipped), got %d", len(messages))
	}
	if messages[0].UUID != "msg-1" {
		t.Errorf("expected first message UUID 'msg-1', got %q", messages[0].UUID)
	}
	if messages[1].UUID != "msg-2" {
		t.Errorf("expected second message UUID 'msg-2', got %q", messages[1].UUID)
	}
}

func TestReadAll_TruncatedLine(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "truncated.jsonl")

	// File with no trailing newline (simulates active write)
	content := `{"type":"user","timestamp":"2024-01-19T10:00:00.000Z","sessionId":"test","uuid":"msg-1","message":{"role":"user","content":[{"type":"text","text":"Hello"}]}}
{"type":"assistant","timestamp":"2024-01-19T10:00:01.000Z","sessionId":"test","uuid":"msg-2","message":{"role":"assistant","content":[{"type":"text","text":"Hi!"}]}}`
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	messages, err := ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Both lines should be parsed even without trailing newline
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestToJSON(t *testing.T) {
	messages := []session.JSONLMessage{
		{Type: "user", UUID: "msg-1", SessionID: "sess-1"},
		{Type: "assistant", UUID: "msg-2", SessionID: "sess-1"},
	}

	data, err := ToJSON(messages)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Should produce valid JSON array
	if data[0] != '[' {
		t.Error("expected JSON array starting with '['")
	}

	// Verify it can be parsed back
	var parsed []session.JSONLMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal ToJSON output: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 messages after round-trip, got %d", len(parsed))
	}
}

func TestReader_Path(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewReader(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if r.Path() != jsonlPath {
		t.Errorf("expected path %q, got %q", jsonlPath, r.Path())
	}
}

func TestFindSessionFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	projectDir := filepath.Join(tmpDir, "-home-user-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some JSONL files
	files := []string{
		filepath.Join(projectDir, "session1.jsonl"),
		filepath.Join(projectDir, "session2.jsonl"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Also create a non-JSONL file
	if err := os.WriteFile(filepath.Join(projectDir, "other.txt"), []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := FindSessionFiles(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(found) != 2 {
		t.Errorf("expected 2 JSONL files, got %d", len(found))
	}
}
