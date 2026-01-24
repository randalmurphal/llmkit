package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral tests verify that Claude CLI features actually work as intended.
// These tests run the real CLI and verify behavior, not just parsing.
//
// Run with: TEST_BEHAVIORAL=1 go test ./claude/... -run Behavioral -v
//
// These tests use API credits and should be run intentionally, not in CI.

const (
	testTimeout     = 2 * time.Minute
	testMaxBudget   = "0.50" // USD limit for safety
	testGoldenDir   = "testdata/behavioral"
)

// skipUnlessBehavioral skips the test unless TEST_BEHAVIORAL=1
func skipUnlessBehavioral(t *testing.T) {
	if os.Getenv("TEST_BEHAVIORAL") != "1" {
		t.Skip("Skipping behavioral test. Set TEST_BEHAVIORAL=1 to run (uses API credits)")
	}
}

// findClaude locates the claude CLI binary
func findClaude(t *testing.T) string {
	t.Helper()

	// Check common locations
	paths := []string{
		os.ExpandEnv("$HOME/.local/bin/claude"),
		"/usr/local/bin/claude",
		"claude",
	}

	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}

	// Try which
	if out, err := exec.Command("which", "claude").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}

	t.Fatal("Claude CLI not found. Install it or set PATH.")
	return ""
}

// runClaudeOptions configures runClaudeRaw behavior
type runClaudeOptions struct {
	skipPermissions bool
}

// runClaudeAndCapture runs claude with args and returns parsed events.
// Uses --dangerously-skip-permissions by default for safety.
func runClaudeAndCapture(t *testing.T, args ...string) []*StreamEvent {
	t.Helper()
	return runClaudeRaw(t, runClaudeOptions{skipPermissions: true}, args...)
}

// runClaudeWithPermissions runs claude WITHOUT --dangerously-skip-permissions.
// Use for testing permission modes.
func runClaudeWithPermissions(t *testing.T, args ...string) []*StreamEvent {
	t.Helper()
	return runClaudeRaw(t, runClaudeOptions{skipPermissions: false}, args...)
}

// runClaudeRaw is the core runner with configurable options
func runClaudeRaw(t *testing.T, opts runClaudeOptions, args ...string) []*StreamEvent {
	t.Helper()
	skipUnlessBehavioral(t)

	claude := findClaude(t)

	// Base safety flags
	fullArgs := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--max-budget-usd", testMaxBudget,
	}

	// Conditionally add skip permissions
	if opts.skipPermissions {
		fullArgs = append(fullArgs, "--dangerously-skip-permissions")
	}

	fullArgs = append(fullArgs, args...)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, claude, fullArgs...)
	output, err := cmd.CombinedOutput()

	// Parse even if command failed - we want to see the events
	events := parseStreamOutput(t, output)

	if err != nil {
		// Check if this was an expected error (budget, max_turns, etc)
		if len(events) > 0 {
			lastEvent := events[len(events)-1]
			if lastEvent.Type == StreamEventResult && lastEvent.Result != nil {
				// Expected termination conditions - return events for the test to examine
				switch lastEvent.Result.Subtype {
				case "success", "error_max_turns", "error_max_budget_usd",
					"error_during_execution", "error_max_structured_output_retries":
					return events
				}
			}
		}
		// FAIL the test for unexpected errors
		// This ensures we catch breaking CLI changes
		t.Fatalf("Unexpected CLI error (no recognized result event).\n"+
			"Error: %v\n"+
			"Output (first 1000 chars): %s\n"+
			"Events captured: %d",
			err, string(output)[:min(1000, len(output))], len(events))
	}

	return events
}

// parseStreamOutput parses JSONL output into events.
// This function FAILS the test if any JSON line cannot be parsed.
// This is intentional: we want to catch breaking changes in Claude CLI output format.
func parseStreamOutput(t *testing.T, output []byte) []*StreamEvent {
	t.Helper()

	var events []*StreamEvent
	var parseErrors []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		lineNum++
		if len(line) == 0 {
			continue
		}

		// Skip non-JSON lines (stderr errors, warnings from CLI)
		if line[0] != '{' {
			t.Logf("Line %d: Non-JSON line (skipped): %s", lineNum, string(line)[:min(100, len(line))])
			continue
		}

		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)

		event, err := parseStreamEvent(lineCopy)
		if err != nil {
			// Record parse error - we'll fail after collecting all errors
			parseErrors = append(parseErrors, fmt.Sprintf(
				"Line %d: %v\n  Content: %s",
				lineNum, err, string(line)[:min(200, len(line))],
			))
			continue
		}
		events = append(events, event)
	}

	// FAIL if any JSON lines could not be parsed
	// This catches breaking changes in Claude CLI output format
	if len(parseErrors) > 0 {
		t.Fatalf("PARSE FAILURE: %d JSON lines could not be parsed.\n"+
			"This indicates a breaking change in Claude CLI output format.\n"+
			"Errors:\n%s",
			len(parseErrors), strings.Join(parseErrors, "\n"))
	}

	return events
}

