package claudecontract

import (
	"encoding/json"
	"testing"
)

// TestEventTypeConstants validates event type constant values match expected CLI output.
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"EventTypeSystem", EventTypeSystem, "system"},
		{"EventTypeAssistant", EventTypeAssistant, "assistant"},
		{"EventTypeUser", EventTypeUser, "user"},
		{"EventTypeResult", EventTypeResult, "result"},
		{"EventTypeStreamEvent", EventTypeStreamEvent, "stream_event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestSubtypeConstants validates all subtype constants.
func TestSubtypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		// System subtypes
		{"SubtypeInit", SubtypeInit, "init"},
		{"SubtypeHookResponse", SubtypeHookResponse, "hook_response"},
		{"SubtypeCompactBoundary", SubtypeCompactBoundary, "compact_boundary"},

		// Result subtypes
		{"ResultSubtypeSuccess", ResultSubtypeSuccess, "success"},
		{"ResultSubtypeErrorMaxTurns", ResultSubtypeErrorMaxTurns, "error_max_turns"},
		{"ResultSubtypeErrorDuringExecution", ResultSubtypeErrorDuringExecution, "error_during_execution"},
		{"ResultSubtypeErrorMaxBudgetUSD", ResultSubtypeErrorMaxBudgetUSD, "error_max_budget_usd"},
		{"ResultSubtypeErrorMaxStructuredOutputRetries", ResultSubtypeErrorMaxStructuredOutputRetries, "error_max_structured_output_retries"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestContentTypeConstants validates content block type constants.
func TestContentTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ContentTypeText", ContentTypeText, "text"},
		{"ContentTypeToolUse", ContentTypeToolUse, "tool_use"},
		{"ContentTypeToolResult", ContentTypeToolResult, "tool_result"},
		{"ContentTypeThinking", ContentTypeThinking, "thinking"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestStopReasonConstants validates stop reason constants.
func TestStopReasonConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"StopReasonEndTurn", StopReasonEndTurn, "end_turn"},
		{"StopReasonMaxTokens", StopReasonMaxTokens, "max_tokens"},
		{"StopReasonStopSequence", StopReasonStopSequence, "stop_sequence"},
		{"StopReasonToolUse", StopReasonToolUse, "tool_use"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestRoleConstants validates message role constants.
func TestRoleConstants(t *testing.T) {
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q, want %q", RoleUser, "user")
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %q, want %q", RoleAssistant, "assistant")
	}
}

// TestAPIKeySourceConstants validates API key source constants.
func TestAPIKeySourceConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"APIKeySourceUser", APIKeySourceUser, "user"},
		{"APIKeySourceProject", APIKeySourceProject, "project"},
		{"APIKeySourceOrg", APIKeySourceOrg, "org"},
		{"APIKeySourceTemporary", APIKeySourceTemporary, "temporary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestMCPStatusConstants validates MCP server status constants.
func TestMCPStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"MCPStatusConnected", MCPStatusConnected, "connected"},
		{"MCPStatusFailed", MCPStatusFailed, "failed"},
		{"MCPStatusNeedsAuth", MCPStatusNeedsAuth, "needs-auth"},
		{"MCPStatusPending", MCPStatusPending, "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// InitEvent represents the expected structure of an init event for testing.
type testInitEvent struct {
	Type              string   `json:"type"`
	Subtype           string   `json:"subtype"`
	SessionID         string   `json:"session_id"`
	CWD               string   `json:"cwd"`
	Model             string   `json:"model"`
	PermissionMode    string   `json:"permission_mode"`
	ClaudeCodeVersion string   `json:"claude_code_version"`
	APIKeySource      string   `json:"api_key_source"`
	Tools             []string `json:"tools"`
	MCPServers        []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers,omitempty"`
}

// TestInitEventStructure validates init event JSON structure matches constants.
func TestInitEventStructure(t *testing.T) {
	// Realistic init event JSON from Claude CLI
	initJSON := `{
		"type": "system",
		"subtype": "init",
		"session_id": "abc-123-def-456",
		"cwd": "/home/user/project",
		"model": "claude-opus-4-5-20251101",
		"permission_mode": "bypassPermissions",
		"claude_code_version": "2.1.19",
		"api_key_source": "user",
		"tools": ["Read", "Write", "Bash"],
		"mcp_servers": [
			{"name": "filesystem", "status": "connected"},
			{"name": "database", "status": "pending"}
		]
	}`

	var event testInitEvent
	if err := json.Unmarshal([]byte(initJSON), &event); err != nil {
		t.Fatalf("Failed to parse init event: %v", err)
	}

	// Validate type and subtype use our constants
	if event.Type != EventTypeSystem {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeSystem)
	}
	if event.Subtype != SubtypeInit {
		t.Errorf("event.Subtype = %q, want %q", event.Subtype, SubtypeInit)
	}

	// Validate API key source uses our constant
	if event.APIKeySource != APIKeySourceUser {
		t.Errorf("event.APIKeySource = %q, want %q", event.APIKeySource, APIKeySourceUser)
	}

	// Validate MCP status values use our constants
	if len(event.MCPServers) < 2 {
		t.Fatal("Expected at least 2 MCP servers")
	}
	if event.MCPServers[0].Status != MCPStatusConnected {
		t.Errorf("mcp_servers[0].status = %q, want %q", event.MCPServers[0].Status, MCPStatusConnected)
	}
	if event.MCPServers[1].Status != MCPStatusPending {
		t.Errorf("mcp_servers[1].status = %q, want %q", event.MCPServers[1].Status, MCPStatusPending)
	}

	// Validate permission mode uses our constant
	if event.PermissionMode != string(PermissionBypassPermissions) {
		t.Errorf("event.PermissionMode = %q, want %q", event.PermissionMode, PermissionBypassPermissions)
	}
}

// testAssistantEvent represents assistant event structure for testing.
type testAssistantEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Message   struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		StopReason *string `json:"stop_reason"`
		Usage      struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
		} `json:"usage"`
	} `json:"message"`
}

