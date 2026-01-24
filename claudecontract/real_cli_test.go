package claudecontract

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// These tests validate that our constants match what the real Claude CLI produces.
// They use golden files from ../claude/testdata/golden/ which contain actual CLI output.
//
// This ensures the contract package stays in sync with real CLI behavior.

func TestEventTypeConstantsMatchRealCLI(t *testing.T) {
	goldenFiles := []string{
		"simple_text_response.jsonl",
		"tool_use_response.jsonl",
	}

	expectedTypes := map[string]bool{
		EventTypeSystem:    false,
		EventTypeAssistant: false,
		EventTypeUser:      false,
		EventTypeResult:    false,
	}

	for _, filename := range goldenFiles {
		events := loadGoldenEventsContract(t, filename)
		for _, event := range events {
			var base struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(event, &base); err != nil {
				continue
			}
			if _, known := expectedTypes[base.Type]; known {
				expectedTypes[base.Type] = true
			} else {
				t.Errorf("Unknown event type in real CLI output: %q", base.Type)
			}
		}
	}

	// Verify we saw the event types we expect
	if !expectedTypes[EventTypeSystem] {
		t.Error("EventTypeSystem constant not found in real CLI output")
	}
	if !expectedTypes[EventTypeAssistant] {
		t.Error("EventTypeAssistant constant not found in real CLI output")
	}
	if !expectedTypes[EventTypeResult] {
		t.Error("EventTypeResult constant not found in real CLI output")
	}
	// User events only appear with tool use
	if !expectedTypes[EventTypeUser] {
		t.Log("EventTypeUser not found - only appears with tool use")
	}
}

func TestSubtypeConstantsMatchRealCLI(t *testing.T) {
	// Check init subtype
	events := loadGoldenEventsContract(t, "simple_text_response.jsonl")

	var foundInit bool
	for _, event := range events {
		var base struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
		}
		if err := json.Unmarshal(event, &base); err != nil {
			continue
		}
		if base.Type == EventTypeSystem && base.Subtype == SubtypeInit {
			foundInit = true
			break
		}
	}

	if !foundInit {
		t.Errorf("SubtypeInit (%q) not found in real CLI output", SubtypeInit)
	}
}

func TestResultSubtypeConstantsMatchRealCLI(t *testing.T) {
	testCases := []struct {
		filename        string
		expectedSubtype string
	}{
		{"simple_text_response.jsonl", ResultSubtypeSuccess},
		{"tool_use_response.jsonl", ResultSubtypeErrorMaxTurns},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			events := loadGoldenEventsContract(t, tc.filename)

			var found bool
			for _, event := range events {
				var base struct {
					Type    string `json:"type"`
					Subtype string `json:"subtype"`
				}
				if err := json.Unmarshal(event, &base); err != nil {
					continue
				}
				if base.Type == EventTypeResult && base.Subtype == tc.expectedSubtype {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Result subtype %q not found in %s", tc.expectedSubtype, tc.filename)
			}
		})
	}
}

func TestContentTypeConstantsMatchRealCLI(t *testing.T) {
	// Check for text content type
	events := loadGoldenEventsContract(t, "simple_text_response.jsonl")

	var foundText bool
	for _, event := range events {
		var assistant struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(event, &assistant); err != nil {
			continue
		}
		if assistant.Type == EventTypeAssistant {
			for _, block := range assistant.Message.Content {
				if block.Type == ContentTypeText {
					foundText = true
				}
			}
		}
	}

	if !foundText {
		t.Errorf("ContentTypeText (%q) not found in real CLI output", ContentTypeText)
	}

	// Check for tool_use content type
	events = loadGoldenEventsContract(t, "tool_use_response.jsonl")

	var foundToolUse bool
	for _, event := range events {
		var assistant struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(event, &assistant); err != nil {
			continue
		}
		if assistant.Type == EventTypeAssistant {
			for _, block := range assistant.Message.Content {
				if block.Type == ContentTypeToolUse {
					foundToolUse = true
				}
			}
		}
	}

	if !foundToolUse {
		t.Errorf("ContentTypeToolUse (%q) not found in real CLI output", ContentTypeToolUse)
	}

	// Check for tool_result in user events
	var foundToolResult bool
	for _, event := range events {
		var user struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(event, &user); err != nil {
			continue
		}
		if user.Type == EventTypeUser {
			for _, block := range user.Message.Content {
				if block.Type == ContentTypeToolResult {
					foundToolResult = true
				}
			}
		}
	}

	if !foundToolResult {
		t.Errorf("ContentTypeToolResult (%q) not found in real CLI output", ContentTypeToolResult)
	}
}