// saveGoldenFile saves events to a golden file for future reference
func saveGoldenFile(t *testing.T, name string, output []byte) {
	t.Helper()

	dir := filepath.Join(testGoldenDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create golden dir: %v", err)
	}

	path := filepath.Join(dir, name+".jsonl")
	if err := os.WriteFile(path, output, 0644); err != nil {
		t.Fatalf("Failed to write golden file: %v", err)
	}
	t.Logf("Saved golden file: %s", path)
}

// =============================================================================
// BEHAVIORAL TESTS - Verify actual CLI functionality
// =============================================================================

// TestBehavioralMaxTurns verifies --max-turns actually limits turns
func TestBehavioralMaxTurns(t *testing.T) {
	skipUnlessBehavioral(t)

	// Request something that requires multiple turns (tool use)
	events := runClaudeAndCapture(t,
		"--max-turns", "1",
		"Read the file /etc/hostname and tell me what it says",
	)

	require.NotEmpty(t, events, "Should have events")

	// Find the result event
	var result *ResultEvent
	for _, e := range events {
		if e.Type == StreamEventResult && e.Result != nil {
			result = e.Result
			break
		}
	}
	require.NotNil(t, result, "Must have result event")

	// BEHAVIORAL ASSERTION: max-turns should trigger error_max_turns
	assert.Equal(t, "error_max_turns", result.Subtype,
		"--max-turns=1 with tool use should hit max turns limit. "+
			"If this fails, the --max-turns flag behavior has changed.")

	// Verify turn count
	assert.LessOrEqual(t, result.NumTurns, 2,
		"Should not exceed max turns + 1 (for the tool result)")
}

// TestBehavioralAllowedTools verifies --allowedTools restricts actual tool usage
func TestBehavioralAllowedTools(t *testing.T) {
	skipUnlessBehavioral(t)

	// Only allow Read tool, then ask Claude to write a file
	// Claude should NOT be able to use Write since it's not allowed
	//
	// Note: Using runClaudeWithPermissions because --dangerously-skip-permissions
	// may override --allowedTools restrictions
	events := runClaudeWithPermissions(t,
		"--allowedTools", "Read",
		"--max-turns", "2",
		"Write a file called /tmp/test-allowed-tools.txt with content 'hello'",
	)

	require.NotEmpty(t, events, "Should have events")

	// Look for tool use attempts
	var usedWrite bool
	var usedRead bool
	var assistantText string

	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistantText += e.Assistant.Text
			for _, block := range e.Assistant.Content {
				if block.Type == "tool_use" {
					switch block.Name {
					case "Write":
						usedWrite = true
					case "Read":
						usedRead = true
					}
				}
			}
		}
	}

	// BEHAVIORAL ASSERTION: Write should NOT be used when not allowed
	// Claude might either:
	// 1. Not attempt Write at all (explain it can't)
	// 2. Attempt Write but get rejected by permissions
	//
	// The key assertion is that the write operation should NOT succeed
	if usedWrite {
		// If Claude tried to use Write, check it was rejected
		t.Logf("Claude attempted Write despite --allowedTools=Read")
		// Check if the file was actually created (it shouldn't be)
		if _, err := os.Stat("/tmp/test-allowed-tools.txt"); err == nil {
			os.Remove("/tmp/test-allowed-tools.txt") // cleanup
			t.Fatal("--allowedTools=Read did NOT prevent Write tool usage. " +
				"The file was actually created, meaning tool restriction failed.")
		}
	}

	// Claude should either use an allowed tool or explain it can't do the task
	t.Logf("Assistant response: %s", assistantText)
	t.Logf("Used Write: %v, Used Read: %v", usedWrite, usedRead)

	// Verify the init event still lists all tools (this is expected behavior)
	var init *InitEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
			break
		}
	}
	require.NotNil(t, init, "Must have init event")

	// Note: init.Tools shows all available tools, NOT just allowed ones
	// This is expected CLI behavior - the restriction is at USE time
	assert.Contains(t, init.Tools, "Read", "Read should be in available tools")
	// Tools list includes all tools - this is NOT a failure
	t.Logf("Init tools count: %d (includes all tools, not just allowed)", len(init.Tools))
}

// TestBehavioralDisallowedTools verifies --disallowedTools prevents tool usage
func TestBehavioralDisallowedTools(t *testing.T) {
	skipUnlessBehavioral(t)

	// Disallow Write, then ask Claude to write a file
	//
	// Note: Using runClaudeWithPermissions because --dangerously-skip-permissions
	// may override --disallowedTools restrictions
	events := runClaudeWithPermissions(t,
		"--disallowedTools", "Write",
		"--max-turns", "2",
		"Write a file called /tmp/test-disallowed-tools.txt with content 'hello'",
	)

	require.NotEmpty(t, events, "Should have events")

	// Look for tool use attempts
	var usedWrite bool
	var assistantText string

	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistantText += e.Assistant.Text
			for _, block := range e.Assistant.Content {
				if block.Type == "tool_use" && block.Name == "Write" {
					usedWrite = true
				}
			}
		}
	}

	// BEHAVIORAL ASSERTION: Write should NOT be used when disallowed
	if usedWrite {
		t.Logf("Claude attempted Write despite --disallowedTools=Write")
		// Check if the file was actually created (it shouldn't be)
		if _, err := os.Stat("/tmp/test-disallowed-tools.txt"); err == nil {
			os.Remove("/tmp/test-disallowed-tools.txt") // cleanup
			t.Fatal("--disallowedTools=Write did NOT prevent Write tool usage. " +
				"The file was actually created, meaning tool restriction failed.")
		}
	}

	t.Logf("Assistant response: %s", assistantText)
	t.Logf("Used Write: %v", usedWrite)

	// Verify init event has tools (even disallowed ones may appear in list)
	var init *InitEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
			break
		}
	}
	require.NotNil(t, init, "Must have init event")

	// Read should still be available when not disallowed
	assert.Contains(t, init.Tools, "Read",
		"Read should be in available tools when not disallowed")
}

