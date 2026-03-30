package llmkit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/randalmurphal/llmkit/v2/claudeconfig"
)

func TestProviderDefinitionsExposeClaudeAndCodex(t *testing.T) {
	defs := ListProviders()
	if len(defs) < 2 {
		t.Fatalf("ListProviders returned %d definitions", len(defs))
	}

	claude, ok := GetProviderDefinition("claude")
	if !ok {
		t.Fatal("claude provider definition missing")
	}
	if !claude.Shared.MCPServers || !claude.Environment.Hooks {
		t.Fatalf("unexpected claude definition: %+v", claude)
	}

	codex, ok := GetProviderDefinition("codex")
	if !ok {
		t.Fatal("codex provider definition missing")
	}
	if !codex.Environment.MCP || codex.Shared.AllowedTools {
		t.Fatalf("unexpected codex definition: %+v", codex)
	}
}

func TestBuildConfigCombinesSharedAndProviderRuntimeConfig(t *testing.T) {
	cfg, err := BuildConfig("codex", "gpt-5", "/repo", RuntimeConfig{
		Shared: SharedRuntimeConfig{
			SystemPrompt: "base",
			Env:          map[string]string{"A": "1"},
			AddDirs:      []string{"/tmp"},
		},
		Providers: RuntimeProviderConfig{
			Codex: &CodexRuntimeConfig{
				ReasoningEffort: "high",
				WebSearchMode:   "live",
			},
		},
	}, &SessionMetadata{Provider: "codex", Data: []byte(`{"session_id":"sess-1"}`)})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	if cfg.ReasoningEffort != "high" || cfg.WebSearchMode != "live" {
		t.Fatalf("unexpected codex config: %+v", cfg)
	}
	if cfg.Session == nil || string(cfg.Session.Data) != `{"session_id":"sess-1"}` {
		t.Fatalf("session metadata not preserved: %+v", cfg.Session)
	}
}

func TestBuildConfigRejectsUnsupportedCodexSharedKnobs(t *testing.T) {
	_, err := BuildConfig("codex", "gpt-5", "/repo", RuntimeConfig{
		Shared: SharedRuntimeConfig{
			AllowedTools:    []string{"Bash"},
			StrictMCPConfig: true,
			MaxBudgetUSD:    12.5,
			MaxTurns:        4,
		},
	}, nil)
	if err == nil {
		t.Fatal("BuildConfig() error = nil, want unsupported shared knob error")
	}
	for _, want := range []string{"shared.allowed_tools", "shared.strict_mcp_config", "shared.max_budget_usd", "shared.max_turns"} {
		if strings.Contains(err.Error(), want) {
			return
		}
	}
	t.Fatalf("BuildConfig() error = %q, want unsupported shared knob context", err)
}

func TestBuildConfigLoadsClaudePromptFiles(t *testing.T) {
	root := t.TempDir()
	systemPath := filepath.Join(root, "system.txt")
	appendPath := filepath.Join(root, "append.txt")
	if err := os.WriteFile(systemPath, []byte("base prompt"), 0o644); err != nil {
		t.Fatalf("write system prompt: %v", err)
	}
	if err := os.WriteFile(appendPath, []byte("append prompt"), 0o644); err != nil {
		t.Fatalf("write append prompt: %v", err)
	}

	cfg, err := BuildConfig("claude", "sonnet", root, RuntimeConfig{
		Shared: SharedRuntimeConfig{
			AppendSystemPrompt: "inline appendix",
		},
		Providers: RuntimeProviderConfig{
			Claude: &ClaudeRuntimeConfig{
				SystemPromptFile:       "system.txt",
				AppendSystemPromptFile: "append.txt",
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("BuildConfig(): %v", err)
	}

	want := "base prompt\n\ninline appendix\n\nappend prompt"
	if cfg.SystemPrompt != want {
		t.Fatalf("SystemPrompt = %q, want %q", cfg.SystemPrompt, want)
	}
}

func TestPrepareRuntimeWritesClaudeAssetsAndRestoresScope(t *testing.T) {
	root := t.TempDir()
	if err := claudeconfig.SaveProjectSettings(root, claudeconfig.NewSettings()); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if err := claudeconfig.SaveProjectMCPConfig(root, claudeconfig.NewMCPConfig()); err != nil {
		t.Fatalf("save mcp: %v", err)
	}

	prepared, err := PrepareRuntime(context.Background(), PrepareRequest{
		Provider: "claude",
		WorkDir:  root,
		RuntimeConfig: RuntimeConfig{
			Shared: SharedRuntimeConfig{
				MCPServers: map[string]MCPServerConfig{
					"docs": {Command: "npx", Args: []string{"demo"}},
				},
				Env: map[string]string{"LLMKIT": "1"},
			},
			Providers: RuntimeProviderConfig{
				Claude: &ClaudeRuntimeConfig{
					SkillRefs: []string{"review"},
					Hooks: map[string][]HookMatcher{
						"Stop": {{
							Matcher: "*",
							Hooks: []HookEntry{{
								Type:    "command",
								Command: "bash {{hook:stop.sh}}",
							}},
						}},
					},
				},
			},
		},
		Assets: &RuntimeAssets{
			Skills: map[string]SkillAsset{
				"review": {
					Name:        "review",
					Description: "review skill",
					Content:     "Use this skill.",
				},
			},
			HookScripts: map[string]string{
				"stop.sh": "#!/bin/sh\necho stop\n",
			},
		},
	})
	if err != nil {
		t.Fatalf("PrepareRuntime: %v", err)
	}

	hookPath := filepath.Join(root, ".claude", "hooks", "stop.sh")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("expected hook script written: %v", err)
	}
	skillPath := filepath.Join(root, ".claude", "skills", "review", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("expected skill written: %v", err)
	}

	if err := prepared.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(skillPath)); !os.IsNotExist(err) {
		t.Fatalf("expected skill dir removed, got %v", err)
	}

	settings, err := claudeconfig.LoadProjectSettings(root)
	if err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if _, ok := settings.Env["LLMKIT"]; ok {
		t.Fatal("expected scoped env to be restored")
	}
}