func TestToolNamesMatchRealCLI(t *testing.T) {
	events := loadGoldenEventsContract(t, "simple_text_response.jsonl")

	// Extract tools from init event
	var cliTools []string
	for _, event := range events {
		var init struct {
			Type    string   `json:"type"`
			Subtype string   `json:"subtype"`
			Tools   []string `json:"tools"`
		}
		if err := json.Unmarshal(event, &init); err != nil {
			continue
		}
		if init.Type == EventTypeSystem && init.Subtype == SubtypeInit {
			cliTools = init.Tools
			break
		}
	}

	if len(cliTools) == 0 {
		t.Fatal("No tools found in init event")
	}

	// Verify our critical tool constants are in the CLI's tool list
	criticalTools := []string{
		ToolRead,
		ToolWrite,
		ToolEdit,
		ToolBash,
		ToolGlob,
		ToolGrep,
		ToolTask,
		ToolWebFetch,
		ToolWebSearch,
	}

	toolSet := make(map[string]bool)
	for _, tool := range cliTools {
		toolSet[tool] = true
	}

	for _, expected := range criticalTools {
		if !toolSet[expected] {
			t.Errorf("Tool constant %q not found in real CLI tools list", expected)
		}
	}
}

func TestPermissionModeMatchesRealCLI(t *testing.T) {
	events := loadGoldenEventsContract(t, "simple_text_response.jsonl")

	// Extract permission mode from init event
	var permMode string
	for _, event := range events {
		var init struct {
			Type           string `json:"type"`
			Subtype        string `json:"subtype"`
			PermissionMode string `json:"permissionMode"`
		}
		if err := json.Unmarshal(event, &init); err != nil {
			continue
		}
		if init.Type == EventTypeSystem && init.Subtype == SubtypeInit {
			permMode = init.PermissionMode
			break
		}
	}

	if permMode == "" {
		t.Fatal("No permission mode found in init event")
	}

	// Verify the permission mode is one we know about
	validModes := []PermissionMode{
		PermissionDefault,
		PermissionAcceptEdits,
		PermissionBypassPermissions,
		PermissionPlan,
	}

	var found bool
	for _, mode := range validModes {
		if string(mode) == permMode {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Permission mode %q from CLI is not in our constants", permMode)
	}
}

func TestUsageFieldNamingMatchesRealCLI(t *testing.T) {
	events := loadGoldenEventsContract(t, "simple_text_response.jsonl")

	// Check result event usage field naming
	for _, event := range events {
		var result struct {
			Type       string         `json:"type"`
			Usage      map[string]any `json:"usage"`
			ModelUsage map[string]any `json:"modelUsage"`
		}
		if err := json.Unmarshal(event, &result); err != nil {
			continue
		}
		if result.Type != EventTypeResult {
			continue
		}

		// Usage uses snake_case
		expectedSnakeCase := []string{
			"input_tokens",
			"output_tokens",
		}
		for _, field := range expectedSnakeCase {
			if _, ok := result.Usage[field]; !ok {
				t.Errorf("Expected snake_case field %q in result.usage", field)
			}
		}

		// ModelUsage uses camelCase
		if len(result.ModelUsage) > 0 {
			for model, usage := range result.ModelUsage {
				usageMap, ok := usage.(map[string]any)
				if !ok {
					continue
				}
				expectedCamelCase := []string{
					"inputTokens",
					"outputTokens",
					"costUSD",
				}
				for _, field := range expectedCamelCase {
					if _, ok := usageMap[field]; !ok {
						t.Errorf("Expected camelCase field %q in modelUsage[%s]", field, model)
					}
				}
				break // Only need to check one model
			}
		}

		break // Only need one result event
	}
}

func TestMCPServerStatusMatchesRealCLI(t *testing.T) {
	events := loadGoldenEventsContract(t, "simple_text_response.jsonl")

	for _, event := range events {
		var init struct {
			Type       string `json:"type"`
			Subtype    string `json:"subtype"`
			MCPServers []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"mcp_servers"`
		}
		if err := json.Unmarshal(event, &init); err != nil {
			continue
		}
		if init.Type != EventTypeSystem || init.Subtype != SubtypeInit {
			continue
		}

		if len(init.MCPServers) == 0 {
			t.Skip("No MCP servers in test output")
		}

		// Verify status values match our constants
		validStatuses := map[string]bool{
			string(MCPStatusConnected): true,
			string(MCPStatusFailed):    true,
			string(MCPStatusNeedsAuth): true,
			string(MCPStatusPending):   true,
		}

		for _, server := range init.MCPServers {
			if !validStatuses[server.Status] {
				t.Errorf("Unknown MCP server status %q for server %s", server.Status, server.Name)
			}
		}

		break
	}
}

// === Helper ===

func loadGoldenEventsContract(t *testing.T, filename string) []json.RawMessage {
	t.Helper()

	// Golden files are in ../claude/testdata/golden/
	goldenPath := filepath.Join("..", "claude", "testdata", "golden", filename)
	f, err := os.Open(goldenPath)
	if os.IsNotExist(err) {
		t.Skipf("Golden file not found: %s", goldenPath)
	}
	if err != nil {
		t.Fatalf("Failed to open golden file: %v", err)
	}
	defer func() { _ = f.Close() }()

	var events []json.RawMessage
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Make a copy since scanner reuses buffer
		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)
		events = append(events, lineCopy)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	return events
}