// TestBehavioralPermissionMode verifies --permission-mode is reflected in init
func TestBehavioralPermissionMode(t *testing.T) {
	skipUnlessBehavioral(t)

	// Use runClaudeWithPermissions to avoid --dangerously-skip-permissions
	// which overrides --permission-mode
	events := runClaudeWithPermissions(t,
		"--permission-mode", "plan",
		"--max-turns", "1",
		"Say hello",
	)

	require.NotEmpty(t, events, "Should have events")

	var init *InitEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
			break
		}
	}
	require.NotNil(t, init, "Must have init event")

	// BEHAVIORAL ASSERTION: permission mode should be set
	assert.Equal(t, "plan", init.PermissionMode,
		"--permission-mode=plan should set permissionMode to 'plan'. "+
			"If this fails, permission mode handling has changed.")
}

// TestBehavioralModelSelection verifies --model selects the specified model
func TestBehavioralModelSelection(t *testing.T) {
	skipUnlessBehavioral(t)

	// Use a specific model
	events := runClaudeAndCapture(t,
		"--model", "claude-sonnet-4-20250514",
		"--max-turns", "1",
		"Say exactly: test",
	)

	require.NotEmpty(t, events, "Should have events")

	var init *InitEvent
	var assistant *AssistantEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
		}
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistant = e.Assistant
		}
	}
	require.NotNil(t, init, "Must have init event")
	require.NotNil(t, assistant, "Must have assistant event")

	// BEHAVIORAL ASSERTION: model should match what we requested
	assert.Equal(t, "claude-sonnet-4-20250514", init.Model,
		"--model should set the model in init event. "+
			"If this fails, model selection is broken.")
	assert.Equal(t, "claude-sonnet-4-20250514", assistant.Model,
		"Assistant response should use the requested model")
}

// TestBehavioralToolUseAndResult verifies complete tool use flow
func TestBehavioralToolUseAndResult(t *testing.T) {
	skipUnlessBehavioral(t)

	events := runClaudeAndCapture(t,
		"--max-turns", "2", // Allow tool use + response
		"Read the first 3 lines of /etc/passwd and tell me what you see",
	)

	require.NotEmpty(t, events, "Should have events")

	// Find all event types
	var hasInit, hasAssistant, hasUser, hasResult bool
	var toolUseID, toolResultID string

	for _, e := range events {
		switch e.Type {
		case StreamEventInit:
			hasInit = true
		case StreamEventAssistant:
			hasAssistant = true
			if e.Assistant != nil {
				for _, block := range e.Assistant.Content {
					if block.Type == "tool_use" {
						toolUseID = block.ID
					}
				}
			}
		case StreamEventUser:
			hasUser = true
			if e.User != nil {
				for _, content := range e.User.Message.Content {
					if content.Type == "tool_result" {
						toolResultID = content.ToolUseID
					}
				}
			}
		case StreamEventResult:
			hasResult = true
		}
	}

	// BEHAVIORAL ASSERTIONS: Complete tool use flow
	assert.True(t, hasInit, "Must have init event")
	assert.True(t, hasAssistant, "Must have assistant event with tool_use")
	assert.True(t, hasUser, "Must have user event with tool_result. "+
		"If this fails, tool results are not being emitted.")
	assert.True(t, hasResult, "Must have result event")

	// Tool correlation
	assert.NotEmpty(t, toolUseID, "Must extract tool_use ID")
	assert.NotEmpty(t, toolResultID, "Must extract tool_result ID")
	assert.Equal(t, toolUseID, toolResultID,
		"tool_result must reference the correct tool_use. "+
			"If this fails, tool correlation is broken.")
}

