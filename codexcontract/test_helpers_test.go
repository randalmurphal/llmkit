package codexcontract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func findCodexCLI() (string, error) {
	if p, err := exec.LookPath("codex"); err == nil {
		return p, nil
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		"/usr/local/bin/codex",
		filepath.Join(home, ".local", "bin", "codex"),
		filepath.Join(home, ".npm-global", "bin", "codex"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", exec.ErrNotFound
}

func runCodexOutput(t *testing.T, codexPath string, args ...string) string {
	t.Helper()
	cmd := exec.Command(codexPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run %s %s: %v\noutput:\n%s", codexPath, strings.Join(args, " "), err, string(out))
	}
	return normalizeOutput(string(out))
}

func normalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "WARNING:") {
			continue
		}
		filtered = append(filtered, strings.TrimRight(line, " \t"))
	}
	return strings.TrimSpace(strings.Join(filtered, "\n")) + "\n"
}
