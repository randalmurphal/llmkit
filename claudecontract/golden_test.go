package claudecontract

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenHelpOutput compares current CLI help against the golden file.
// Run with -update to regenerate the golden file.
func TestGoldenHelpOutput(t *testing.T) {
	claudePath, err := findClaudeCLI()
	if err != nil {
		t.Skip("Claude CLI not found, skipping golden file test")
	}

	// Get current help output
	cmd := claudeCmd(claudePath, "--help")
	currentOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get CLI help: %v", err)
	}

	// Determine golden file path based on version
	version, err := DetectCLIVersion(claudePath)
	if err != nil {
		t.Fatalf("Failed to detect CLI version: %v", err)
	}

	goldenPath := filepath.Join("testdata", "help_v"+version.Raw+".txt")

	// Check if we should update the golden file
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("Failed to create testdata directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, currentOutput, 0o644); err != nil {
			t.Fatalf("Failed to write golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read golden file
	goldenOutput, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Logf("Golden file not found: %s", goldenPath)
			t.Log("Run with UPDATE_GOLDEN=1 to create it:")
			t.Log("  UPDATE_GOLDEN=1 go test ./claudecontract/... -run TestGoldenHelpOutput")
			t.Skip("Golden file does not exist")
		}
		t.Fatalf("Failed to read golden file: %v", err)
	}

	// Compare outputs
	if !bytes.Equal(currentOutput, goldenOutput) {
		// Find differences
		currentLines := strings.Split(string(currentOutput), "\n")
		goldenLines := strings.Split(string(goldenOutput), "\n")

		t.Error("CLI help output differs from golden file")
		t.Log("Differences:")

		// Simple diff - show added/removed lines
		goldenSet := make(map[string]bool)
		for _, line := range goldenLines {
			goldenSet[strings.TrimSpace(line)] = true
		}

		currentSet := make(map[string]bool)
		for _, line := range currentLines {
			currentSet[strings.TrimSpace(line)] = true
		}

		for _, line := range currentLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !goldenSet[trimmed] {
				t.Logf("  + %s", line)
			}
		}

		for _, line := range goldenLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !currentSet[trimmed] {
				t.Logf("  - %s", line)
			}
		}

		t.Log("")
		t.Log("Run with UPDATE_GOLDEN=1 to update the golden file if these changes are expected:")
		t.Log("  UPDATE_GOLDEN=1 go test ./claudecontract/... -run TestGoldenHelpOutput")
	}
}

// TestEventTypesWithRealCLI validates event types by checking a minimal CLI interaction.
// This test requires the CLI to be available and may use API credits.
func TestEventTypesWithRealCLI(t *testing.T) {
	if os.Getenv("TEST_REAL_CLI") != "1" {
		t.Skip("Skipping real CLI test. Set TEST_REAL_CLI=1 to run.")
	}

	claudePath, err := findClaudeCLI()
	if err != nil {
		t.Skip("Claude CLI not found")
	}

	// Run a minimal command to get stream-json output
	cmd := claudeCmd(claudePath,
		"--print",
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
		"--max-budget-usd", "0.01",
		"Say exactly: test",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("CLI output: %s", string(output))
		t.Fatalf("CLI command failed: %v", err)
	}

	outputStr := string(output)

	// Validate we see expected event types
	expectedTypes := []string{
		`"type":"` + EventTypeSystem + `"`,
		`"type":"` + EventTypeAssistant + `"`,
		`"type":"` + EventTypeResult + `"`,
	}

	for _, expected := range expectedTypes {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected event type pattern %q not found in output", expected)
		}
	}

	// Validate we see init subtype
	if !strings.Contains(outputStr, `"subtype":"`+SubtypeInit+`"`) {
		t.Error("Expected init subtype not found in output")
	}

	// Validate we see success subtype
	if !strings.Contains(outputStr, `"subtype":"`+ResultSubtypeSuccess+`"`) {
		t.Error("Expected success subtype not found in output")
	}
}