// TestAssistantEventWithTextContent validates assistant event with text content.
func TestAssistantEventWithTextContent(t *testing.T) {
	assistantJSON := `{
		"type": "assistant",
		"session_id": "abc-123-def-456",
		"message": {
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-opus-4-5-20251101",
			"content": [
				{"type": "text", "text": "Hello! I'll help you with that."}
			],
			"stop_reason": "end_turn",
			"usage": {
				"input_tokens": 100,
				"output_tokens": 25
			}
		}
	}`

	var event testAssistantEvent
	if err := json.Unmarshal([]byte(assistantJSON), &event); err != nil {
		t.Fatalf("Failed to parse assistant event: %v", err)
	}

	if event.Type != EventTypeAssistant {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeAssistant)
	}
	if event.Message.Role != RoleAssistant {
		t.Errorf("message.role = %q, want %q", event.Message.Role, RoleAssistant)
	}
	if len(event.Message.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(event.Message.Content))
	}
	if event.Message.Content[0].Type != ContentTypeText {
		t.Errorf("content[0].type = %q, want %q", event.Message.Content[0].Type, ContentTypeText)
	}
	if event.Message.StopReason == nil || *event.Message.StopReason != StopReasonEndTurn {
		t.Errorf("stop_reason = %v, want %q", event.Message.StopReason, StopReasonEndTurn)
	}
}

// TestAssistantEventWithToolUse validates assistant event with tool_use content.
func TestAssistantEventWithToolUse(t *testing.T) {
	assistantJSON := `{
		"type": "assistant",
		"session_id": "abc-123-def-456",
		"message": {
			"id": "msg_456",
			"type": "message",
			"role": "assistant",
			"model": "claude-opus-4-5-20251101",
			"content": [
				{"type": "text", "text": "Let me read that file for you."},
				{
					"type": "tool_use",
					"id": "toolu_01ABC",
					"name": "Read",
					"input": {"file_path": "/home/user/test.txt"}
				}
			],
			"stop_reason": "tool_use",
			"usage": {
				"input_tokens": 150,
				"output_tokens": 50,
				"cache_read_input_tokens": 25
			}
		}
	}`

	var event testAssistantEvent
	if err := json.Unmarshal([]byte(assistantJSON), &event); err != nil {
		t.Fatalf("Failed to parse assistant event: %v", err)
	}

	if len(event.Message.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(event.Message.Content))
	}

	// Validate text block
	if event.Message.Content[0].Type != ContentTypeText {
		t.Errorf("content[0].type = %q, want %q", event.Message.Content[0].Type, ContentTypeText)
	}

	// Validate tool_use block
	toolBlock := event.Message.Content[1]
	if toolBlock.Type != ContentTypeToolUse {
		t.Errorf("content[1].type = %q, want %q", toolBlock.Type, ContentTypeToolUse)
	}
	if toolBlock.Name != ToolRead {
		t.Errorf("content[1].name = %q, want %q", toolBlock.Name, ToolRead)
	}
	if toolBlock.ID == "" {
		t.Error("content[1].id should not be empty for tool_use")
	}

	// Validate stop_reason is tool_use when tool is invoked
	if event.Message.StopReason == nil || *event.Message.StopReason != StopReasonToolUse {
		t.Errorf("stop_reason = %v, want %q", event.Message.StopReason, StopReasonToolUse)
	}

	// Validate cache tokens are parsed
	if event.Message.Usage.CacheReadInputTokens != 25 {
		t.Errorf("cache_read_input_tokens = %d, want 25", event.Message.Usage.CacheReadInputTokens)
	}
}

