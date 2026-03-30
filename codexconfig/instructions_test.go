package codexconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInstructionsHierarchy(t *testing.T) {
	home := t.TempDir()
	projectRoot := t.TempDir()
	cwd := filepath.Join(projectRoot, "app", "handlers")
	t.Setenv("HOME", home)
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("MkdirAll cwd: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("MkdirAll global: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "AGENTS.md"), []byte("global"), 0o644); err != nil {
		t.Fatalf("write global AGENTS: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("project"), 0o644); err != nil {
		t.Fatalf("write project AGENTS: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "AGENTS.override.md"), []byte("local override"), 0o644); err != nil {
		t.Fatalf("write local override: %v", err)
	}

	h, err := ResolveInstructions(projectRoot, cwd, &ConfigFile{})
	if err != nil {
		t.Fatalf("ResolveInstructions: %v", err)
	}
	if h.Global == nil || h.Global.Content != "global" {
		t.Fatalf("unexpected global instructions: %#v", h.Global)
	}
	if len(h.Project) != 2 {
		t.Fatalf("len(project instructions) = %d, want 2", len(h.Project))
	}
}