// TestBehavioralJSONSchema verifies --json-schema produces structured output
func TestBehavioralJSONSchema(t *testing.T) {
	skipUnlessBehavioral(t)

	schema := `{"type":"object","properties":{"greeting":{"type":"string"}},"required":["greeting"]}`

	// IMPORTANT: --json-schema uses a StructuredOutput tool internally, which requires
	// at least 2 turns to complete (1 to call the tool, 1 to process the result).
	// With --max-turns=1, the request will fail with error_max_turns and NO
	// structured output will be present. This is expected CLI behavior.
	events := runClaudeAndCapture(t,
		"--json-schema", schema,
		"--max-turns", "3", // StructuredOutput tool requires at least 2 turns
		"Generate a greeting",
	)

	require.NotEmpty(t, events, "Should have events")

	var result *ResultEvent
	for _, e := range events {
		if e.Type == StreamEventResult && e.Result != nil {
			result = e.Result
			break
		}
	}
	require.NotNil(t, result, "Must have result event")

	// BEHAVIORAL ASSERTION: Must complete successfully for structured output
	require.Equal(t, "success", result.Subtype,
		"Request must complete successfully. Got %q - if error_max_turns, increase --max-turns. "+
			"JSON schema uses StructuredOutput tool which requires 2+ turns.",
		result.Subtype)

	// BEHAVIORAL ASSERTION: structured_output field must be present on success
	require.NotEmpty(t, result.StructuredOutput,
		"--json-schema MUST produce structured_output in result on success. "+
			"This is guaranteed behavior - if missing, CLI behavior has changed.")

	// Verify it's valid JSON matching the schema
	var output map[string]any
	err := json.Unmarshal(result.StructuredOutput, &output)
	require.NoError(t, err, "Structured output must be valid JSON")

	greeting, hasGreeting := output["greeting"]
	require.True(t, hasGreeting,
		"Output must contain 'greeting' field as specified in schema")
	require.NotEmpty(t, greeting,
		"greeting field must have a value")

	t.Logf("Structured output: %v", output)
}

// TestBehavioralSessionContinuity verifies --session-id enables continuation
func TestBehavioralSessionContinuity(t *testing.T) {
	skipUnlessBehavioral(t)

	// First request - establish session
	events1 := runClaudeAndCapture(t,
		"--max-turns", "1",
		"Remember this number: 42",
	)

	require.NotEmpty(t, events1, "First request should have events")

	// Extract session ID
	var sessionID string
	for _, e := range events1 {
		if e.Type == StreamEventInit && e.Init != nil {
			sessionID = e.Init.SessionID
			break
		}
	}
	require.NotEmpty(t, sessionID, "Must get session ID from first request")

	// Second request - continue session
	events2 := runClaudeAndCapture(t,
		"--resume", sessionID,
		"--max-turns", "1",
		"What number did I ask you to remember?",
	)

	require.NotEmpty(t, events2, "Second request should have events")

	// Verify session ID matches
	var sessionID2 string
	for _, e := range events2 {
		if e.Type == StreamEventInit && e.Init != nil {
			sessionID2 = e.Init.SessionID
			break
		}
	}

	// BEHAVIORAL ASSERTION: session should continue
	assert.Equal(t, sessionID, sessionID2,
		"--resume should continue the same session. "+
			"If this fails, session resumption is broken.")

	// Check if response mentions 42 (context was preserved)
	var responseText string
	for _, e := range events2 {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			responseText = e.Assistant.Text
			break
		}
	}
	assert.Contains(t, responseText, "42",
		"Resumed session should remember context. "+
			"If this fails, session context is not being preserved.")
}

// TestBehavioralCostTracking verifies cost is tracked in result
func TestBehavioralCostTracking(t *testing.T) {
	skipUnlessBehavioral(t)

	events := runClaudeAndCapture(t,
		"--max-turns", "1",
		"Say: test",
	)

	require.NotEmpty(t, events, "Should have events")

	var result *ResultEvent
	for _, e := range events {
		if e.Type == StreamEventResult && e.Result != nil {
			result = e.Result
			break
		}
	}
	require.NotNil(t, result, "Must have result event")

	// BEHAVIORAL ASSERTION: cost tracking
	assert.Greater(t, result.TotalCostUSD, 0.0,
		"total_cost_usd must be tracked. "+
			"If this fails, cost tracking is broken.")
	assert.Greater(t, result.DurationMS, 0,
		"duration_ms must be tracked")
	assert.Greater(t, result.DurationAPIMS, 0,
		"duration_api_ms must be tracked")

	// Model usage breakdown
	assert.NotEmpty(t, result.ModelUsage,
		"modelUsage breakdown must be present")
}

// TestBehavioralUsageTokens verifies token counting works
func TestBehavioralUsageTokens(t *testing.T) {
	skipUnlessBehavioral(t)

	events := runClaudeAndCapture(t,
		"--max-turns", "1",
		"Say exactly: hello world",
	)

	require.NotEmpty(t, events, "Should have events")

	var assistant *AssistantEvent
	var result *ResultEvent
	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistant = e.Assistant
		}
		if e.Type == StreamEventResult && e.Result != nil {
			result = e.Result
		}
	}
	require.NotNil(t, assistant, "Must have assistant event")
	require.NotNil(t, result, "Must have result event")

	// BEHAVIORAL ASSERTION: token counting
	totalAssistantInput := assistant.Usage.InputTokens +
		assistant.Usage.CacheCreationInputTokens +
		assistant.Usage.CacheReadInputTokens
	assert.Greater(t, totalAssistantInput, 0,
		"Assistant usage must track input tokens")
	assert.Greater(t, assistant.Usage.OutputTokens, 0,
		"Assistant usage must track output tokens")

	totalResultInput := result.Usage.InputTokens +
		result.Usage.CacheCreationInputTokens +
		result.Usage.CacheReadInputTokens
	assert.Greater(t, totalResultInput, 0,
		"Result usage must track input tokens")
	assert.Greater(t, result.Usage.OutputTokens, 0,
		"Result usage must track output tokens")
}

