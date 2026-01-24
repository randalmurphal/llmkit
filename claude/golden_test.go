package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Golden file tests validate that our parsing correctly extracts meaningful values
// from real Claude CLI output. These tests verify BEHAVIOR and FAIL when gaps exist.
//
// To update golden files, run real CLI commands and save output:
//   claude --print --output-format stream-json --verbose ... > testdata/golden/xxx.jsonl

// TestGoldenSimpleTextResponse validates we correctly parse a simple text response.
func TestGoldenSimpleTextResponse(t *testing.T) {
	events := loadGoldenEvents(t, "simple_text_response.jsonl")
	require.GreaterOrEqual(t, len(events), 3, "Expected at least init, assistant, result events")

	// === INIT EVENT ===
	initEvent := findEventByType(events, StreamEventInit)
	require.NotNil(t, initEvent, "Expected init event")
	require.NotNil(t, initEvent.Init, "Init field should be populated")

	init := initEvent.Init

	// Session ID - REQUIRED: users need this to resume sessions
	assert.NotEmpty(t, init.SessionID, "Session ID must be extracted - users need it to resume sessions")
	assert.Regexp(t, `^[a-f0-9-]{36}$`, init.SessionID, "Session ID should be a valid UUID")

	// Model - REQUIRED: users need to know which model is being used
	assert.NotEmpty(t, init.Model, "Model must be extracted - users need to know which model responds")

	// CWD - REQUIRED: important for understanding file operations context
	assert.NotEmpty(t, init.CWD, "CWD must be extracted - context for file operations")

	// Tools - REQUIRED: users need to know what tools are available
	assert.NotEmpty(t, init.Tools, "Tools list must be extracted - users need to know available tools")
	assert.Contains(t, init.Tools, "Read", "Expected Read tool to be available")
	assert.Contains(t, init.Tools, "Bash", "Expected Bash tool to be available")

	// Permission mode - REQUIRED: critical for understanding safety constraints
	assert.NotEmpty(t, init.PermissionMode, "Permission mode must be extracted - critical for safety")

	// Claude Code version - REQUIRED: important for compatibility checking
	assert.NotEmpty(t, init.ClaudeCodeVersion, "Claude Code version must be extracted")

	// === ASSISTANT EVENT ===
	assistantEvent := findEventByType(events, StreamEventAssistant)
	require.NotNil(t, assistantEvent, "Expected assistant event")
	require.NotNil(t, assistantEvent.Assistant, "Assistant field should be populated")

	assistant := assistantEvent.Assistant

	// Text content - REQUIRED: the actual response the user sees
	assert.NotEmpty(t, assistant.Text, "Text must be extracted - this is the actual response")
	assert.Contains(t, assistant.Text, "hello", "Response should contain 'hello' as requested")

	// Model should match init
	assert.Equal(t, init.Model, assistant.Model, "Assistant model should match init model")

	// Message ID - REQUIRED: needed for conversation threading
	assert.NotEmpty(t, assistant.MessageID, "Message ID must be extracted")

	// Usage tokens - REQUIRED: users need to track consumption
	totalInput := assistant.Usage.InputTokens + assistant.Usage.CacheCreationInputTokens + assistant.Usage.CacheReadInputTokens
	assert.Greater(t, totalInput, 0, "Input tokens must be tracked")
	assert.Greater(t, assistant.Usage.OutputTokens, 0, "Output tokens must be tracked")

	// === RESULT EVENT ===
	resultEvent := findEventByType(events, StreamEventResult)
	require.NotNil(t, resultEvent, "Expected result event")
	require.NotNil(t, resultEvent.Result, "Result field should be populated")

	result := resultEvent.Result

	// Subtype - REQUIRED: indicates success/error
	assert.Equal(t, "success", result.Subtype, "Subtype should indicate success")
	assert.False(t, result.IsError, "IsError should be false for successful requests")

	// Session ID consistency
	assert.Equal(t, init.SessionID, result.SessionID, "Result session ID should match init")

	// Duration - REQUIRED: performance monitoring
	assert.Greater(t, result.DurationMS, 0, "Duration must be tracked")

	// Cost - REQUIRED: budget tracking
	assert.Greater(t, result.TotalCostUSD, 0.0, "Cost must be tracked for budget management")

	// Turns - REQUIRED: conversation progress
	assert.Greater(t, result.NumTurns, 0, "Number of turns must be tracked")

	// Model usage breakdown
	assert.NotEmpty(t, result.ModelUsage, "ModelUsage breakdown should be present")
	modelDetail, ok := result.ModelUsage[init.Model]
	assert.True(t, ok, "ModelUsage should have entry for the model used: %s", init.Model)
	if ok {
		assert.Greater(t, modelDetail.CostUSD, 0.0, "Per-model cost should be tracked")
	}
}

