package codexcontract

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestEventTypesWithRealCLI validates event type constants against a real codex exec invocation.
// This test may consume API credits and requires authenticated Codex CLI.
func TestEventTypesWithRealCLI(t *testing.T) {
	if os.Getenv("TEST_REAL_CLI") != "1" {
		t.Skip("Skipping real CLI test. Set TEST_REAL_CLI=1 to run.")
	}

	codexPath, err := findCodexCLI()
	if err != nil {
		t.Skip("codex CLI not found")
	}

	cmd := exec.Command(codexPath,
		"exec",
		"--json",
		"--skip-git-repo-check",
		"-c", `model_reasoning_effort="minimal"`,
		"Say exactly: test",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.ToLower(string(output))
		if strings.Contains(outStr, "permission denied") ||
			strings.Contains(outStr, "codex login") ||
			strings.Contains(outStr, "not authenticated") {
			t.Skipf("Skipping real CLI test due environment/auth issue: %s", string(output))
		}
		t.Logf("CLI output:\n%s", string(output))
		t.Fatalf("codex exec failed: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	found := map[string]bool{
		EventThreadStarted: false,
		EventTurnCompleted: false,
		EventTurnFailed:    false,
	}

	var parsedLines int
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &base); err != nil {
			continue
		}
		parsedLines++
		if _, ok := found[base.Type]; ok {
			found[base.Type] = true
		}
	}

	if parsedLines == 0 {
		t.Fatalf("did not parse any JSON event lines from codex exec output:\n%s", string(output))
	}
	if !found[EventThreadStarted] {
		t.Errorf("expected %q event type in output", EventThreadStarted)
	}
	if !found[EventTurnCompleted] && !found[EventTurnFailed] {
		t.Errorf("expected either %q or %q event type in output", EventTurnCompleted, EventTurnFailed)
	}
}