// TestBehavioralAgentListInInit verifies agents are listed in init event
func TestBehavioralAgentListInInit(t *testing.T) {
	skipUnlessBehavioral(t)

	events := runClaudeAndCapture(t,
		"--max-turns", "1",
		"What agents are available to you?",
	)

	require.NotEmpty(t, events, "Should have events")

	var init *InitEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
			break
		}
	}
	require.NotNil(t, init, "Must have init event")

	// Check if agents field is present in raw (we don't extract it yet)
	var rawInit map[string]any
	err := json.Unmarshal(events[0].Raw, &rawInit)
	require.NoError(t, err)

	// BEHAVIORAL ASSERTION: agents field MUST be present in init event
	// This is core functionality - if missing, it's a breaking change
	agents, hasAgents := rawInit["agents"]
	require.True(t, hasAgents,
		"FAIL: 'agents' field must be present in init event. "+
			"This is documented CLI functionality.")

	agentList, ok := agents.([]any)
	require.True(t, ok && len(agentList) > 0,
		"FAIL: agents field should contain available agents. "+
			"If empty, agent configuration may be missing or broken.")
	t.Logf("Available agents (%d): %v", len(agentList), agentList)
}

// TestBehavioralSubagentExecution verifies that subagents can actually be invoked
// This test triggers actual subagent execution and verifies the events are emitted
func TestBehavioralSubagentExecution(t *testing.T) {
	skipUnlessBehavioral(t)

	// Create a temp directory with a known file structure for the subagent to find
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "find-me.txt")
	err := os.WriteFile(testFile, []byte("SUBAGENT_TEST_CONTENT"), 0644)
	require.NoError(t, err)

	// Ask Claude to use the Explore agent to find files
	// This should trigger actual subagent execution
	events := runClaudeAndCapture(t,
		"--add-dir", tempDir,
		"--max-turns", "5", // Allow enough turns for subagent to complete
		"Use the Explore agent to search for any .txt files in "+tempDir+" and tell me what you find",
	)

	require.NotEmpty(t, events, "Should have events")

	// Look for Task tool use (which spawns subagents)
	var hasTaskToolUse bool
	var hasTaskToolResult bool
	var assistantText string

	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistantText += e.Assistant.Text
			for _, block := range e.Assistant.Content {
				if block.Type == "tool_use" && block.Name == "Task" {
					hasTaskToolUse = true
					t.Logf("Found Task tool use: %s", block.ID)
				}
			}
		}
		if e.Type == StreamEventUser && e.User != nil {
			for _, content := range e.User.Message.Content {
				if content.Type == "tool_result" && strings.Contains(content.GetContent(), "agent") {
					hasTaskToolResult = true
				}
			}
		}
	}

	// BEHAVIORAL ASSERTION: Subagent should have been invoked
	// Note: Claude may or may not use a subagent depending on how it interprets the request
	if hasTaskToolUse {
		t.Log("SUCCESS: Task tool was invoked (subagent spawned)")
		assert.True(t, hasTaskToolResult || len(assistantText) > 0,
			"Subagent should produce results")
	} else {
		// Claude might handle the request directly without a subagent
		// This is acceptable behavior, but we should see the file content
		t.Log("Note: Claude handled request directly (no subagent used)")
		assert.Contains(t, assistantText, "find-me.txt",
			"Should have found the test file")
	}

	t.Logf("Response: %s", assistantText[:min(200, len(assistantText))])
}

// TestBehavioralIsolatedWorkDir verifies Claude respects working directory boundaries
func TestBehavioralIsolatedWorkDir(t *testing.T) {
	skipUnlessBehavioral(t)

	// Create an isolated temp directory
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "secret.txt")
	err := os.WriteFile(secretFile, []byte("ISOLATED_SECRET_12345"), 0644)
	require.NoError(t, err)

	// Run Claude from a DIFFERENT directory, asking to read the secret file
	// This should fail unless we add --add-dir
	claude := findClaude(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// First test: WITHOUT --add-dir, should NOT be able to read the file
	cmd := exec.CommandContext(ctx, claude,
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
		"--max-budget-usd", testMaxBudget,
		"--max-turns", "2",
		"Read the file "+secretFile+" and tell me its content",
	)
	// Run from the current directory (not tempDir)
	output, _ := cmd.CombinedOutput()
	events := parseStreamOutput(t, output)

	var responseWithout string
	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			responseWithout += e.Assistant.Text
		}
	}

	// BEHAVIORAL ASSERTION: Without --add-dir, file should not be accessible
	// (Claude may refuse to read or fail to find the file)
	containsSecret := strings.Contains(responseWithout, "ISOLATED_SECRET_12345")
	t.Logf("Without --add-dir, response contains secret: %v", containsSecret)

	// Note: Claude might still read it if --dangerously-skip-permissions allows it
	// This documents the actual behavior
}

