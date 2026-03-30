package codexconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCustomAgents(t *testing.T) {
	home := t.TempDir()
	projectRoot := t.TempDir()
	t.Setenv("HOME", home)

	userDir := filepath.Join(home, ".codex", "agents")
	projectDir := filepath.Join(projectRoot, ".codex", "agents")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll userDir: %v", err)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll projectDir: %v", err)
	}

	userAgent := "name = \"reviewer\"\ndescription = \"Reviewer\"\ndeveloper_instructions = \"Review everything.\"\n"
	projectAgent := "name = \"explorer\"\ndescription = \"Explorer\"\ndeveloper_instructions = \"Map the codebase.\"\n"

	if err := os.WriteFile(filepath.Join(userDir, "reviewer.toml"), []byte(userAgent), 0o644); err != nil {
		t.Fatalf("write user agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "explorer.toml"), []byte(projectAgent), 0o644); err != nil {
		t.Fatalf("write project agent: %v", err)
	}

	agents, err := DiscoverCustomAgents(projectRoot)
	if err != nil {
		t.Fatalf("DiscoverCustomAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("len(agents) = %d, want 2", len(agents))
	}
}
