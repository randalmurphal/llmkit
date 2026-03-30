package codexconfig

import (
	"path/filepath"
	"testing"
)

func TestValidHookEventsIncludesToolHooks(t *testing.T) {
	events := ValidHookEvents()
	want := []HookEvent{HookSessionStart, HookPreToolUse, HookPostToolUse, HookUserPromptSubmit, HookStop}
	for _, event := range want {
		found := false
		for _, got := range events {
			if got == event {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing hook event %q", event)
		}
	}
}

func TestHooksRoundTrip(t *testing.T) {
	root := t.TempDir()
	cfg := &HookConfig{
		Hooks: map[string][]HookMatcher{
			string(HookPreToolUse): {{
				Matcher: "shell",
				Hooks: []HookEntry{{
					Type:          "command",
					Command:       "echo pre",
					Timeout:       30,
					StatusMessage: "running",
				}},
			}},
		},
	}

	if err := SaveHooks(root, cfg); err != nil {
		t.Fatalf("SaveHooks: %v", err)
	}
	if got := HooksPath(root); got != filepath.Join(root, ".codex", "hooks.json") {
		t.Fatalf("HooksPath = %q", got)
	}

	loaded, err := LoadHooks(root)
	if err != nil {
		t.Fatalf("LoadHooks: %v", err)
	}
	if len(loaded.Hooks[string(HookPreToolUse)]) != 1 {
		t.Fatalf("expected 1 pre-tool matcher, got %d", len(loaded.Hooks[string(HookPreToolUse)]))
	}
	if loaded.Hooks[string(HookPreToolUse)][0].Hooks[0].Command != "echo pre" {
		t.Fatalf("unexpected command %q", loaded.Hooks[string(HookPreToolUse)][0].Hooks[0].Command)
	}
}