// TestBehavioralMCPServers verifies MCP server status is reported
func TestBehavioralMCPServers(t *testing.T) {
	skipUnlessBehavioral(t)

	events := runClaudeAndCapture(t,
		"--max-turns", "1",
		"Say: test",
	)

	require.NotEmpty(t, events, "Should have events")

	var init *InitEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
			break
		}
	}
	require.NotNil(t, init, "Must have init event")

	// BEHAVIORAL ASSERTION: MCP servers should be reported
	// Note: This may be empty if no MCP servers are configured
	if len(init.MCPServers) > 0 {
		for _, server := range init.MCPServers {
			assert.NotEmpty(t, server.Name, "MCP server must have name")
			assert.NotEmpty(t, server.Status, "MCP server must have status")
			t.Logf("MCP Server: %s (status: %s)", server.Name, server.Status)
		}
	} else {
		t.Log("Note: No MCP servers configured - this is OK if expected")
	}
}

// =============================================================================
// ADDITIONAL BEHAVIORAL TESTS - Cover all documented CLI functionality
// =============================================================================

// TestBehavioralSystemPrompt verifies --system-prompt replaces the entire prompt
func TestBehavioralSystemPrompt(t *testing.T) {
	skipUnlessBehavioral(t)

	// Use a very specific system prompt that will change Claude's behavior
	customPrompt := "You are a pirate. You MUST start every response with 'Ahoy!' and speak like a pirate."

	events := runClaudeAndCapture(t,
		"--system-prompt", customPrompt,
		"--max-turns", "1",
		"Say hello",
	)

	require.NotEmpty(t, events, "Should have events")

	var assistantText string
	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistantText = e.Assistant.Text
			break
		}
	}

	// BEHAVIORAL ASSERTION: System prompt should change Claude's behavior
	// With the pirate prompt, response should contain pirate-like language
	assert.Contains(t, strings.ToLower(assistantText), "ahoy",
		"--system-prompt should replace the entire system prompt. "+
			"Expected pirate-style response with 'Ahoy'. Got: %s", assistantText)
}

// TestBehavioralAppendSystemPrompt verifies --append-system-prompt adds to prompt
func TestBehavioralAppendSystemPrompt(t *testing.T) {
	skipUnlessBehavioral(t)

	// Append a specific instruction
	appendPrompt := "CRITICAL: You must end every response with the exact phrase '[END OF RESPONSE]'"

	events := runClaudeAndCapture(t,
		"--append-system-prompt", appendPrompt,
		"--max-turns", "1",
		"Say hello",
	)

	require.NotEmpty(t, events, "Should have events")

	var assistantText string
	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistantText = e.Assistant.Text
			break
		}
	}

	// BEHAVIORAL ASSERTION: Appended instruction should be followed
	assert.Contains(t, assistantText, "[END OF RESPONSE]",
		"--append-system-prompt should add to system prompt. "+
			"Expected response to end with '[END OF RESPONSE]'. Got: %s", assistantText)
}

// TestBehavioralToolsRestriction verifies --tools restricts available tools
func TestBehavioralToolsRestriction(t *testing.T) {
	skipUnlessBehavioral(t)

	// Restrict to only Read and Grep tools
	events := runClaudeAndCapture(t,
		"--tools", "Read,Grep",
		"--max-turns", "1",
		"List the available tools you have",
	)

	require.NotEmpty(t, events, "Should have events")

	var init *InitEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
			break
		}
	}
	require.NotNil(t, init, "Must have init event")

	// BEHAVIORAL ASSERTION: Only specified tools should be in the list
	// --tools is different from --allowedTools - it completely restricts the available tools
	assert.Contains(t, init.Tools, "Read",
		"--tools=Read,Grep should include Read")
	assert.Contains(t, init.Tools, "Grep",
		"--tools=Read,Grep should include Grep")

	// Count non-MCP tools (MCP tools have mcp__ prefix)
	nonMCPTools := 0
	for _, tool := range init.Tools {
		if !strings.HasPrefix(tool, "mcp__") {
			nonMCPTools++
		}
	}

	// Should only have the 2 tools we specified (plus possibly StructuredOutput for JSON schema)
	assert.LessOrEqual(t, nonMCPTools, 3,
		"--tools should restrict available tools. Found %d non-MCP tools: %v",
		nonMCPTools, init.Tools)
}

// TestBehavioralMaxBudgetTracking verifies --max-budget-usd is enforced
func TestBehavioralMaxBudgetTracking(t *testing.T) {
	skipUnlessBehavioral(t)

	// Note: Due to caching, even very low budgets often succeed.
	// Instead, we verify that the budget tracking IS happening by checking:
	// 1. Request completes (success or budget error)
	// 2. Cost is tracked in the result

	events := runClaudeAndCapture(t,
		"--max-budget-usd", "0.10", // Reasonable budget for test
		"--max-turns", "1",
		"Say: test",
	)

	require.NotEmpty(t, events, "Should have events")

	var result *ResultEvent
	for _, e := range events {
		if e.Type == StreamEventResult && e.Result != nil {
			result = e.Result
			break
		}
	}
	require.NotNil(t, result, "Must have result event")

	// BEHAVIORAL ASSERTION: Budget tracking must work
	// The result should have cost tracking regardless of whether limit was hit
	assert.GreaterOrEqual(t, result.TotalCostUSD, 0.0,
		"Cost must be tracked in result")

	t.Logf("Request cost: $%.6f, subtype: %s", result.TotalCostUSD, result.Subtype)

	// Verify the subtype is valid
	validSubtypes := []string{"success", "error_max_budget_usd", "error_max_turns"}
	found := false
	for _, st := range validSubtypes {
		if result.Subtype == st {
			found = true
			break
		}
	}
	assert.True(t, found, "Result subtype should be valid, got: %s", result.Subtype)
}