// TestGoldenToolUseResponse validates we correctly parse tool use interactions.
func TestGoldenToolUseResponse(t *testing.T) {
	events := loadGoldenEvents(t, "tool_use_response.jsonl")
	require.GreaterOrEqual(t, len(events), 3, "Expected at least init, assistant, result events")

	// === ASSISTANT EVENT WITH TOOL USE ===
	assistantEvent := findEventByType(events, StreamEventAssistant)
	require.NotNil(t, assistantEvent, "Expected assistant event")
	require.NotNil(t, assistantEvent.Assistant, "Assistant field should be populated")

	assistant := assistantEvent.Assistant
	require.NotEmpty(t, assistant.Content, "Content blocks must be present")

	// Find tool_use block
	var toolUseBlock *ContentBlock
	for i := range assistant.Content {
		if assistant.Content[i].Type == "tool_use" {
			toolUseBlock = &assistant.Content[i]
			break
		}
	}
	require.NotNil(t, toolUseBlock, "Expected tool_use content block - CRITICAL for agentic workflows")

	// Tool name - REQUIRED: users need to know which tool was called
	assert.Equal(t, "Read", toolUseBlock.Name, "Tool name must be extracted correctly")

	// Tool use ID - REQUIRED: needed to match with tool result
	assert.NotEmpty(t, toolUseBlock.ID, "Tool use ID must be extracted - needed to correlate with results")
	assert.True(t, strings.HasPrefix(toolUseBlock.ID, "toolu_"), "Tool use ID should have expected prefix")

	// Tool input - REQUIRED: users may need to inspect what was requested
	assert.NotEmpty(t, toolUseBlock.Input, "Tool input must be extracted")
	var input map[string]any
	err := json.Unmarshal(toolUseBlock.Input, &input)
	require.NoError(t, err, "Tool input should be valid JSON")
	assert.Equal(t, "/etc/passwd", input["file_path"], "Tool input should contain file_path")

	// === RESULT EVENT ===
	resultEvent := findEventByType(events, StreamEventResult)
	require.NotNil(t, resultEvent, "Expected result event")

	result := resultEvent.Result
	// This test uses max_turns=1, so it hits the limit
	assert.Equal(t, "error_max_turns", result.Subtype, "Should detect error_max_turns subtype")
	assert.False(t, result.IsError, "error_max_turns is not a failure (request completed)")
	assert.Equal(t, 2, result.NumTurns, "Should track 2 turns (request + tool use)")
}

// TestGoldenUserEventMustBeParsed validates that user events (tool results) are parsed.
// This test FAILS if user events are not properly handled - they contain critical tool result data.
func TestGoldenUserEventMustBeParsed(t *testing.T) {
	events := loadGoldenEvents(t, "tool_use_response.jsonl")

	// The tool_use_response.jsonl contains a user event with tool result
	// Our parseStreamEvent MUST handle this - it's critical for agentic workflows

	// Check if any event has the user type properly parsed
	var userEventFound bool
	var userEventRaw json.RawMessage

	for _, event := range events {
		// Check if we parsed a user event
		// Currently parseStreamEvent doesn't handle "user" type, so Type will be empty
		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(event.Raw, &base); err == nil && base.Type == "user" {
			userEventRaw = event.Raw
			userEventFound = true
			break
		}
	}

	require.True(t, userEventFound, "Golden file must contain a user event")

	// Now verify that our parseStreamEvent actually handles it
	// The event.Type should be set to something meaningful, not empty
	parsedEvent, err := parseStreamEvent(userEventRaw)
	require.NoError(t, err, "parseStreamEvent should not error on user event")

	// THIS IS THE KEY TEST: Does parseStreamEvent set a proper Type for user events?
	// If this fails, it means we're silently dropping user events!
	if parsedEvent.Type == "" {
		t.Fatal("CRITICAL: User events are not being parsed! " +
			"parseStreamEvent returns empty Type for 'user' events. " +
			"User events contain tool results which are essential for agentic workflows. " +
			"Add handling for EventTypeUser in parseStreamEvent.")
	}

	// If we get here, verify the type is correct
	// Note: We need to add StreamEventUser constant and handling
	t.Logf("User event Type: %v", parsedEvent.Type)
}