// TestAssistantEventWithThinking validates thinking content block parsing.
func TestAssistantEventWithThinking(t *testing.T) {
	assistantJSON := `{
		"type": "assistant",
		"session_id": "abc-123-def-456",
		"message": {
			"id": "msg_789",
			"type": "message",
			"role": "assistant",
			"model": "claude-opus-4-5-20251101",
			"content": [
				{"type": "thinking", "thinking": "Let me think about this step by step..."},
				{"type": "text", "text": "Here's my answer."}
			],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 200, "output_tokens": 100}
		}
	}`

	var event testAssistantEvent
	if err := json.Unmarshal([]byte(assistantJSON), &event); err != nil {
		t.Fatalf("Failed to parse assistant event: %v", err)
	}

	if len(event.Message.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(event.Message.Content))
	}

	if event.Message.Content[0].Type != ContentTypeThinking {
		t.Errorf("content[0].type = %q, want %q", event.Message.Content[0].Type, ContentTypeThinking)
	}
}

// testResultEvent represents result event structure for testing.
type testResultEvent struct {
	Type             string  `json:"type"`
	Subtype          string  `json:"subtype"`
	SessionID        string  `json:"session_id"`
	IsError          bool    `json:"is_error"`
	Result           string  `json:"result"`
	StructuredOutput any     `json:"structured_output,omitempty"`
	DurationMS       int     `json:"duration_ms"`
	DurationAPIMS    int     `json:"duration_api_ms"`
	NumTurns         int     `json:"num_turns"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
	Usage            struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	} `json:"usage"`
	ModelUsage map[string]struct {
		InputTokens              int     `json:"inputTokens"`
		OutputTokens             int     `json:"outputTokens"`
		CacheReadInputTokens     int     `json:"cacheReadInputTokens,omitempty"`
		CacheCreationInputTokens int     `json:"cacheCreationInputTokens,omitempty"`
		CostUSD                  float64 `json:"costUSD"`
	} `json:"modelUsage,omitempty"`
}

// TestResultEventSuccess validates successful result event structure.
func TestResultEventSuccess(t *testing.T) {
	resultJSON := `{
		"type": "result",
		"subtype": "success",
		"session_id": "abc-123-def-456",
		"is_error": false,
		"result": "Task completed successfully.",
		"duration_ms": 5000,
		"duration_api_ms": 4500,
		"num_turns": 3,
		"total_cost_usd": 0.0123,
		"usage": {
			"input_tokens": 500,
			"output_tokens": 200,
			"cache_creation_input_tokens": 100,
			"cache_read_input_tokens": 50
		},
		"modelUsage": {
			"claude-opus-4-5-20251101": {
				"inputTokens": 500,
				"outputTokens": 200,
				"cacheReadInputTokens": 50,
				"costUSD": 0.0123
			}
		}
	}`

	var event testResultEvent
	if err := json.Unmarshal([]byte(resultJSON), &event); err != nil {
		t.Fatalf("Failed to parse result event: %v", err)
	}

	if event.Type != EventTypeResult {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeResult)
	}
	if event.Subtype != ResultSubtypeSuccess {
		t.Errorf("event.Subtype = %q, want %q", event.Subtype, ResultSubtypeSuccess)
	}
	if event.IsError != false {
		t.Error("is_error should be false for success")
	}

	// Validate usage field naming (snake_case)
	if event.Usage.InputTokens != 500 {
		t.Errorf("usage.input_tokens = %d, want 500", event.Usage.InputTokens)
	}

	// Validate modelUsage field naming (camelCase)
	modelData, ok := event.ModelUsage["claude-opus-4-5-20251101"]
	if !ok {
		t.Fatal("expected modelUsage entry for claude-opus-4-5-20251101")
	}
	if modelData.InputTokens != 500 {
		t.Errorf("modelUsage.inputTokens = %d, want 500", modelData.InputTokens)
	}
}

// TestResultEventErrorSubtypes validates all error subtypes are recognized.
func TestResultEventErrorSubtypes(t *testing.T) {
	errorSubtypes := []string{
		ResultSubtypeErrorMaxTurns,
		ResultSubtypeErrorDuringExecution,
		ResultSubtypeErrorMaxBudgetUSD,
		ResultSubtypeErrorMaxStructuredOutputRetries,
	}

	for _, subtype := range errorSubtypes {
		t.Run(subtype, func(t *testing.T) {
			resultJSON := `{
				"type": "result",
				"subtype": "` + subtype + `",
				"session_id": "abc-123",
				"is_error": true,
				"result": "Error occurred",
				"duration_ms": 1000,
				"num_turns": 1,
				"total_cost_usd": 0.001,
				"usage": {"input_tokens": 100, "output_tokens": 10}
			}`

			var event testResultEvent
			if err := json.Unmarshal([]byte(resultJSON), &event); err != nil {
				t.Fatalf("Failed to parse result event: %v", err)
			}

			if event.Subtype != subtype {
				t.Errorf("event.Subtype = %q, want %q", event.Subtype, subtype)
			}
			if !event.IsError {
				t.Errorf("is_error should be true for %s", subtype)
			}
		})
	}
}

// TestResultEventWithStructuredOutput validates structured output field.
func TestResultEventWithStructuredOutput(t *testing.T) {
	resultJSON := `{
		"type": "result",
		"subtype": "success",
		"session_id": "abc-123",
		"is_error": false,
		"result": "",
		"structured_output": {"name": "John", "age": 30},
		"duration_ms": 2000,
		"num_turns": 1,
		"total_cost_usd": 0.005,
		"usage": {"input_tokens": 200, "output_tokens": 50}
	}`

	var event testResultEvent
	if err := json.Unmarshal([]byte(resultJSON), &event); err != nil {
		t.Fatalf("Failed to parse result event: %v", err)
	}

	if event.StructuredOutput == nil {
		t.Error("structured_output should be populated when --json-schema is used")
	}

	// Validate structure
	structured, ok := event.StructuredOutput.(map[string]any)
	if !ok {
		t.Fatalf("structured_output should be a map, got %T", event.StructuredOutput)
	}
	if structured["name"] != "John" {
		t.Errorf("structured_output.name = %v, want 'John'", structured["name"])
	}
}

// testHookEvent represents hook response event structure for testing.
type testHookEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
}

// TestHookResponseEvent validates hook response event structure.
func TestHookResponseEvent(t *testing.T) {
	hookJSON := `{
		"type": "system",
		"subtype": "hook_response",
		"session_id": "abc-123",
		"hook_name": "SessionStart:startup",
		"hook_event": "SessionStart",
		"stdout": "Hook executed successfully",
		"stderr": "",
		"exit_code": 0
	}`

	var event testHookEvent
	if err := json.Unmarshal([]byte(hookJSON), &event); err != nil {
		t.Fatalf("Failed to parse hook event: %v", err)
	}

	if event.Type != EventTypeSystem {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeSystem)
	}
	if event.Subtype != SubtypeHookResponse {
		t.Errorf("event.Subtype = %q, want %q", event.Subtype, SubtypeHookResponse)
	}
	if event.HookEvent != string(HookSessionStart) {
		t.Errorf("event.HookEvent = %q, want %q", event.HookEvent, HookSessionStart)
	}
}

// TestAllHookEventTypes validates all hook event type constants are valid.
func TestAllHookEventTypes(t *testing.T) {
	hookEvents := ValidHookEvents()
	expectedHooks := []HookEvent{
		HookSessionStart,
		HookUserPromptSubmit,
		HookPreToolUse,
		HookPermissionRequest,
		HookPostToolUse,
		HookPostToolUseFailure,
		HookSubagentStart,
		HookSubagentStop,
		HookStop,
		HookPreCompact,
		HookSessionEnd,
		HookNotification,
	}

	if len(hookEvents) != len(expectedHooks) {
		t.Errorf("ValidHookEvents() returned %d hooks, want %d", len(hookEvents), len(expectedHooks))
	}

	for _, expected := range expectedHooks {
		if !expected.IsValid() {
			t.Errorf("HookEvent %q should be valid", expected)
		}
	}

	// Test invalid hook
	invalid := HookEvent("InvalidHook")
	if invalid.IsValid() {
		t.Error("Invalid hook event should not be valid")
	}
}

// TestHookEventJSONMarshaling validates hook events can be marshaled to JSON.
func TestHookEventJSONMarshaling(t *testing.T) {
	for _, hook := range ValidHookEvents() {
		t.Run(string(hook), func(t *testing.T) {
			// Create a sample event
			event := struct {
				HookEvent HookEvent `json:"hook_event"`
			}{HookEvent: hook}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal hook event: %v", err)
			}

			// Verify it produces valid JSON with the hook name
			expected := `{"hook_event":"` + string(hook) + `"}`
			if string(data) != expected {
				t.Errorf("Marshal = %s, want %s", string(data), expected)
			}
		})
	}
}