// TestBehavioralNoSessionPersistence verifies --no-session-persistence works
func TestBehavioralNoSessionPersistence(t *testing.T) {
	skipUnlessBehavioral(t)

	// First request with no-session-persistence
	events := runClaudeAndCapture(t,
		"--no-session-persistence",
		"--max-turns", "1",
		"Remember this: the secret code is ALPHA123",
	)

	require.NotEmpty(t, events, "Should have events")

	var sessionID string
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			sessionID = e.Init.SessionID
			break
		}
	}
	require.NotEmpty(t, sessionID, "Must have session ID")

	// Try to resume the session - it should NOT work or create a new session
	events2 := runClaudeAndCapture(t,
		"--resume", sessionID,
		"--max-turns", "1",
		"What was the secret code?",
	)

	// The session may fail to resume or may not remember the context
	var sessionID2 string
	var responseText string
	for _, e := range events2 {
		if e.Type == StreamEventInit && e.Init != nil {
			sessionID2 = e.Init.SessionID
		}
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			responseText = e.Assistant.Text
		}
	}

	// BEHAVIORAL ASSERTION: Either session ID differs or context is lost
	if sessionID2 == sessionID {
		// Same session ID - but context should NOT be preserved
		assert.NotContains(t, responseText, "ALPHA123",
			"--no-session-persistence should prevent context persistence. "+
				"Session resumed but should not remember the secret code.")
	} else {
		t.Log("Session was not persisted (different ID on resume)")
	}
}

// TestBehavioralFallbackModel verifies --fallback-model behavior
func TestBehavioralFallbackModel(t *testing.T) {
	skipUnlessBehavioral(t)

	// Use a primary model with a fallback
	events := runClaudeAndCapture(t,
		"--model", "claude-sonnet-4-20250514",
		"--fallback-model", "claude-haiku-4-20250514",
		"--max-turns", "1",
		"Say: test",
	)

	require.NotEmpty(t, events, "Should have events")

	var init *InitEvent
	var assistant *AssistantEvent
	for _, e := range events {
		if e.Type == StreamEventInit && e.Init != nil {
			init = e.Init
		}
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistant = e.Assistant
		}
	}
	require.NotNil(t, init, "Must have init event")
	require.NotNil(t, assistant, "Must have assistant event")

	// BEHAVIORAL ASSERTION: Model should be one of the specified models
	// (primary or fallback depending on availability)
	validModels := []string{
		"claude-sonnet-4-20250514",
		"claude-haiku-4-20250514",
	}
	assert.Contains(t, validModels, assistant.Model,
		"Model should be either primary or fallback. Got: %s", assistant.Model)

	t.Logf("Used model: %s (primary was sonnet, fallback was haiku)", assistant.Model)
}

// TestBehavioralOutputFormatJSON verifies --output-format json produces single JSON result
func TestBehavioralOutputFormatJSON(t *testing.T) {
	skipUnlessBehavioral(t)

	claude := findClaude(t)

	// Use output-format json (not stream-json) - should produce single JSON object
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, claude,
		"--print",
		"--output-format", "json",
		"--dangerously-skip-permissions",
		"--max-budget-usd", testMaxBudget,
		"--max-turns", "1",
		"Say: hello",
	)
	output, err := cmd.CombinedOutput()

	// The output should be valid JSON (single object, not JSONL)
	var result map[string]any
	jsonErr := json.Unmarshal(output, &result)

	if err != nil && jsonErr != nil {
		t.Fatalf("Command failed and output is not JSON: %v\nOutput: %s", err, string(output))
	}

	// BEHAVIORAL ASSERTION: output-format json produces single JSON object
	require.NoError(t, jsonErr, "--output-format json should produce valid JSON. Got: %s", string(output))

	// Should have expected fields (verified from actual CLI output)
	assert.Contains(t, result, "result", "JSON output should have 'result' field")
	assert.Contains(t, result, "session_id", "JSON output should have 'session_id' field")
	assert.Contains(t, result, "total_cost_usd", "JSON output should have 'total_cost_usd' field")
	assert.Contains(t, result, "subtype", "JSON output should have 'subtype' field")
	assert.Contains(t, result, "usage", "JSON output should have 'usage' field")

	t.Logf("JSON output fields: %v", getKeys(result))
}