// TestGoldenToolResultCorrelation validates we can correlate tool_use with tool_result.
// This is CRITICAL for agentic workflows - you must match tool calls with their results.
func TestGoldenToolResultCorrelation(t *testing.T) {
	events := loadGoldenEvents(t, "tool_use_response.jsonl")

	// Extract tool_use ID from assistant event
	var toolUseID string
	for _, event := range events {
		if event.Type == StreamEventAssistant && event.Assistant != nil {
			for _, block := range event.Assistant.Content {
				if block.Type == "tool_use" {
					toolUseID = block.ID
					break
				}
			}
		}
	}
	require.NotEmpty(t, toolUseID, "Must extract tool_use ID from assistant event")

	// Find the user event and extract tool_result's tool_use_id
	// This uses our parsed UserEvent struct, not raw JSON
	var toolResultID string
	for _, event := range events {
		if event.Type == StreamEventUser && event.User != nil {
			for _, content := range event.User.Message.Content {
				if content.Type == "tool_result" {
					toolResultID = content.ToolUseID
					break
				}
			}
		}
	}

	require.NotEmpty(t, toolResultID, "Must extract tool_result's tool_use_id from UserEvent")
	assert.Equal(t, toolUseID, toolResultID,
		"Tool result's tool_use_id must match the tool_use ID for correlation")
}

// TestGoldenUserEventFields validates we correctly parse all user event fields.
func TestGoldenUserEventFields(t *testing.T) {
	events := loadGoldenEvents(t, "tool_use_response.jsonl")

	// Find user event
	var userEvent *StreamEvent
	for _, event := range events {
		if event.Type == StreamEventUser {
			userEvent = event
			break
		}
	}
	require.NotNil(t, userEvent, "Expected user event in tool_use_response")
	require.NotNil(t, userEvent.User, "User field must be populated")

	user := userEvent.User

	// Session ID - REQUIRED for correlation
	assert.NotEmpty(t, user.SessionID, "Session ID must be extracted")

	// Message - REQUIRED for tool result content
	assert.Equal(t, "user", user.Message.Role, "Message role must be 'user'")
	require.NotEmpty(t, user.Message.Content, "Message content must be present")

	// Tool result content
	content := user.Message.Content[0]
	assert.Equal(t, "tool_result", content.Type, "Content type must be 'tool_result'")
	assert.NotEmpty(t, content.ToolUseID, "ToolUseID must be extracted - needed for correlation")
	assert.NotEmpty(t, content.GetContent(), "Content text must be extracted")

	// ToolUseResult - structured tool output
	toolResult := user.GetToolUseResult()
	require.NotNil(t, toolResult, "ToolUseResult should be present for Read tool")
	assert.Equal(t, "text", toolResult.Type, "ToolUseResult type should be 'text'")

	// File result - specific to Read tool
	require.NotNil(t, toolResult.File, "File result should be present")
	assert.Equal(t, "/etc/passwd", toolResult.File.FilePath, "FilePath must match")
	assert.Contains(t, toolResult.File.Content, "root:", "File content should be present")
	assert.Equal(t, 5, toolResult.File.NumLines, "NumLines must be extracted")
}

// TestGoldenAllEventTypesParsed validates that ALL event types in the golden file are parsed.
// This test FAILS if any event type is silently dropped.
func TestGoldenAllEventTypesParsed(t *testing.T) {
	testFiles := []string{"simple_text_response.jsonl", "tool_use_response.jsonl"}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			events := loadGoldenEvents(t, filename)

			for i, event := range events {
				// Get the raw type
				var base struct {
					Type string `json:"type"`
				}
				err := json.Unmarshal(event.Raw, &base)
				require.NoError(t, err, "Event %d should have parseable type", i)

				// Verify parseStreamEvent assigned a Type
				if event.Type == "" && base.Type != "" {
					t.Errorf("Event %d: Raw type is %q but parsed Type is empty - event is being dropped!", i, base.Type)
				}
			}
		})
	}
}

// TestGoldenInitFieldCompleteness validates we extract ALL important fields from init.
// This test FAILS if required fields are missing from our InitEvent struct.
func TestGoldenInitFieldCompleteness(t *testing.T) {
	events := loadGoldenEvents(t, "simple_text_response.jsonl")
	initEvent := findEventByType(events, StreamEventInit)
	require.NotNil(t, initEvent, "Expected init event")

	// Parse raw to see all available fields
	var rawInit map[string]any
	err := json.Unmarshal(initEvent.Raw, &rawInit)
	require.NoError(t, err)

	// These fields are REQUIRED - test fails if we don't extract them
	requiredFields := map[string]func() bool{
		"session_id":          func() bool { return initEvent.Init.SessionID != "" },
		"model":               func() bool { return initEvent.Init.Model != "" },
		"cwd":                 func() bool { return initEvent.Init.CWD != "" },
		"tools":               func() bool { return len(initEvent.Init.Tools) > 0 },
		"claude_code_version": func() bool { return initEvent.Init.ClaudeCodeVersion != "" },
		"permissionMode":      func() bool { return initEvent.Init.PermissionMode != "" },
	}

	for field, check := range requiredFields {
		// Verify field exists in raw output
		_, inRaw := rawInit[field]
		require.True(t, inRaw, "Required field %q must be in CLI output", field)

		// Verify we extract it
		assert.True(t, check(), "Required field %q must be extracted into InitEvent struct", field)
	}

	// These fields SHOULD be extracted - log if missing but don't fail (yet)
	// TODO: Add these to InitEvent struct and change to assertions
	optionalButUseful := []string{
		"slash_commands", // Available slash commands
		"agents",         // Available agents for delegation
		"skills",         // Available skills
		"plugins",        // Loaded plugins
		"apiKeySource",   // Debugging auth issues
		"uuid",           // Event identifier
	}

	for _, field := range optionalButUseful {
		if _, inRaw := rawInit[field]; inRaw {
			t.Logf("TODO: Field %q is in CLI output but not extracted", field)
		}
	}
}