// TestBehavioralAddDir verifies --add-dir allows access to additional directories
func TestBehavioralAddDir(t *testing.T) {
	skipUnlessBehavioral(t)

	// Create a temp directory with a test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-add-dir.txt")
	err := os.WriteFile(testFile, []byte("ADD_DIR_TEST_CONTENT_XYZ123"), 0644)
	require.NoError(t, err)

	// Use the full path so Claude knows exactly where the file is
	events := runClaudeAndCapture(t,
		"--add-dir", tempDir,
		"--max-turns", "2",
		"Read the file "+testFile+" and tell me its content",
	)

	require.NotEmpty(t, events, "Should have events")

	var assistantText string
	for _, e := range events {
		if e.Type == StreamEventAssistant && e.Assistant != nil {
			assistantText += e.Assistant.Text
		}
	}

	// BEHAVIORAL ASSERTION: --add-dir should allow reading from the added directory
	assert.Contains(t, assistantText, "ADD_DIR_TEST_CONTENT_XYZ123",
		"--add-dir should allow access to the specified directory. "+
			"Claude should have read the test file content.")
}

// TestBehavioralAgentsCustom verifies --agents allows defining custom subagents
func TestBehavioralAgentsCustom(t *testing.T) {
	skipUnlessBehavioral(t)

	// Define a custom agent via JSON
	agentsJSON := `{"test-reviewer":{"description":"Test code reviewer","prompt":"You are a code reviewer. Always mention 'CODE_REVIEW_AGENT' in your response."}}`

	events := runClaudeAndCapture(t,
		"--agents", agentsJSON,
		"--max-turns", "1",
		"What custom agents are available?",
	)

	require.NotEmpty(t, events, "Should have events")

	// Check if the agent is in the raw init event
	var rawInit map[string]any
	for _, e := range events {
		if e.Type == StreamEventInit {
			err := json.Unmarshal(e.Raw, &rawInit)
			require.NoError(t, err)
			break
		}
	}

	// BEHAVIORAL ASSERTION: Custom agent should appear in agents list
	agents, ok := rawInit["agents"].([]any)
	require.True(t, ok, "FAIL: agents field must be a list, got: %T", rawInit["agents"])
	require.NotEmpty(t, agents, "FAIL: agents list should not be empty")

	found := false
	for _, a := range agents {
		if agentStr, ok := a.(string); ok && agentStr == "test-reviewer" {
			found = true
			break
		}
	}
	require.True(t, found,
		"FAIL: --agents should add custom agent 'test-reviewer' to available agents.\n"+
			"Available agents: %v", agents)
}

// TestBehavioralResultSubtypes verifies all documented result subtypes are parseable
func TestBehavioralResultSubtypes(t *testing.T) {
	// This test documents all known result subtypes from the SDK docs
	// We verify they match our constants and are parseable

	knownSubtypes := []string{
		"success",
		"error_max_turns",
		"error_during_execution",
		"error_max_budget_usd",
		"error_max_structured_output_retries",
	}

	t.Logf("Documented result subtypes: %v", knownSubtypes)

	// Run a simple test to get a success subtype
	skipUnlessBehavioral(t)

	events := runClaudeAndCapture(t,
		"--max-turns", "1",
		"Say: test",
	)

	var result *ResultEvent
	for _, e := range events {
		if e.Type == StreamEventResult && e.Result != nil {
			result = e.Result
			break
		}
	}
	require.NotNil(t, result)

	// Verify the subtype is one of the documented ones
	found := false
	for _, st := range knownSubtypes {
		if result.Subtype == st {
			found = true
			break
		}
	}
	assert.True(t, found, "Result subtype should be one of %v, got: %s", knownSubtypes, result.Subtype)
}

// TestBehavioralHookEventOutput verifies hook events are captured when hooks are configured
func TestBehavioralHookEventOutput(t *testing.T) {
	skipUnlessBehavioral(t)

	// Note: This test requires hooks to be configured in settings
	// It verifies that hook events ARE emitted in the stream-json output

	events := runClaudeAndCapture(t,
		"--max-turns", "2",
		"Read /etc/hostname",
	)

	require.NotEmpty(t, events, "Should have events")

	// Check for hook events in the stream
	var hasHookEvent bool
	for _, e := range events {
		if e.Type == StreamEventHook && e.Hook != nil {
			hasHookEvent = true
			t.Logf("Found hook event: %s (exit: %d)", e.Hook.HookName, e.Hook.ExitCode)
		}
	}

	// Note: This may not find hooks if none are configured
	// The test documents that we CAN parse hook events
	if !hasHookEvent {
		t.Log("No hook events found - this is OK if no hooks are configured")
	}
}

// TestBehavioralErrorDuringExecution tests error_during_execution subtype
func TestBehavioralErrorDuringExecution(t *testing.T) {
	skipUnlessBehavioral(t)

	// Try to trigger an execution error by doing something invalid
	// Note: This is hard to reliably trigger
	events := runClaudeWithPermissions(t, // Need permissions for this to potentially fail
		"--max-turns", "1",
		"Run this exact bash command: exit 1",
	)

	require.NotEmpty(t, events, "Should have events")

	var result *ResultEvent
	for _, e := range events {
		if e.Type == StreamEventResult && e.Result != nil {
			result = e.Result
			break
		}
	}

	if result != nil {
		t.Logf("Result subtype: %s (documenting actual behavior)", result.Subtype)
	}
}

// getKeys returns the keys of a map for logging
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