// TestGoldenResultFieldCompleteness validates we extract ALL important fields from result.
func TestGoldenResultFieldCompleteness(t *testing.T) {
	events := loadGoldenEvents(t, "simple_text_response.jsonl")
	resultEvent := findEventByType(events, StreamEventResult)
	require.NotNil(t, resultEvent, "Expected result event")

	// Parse raw to see all available fields
	var rawResult map[string]any
	err := json.Unmarshal(resultEvent.Raw, &rawResult)
	require.NoError(t, err)

	result := resultEvent.Result

	// Required fields - test FAILS if not extracted
	assert.NotEmpty(t, result.Subtype, "subtype must be extracted")
	assert.NotEmpty(t, result.SessionID, "session_id must be extracted")
	assert.Greater(t, result.DurationMS, 0, "duration_ms must be extracted")
	assert.Greater(t, result.TotalCostUSD, 0.0, "total_cost_usd must be extracted")
	assert.Greater(t, result.NumTurns, 0, "num_turns must be extracted")
	assert.NotEmpty(t, result.ModelUsage, "modelUsage must be extracted")

	// Verify usage struct is populated
	totalTokens := result.Usage.InputTokens + result.Usage.OutputTokens +
		result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens
	assert.Greater(t, totalTokens, 0, "usage token counts must be extracted")
}

// TestGoldenAssistantFieldCompleteness validates we extract ALL important fields from assistant.
func TestGoldenAssistantFieldCompleteness(t *testing.T) {
	events := loadGoldenEvents(t, "simple_text_response.jsonl")
	assistantEvent := findEventByType(events, StreamEventAssistant)
	require.NotNil(t, assistantEvent, "Expected assistant event")

	// Parse raw to see all available fields
	var rawAssistant map[string]any
	err := json.Unmarshal(assistantEvent.Raw, &rawAssistant)
	require.NoError(t, err)

	assistant := assistantEvent.Assistant

	// Required fields - test FAILS if not extracted
	assert.NotEmpty(t, assistant.MessageID, "message.id must be extracted")
	assert.NotEmpty(t, assistant.Model, "message.model must be extracted")
	assert.NotEmpty(t, assistant.Content, "message.content must be extracted")

	// Usage must be extracted
	totalTokens := assistant.Usage.InputTokens + assistant.Usage.OutputTokens +
		assistant.Usage.CacheCreationInputTokens + assistant.Usage.CacheReadInputTokens
	assert.Greater(t, totalTokens, 0, "message.usage must be extracted")
}

// TestGoldenSessionIDConsistency validates session IDs match across all events.
func TestGoldenSessionIDConsistency(t *testing.T) {
	for _, filename := range []string{"simple_text_response.jsonl", "tool_use_response.jsonl"} {
		t.Run(filename, func(t *testing.T) {
			events := loadGoldenEvents(t, filename)

			var expectedSessionID string
			for _, event := range events {
				if event.SessionID != "" {
					if expectedSessionID == "" {
						expectedSessionID = event.SessionID
					} else {
						assert.Equal(t, expectedSessionID, event.SessionID,
							"All events must have consistent session ID")
					}
				}
			}
			require.NotEmpty(t, expectedSessionID, "Must extract session ID from events")
		})
	}
}

// === Helper functions ===

func loadGoldenEvents(t *testing.T, filename string) []*StreamEvent {
	t.Helper()

	goldenPath := filepath.Join("testdata", "golden", filename)
	f, err := os.Open(goldenPath)
	if os.IsNotExist(err) {
		t.Skipf("Golden file not found: %s - run CLI to generate it", goldenPath)
	}
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	var events []*StreamEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// IMPORTANT: Copy the bytes because scanner reuses its buffer
		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)

		event, err := parseStreamEvent(lineCopy)
		require.NoError(t, err, "Failed to parse line: %s", string(lineCopy))
		events = append(events, event)
	}

	require.NoError(t, scanner.Err())
	require.NotEmpty(t, events, "Golden file should contain events")

	return events
}

func findEventByType(events []*StreamEvent, eventType StreamEventType) *StreamEvent {
	for _, e := range events {
		if e.Type == eventType {
			return e
		}
	}
	return nil
}
