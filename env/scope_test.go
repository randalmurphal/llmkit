package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/randalmurphal/llmkit/v2/claudeconfig"
	"github.com/randalmurphal/llmkit/v2/contract"
	"github.com/randalmurphal/llmkit/v2/codexconfig"
)

func TestClaudeScopeRestorePreservesManualEdits(t *testing.T) {
	root := t.TempDir()
	settings := claudeconfig.NewSettings()
	settings.Env["BASE"] = "1"
	if err := claudeconfig.SaveProjectSettings(root, settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if err := claudeconfig.SaveProjectMCPConfig(root, claudeconfig.NewMCPConfig()); err != nil {
		t.Fatalf("save mcp: %v", err)
	}

	scope, err := NewScope("claude", root, ScopeConfig{
		Hooks: map[string][]Hook{
			"Stop": {{
				Matcher: "*",
				Type:    "command",
				Command: "echo stop",
			}},
		},
		MCPServers: map[string]contract.MCPServerConfig{
			"demo": {Command: "npx", Args: []string{"demo"}},
		},
		Env: map[string]string{"LLMKIT": "1"},
	})
	if err != nil {
		t.Fatalf("NewScope: %v", err)
	}

	project, err := claudeconfig.LoadProjectSettings(root)
	if err != nil {
		t.Fatalf("load project settings: %v", err)
	}
	project.Env["LLMKIT"] = "manual-change"
	project.AddHook(claudeconfig.HookStop, claudeconfig.Hook{
		Matcher: "manual",
		Hooks: []claudeconfig.HookEntry{{
			Type:    "command",
			Command: "echo manual",
		}},
	})
	if err := claudeconfig.SaveProjectSettings(root, project); err != nil {
		t.Fatalf("save project settings: %v", err)
	}
	mcp, err := claudeconfig.LoadProjectMCPConfig(root)
	if err != nil {
		t.Fatalf("load project mcp: %v", err)
	}
	mcp.MCPServers["manual"] = &claudeconfig.MCPServer{Command: "manual"}
	if err := claudeconfig.SaveProjectMCPConfig(root, mcp); err != nil {
		t.Fatalf("save project mcp: %v", err)
	}

	if err := scope.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	project, err = claudeconfig.LoadProjectSettings(root)
	if err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if project.Env["LLMKIT"] != "manual-change" {
		t.Fatalf("expected manual env override preserved, got %q", project.Env["LLMKIT"])
	}
	stopHooks := project.GetHooks(claudeconfig.HookStop)
	if len(stopHooks) != 1 || stopHooks[0].Matcher != "manual" {
		t.Fatalf("unexpected stop hooks after restore: %#v", stopHooks)
	}
	mcp, err = claudeconfig.LoadProjectMCPConfig(root)
	if err != nil {
		t.Fatalf("reload mcp: %v", err)
	}
	if _, ok := mcp.MCPServers["demo"]; ok {
		t.Fatal("expected llmkit MCP server to be removed")
	}
	if _, ok := mcp.MCPServers["manual"]; !ok {
		t.Fatal("expected manual MCP server preserved")
	}
}

func TestCodexScopeRestorePreservesManualEdits(t *testing.T) {
	root := t.TempDir()
	configPath := codexconfig.ProjectConfigPath(root)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("experimental_feature = true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := codexconfig.SaveHooks(root, &codexconfig.HookConfig{Hooks: map[string][]codexconfig.HookMatcher{}}); err != nil {
		t.Fatalf("save hooks: %v", err)
	}

	scope, err := NewScope("codex", root, ScopeConfig{
		Hooks: map[string][]Hook{
			string(codexconfig.HookStop): {{
				Matcher: "*",
				Type:    "command",
				Command: "echo stop",
			}},
		},
		MCPServers: map[string]contract.MCPServerConfig{
			"docs": {URL: "https://example.com/mcp", Type: "http"},
		},
	})
	if err != nil {
		t.Fatalf("NewScope: %v", err)
	}

	hooks, err := codexconfig.LoadHooks(root)
	if err != nil {
		t.Fatalf("load hooks: %v", err)
	}
	hooks.Hooks[string(codexconfig.HookStop)] = append(hooks.Hooks[string(codexconfig.HookStop)], codexconfig.HookMatcher{
		Matcher: "manual",
		Hooks: []codexconfig.HookEntry{{
			Type:    "command",
			Command: "echo manual",
		}},
	})
	if err := codexconfig.SaveHooks(root, hooks); err != nil {
		t.Fatalf("save hooks: %v", err)
	}
	cfg, err := codexconfig.LoadProjectConfig(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = map[string]codexconfig.MCPServer{}
	}
	cfg.MCPServers["manual"] = codexconfig.MCPServer{Command: "manual"}
	if err := codexconfig.SaveProjectConfig(root, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	if err := scope.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	hooks, err = codexconfig.LoadHooks(root)
	if err != nil {
		t.Fatalf("reload hooks: %v", err)
	}
	matchers := hooks.Hooks[string(codexconfig.HookStop)]
	if len(matchers) != 1 || matchers[0].Matcher != "manual" {
		t.Fatalf("unexpected hooks after restore: %#v", matchers)
	}
	cfg, err = codexconfig.LoadProjectConfig(root)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if _, ok := cfg.MCPServers["docs"]; ok {
		t.Fatal("expected llmkit MCP server removed")
	}
	if _, ok := cfg.MCPServers["manual"]; !ok {
		t.Fatal("expected manual MCP server preserved")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(data), "experimental_feature = true") {
		t.Fatalf("expected unknown codex config key preserved, got:\n%s", string(data))
	}
}

func TestRecoverOrphanedScopes(t *testing.T) {
	root := t.TempDir()
	settings := claudeconfig.NewSettings()
	settings.Env["LLMKIT"] = "1"
	settings.AddHook(claudeconfig.HookStop, claudeconfig.Hook{
		Matcher: "orphan",
		Hooks: []claudeconfig.HookEntry{{
			Type:    "command",
			Command: "echo orphan",
		}},
	})
	if err := claudeconfig.SaveProjectSettings(root, settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	mcp := claudeconfig.NewMCPConfig()
	mcp.MCPServers["orphan"] = &claudeconfig.MCPServer{Command: "npx"}
	if err := claudeconfig.SaveProjectMCPConfig(root, mcp); err != nil {
		t.Fatalf("save mcp: %v", err)
	}
	reg := &scopeRegistry{
		Scopes: map[string]scopeRecord{
			"orphan": {
				Tag:      "orphan",
				Provider: "claude",
				WorkDir:  root,
				PID:      999999,
				Hooks: map[string][]Hook{
					"Stop": {{
						Matcher: "orphan",
						Type:    "command",
						Command: "echo orphan",
					}},
				},
				MCPServers: map[string]contract.MCPServerConfig{
					"orphan": {Command: "npx"},
				},
				Env: map[string]string{"LLMKIT": "1"},
			},
		},
	}
	if err := saveRegistry(root, reg); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	scope, err := NewScope("claude", root, ScopeConfig{
		RecoverOrphans: true,
	})
	if err != nil {
		t.Fatalf("NewScope: %v", err)
	}
	defer scope.Restore()

	project, err := claudeconfig.LoadProjectSettings(root)
	if err != nil {
		t.Fatalf("load project settings: %v", err)
	}
	if _, ok := project.Env["LLMKIT"]; ok {
		t.Fatal("expected orphan env cleaned up")
	}
}

func TestTempFileCleanup(t *testing.T) {
	file, err := TempFile("", "llmkit-test-*")
	if err != nil {
		t.Fatalf("TempFile: %v", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected temp file to exist: %v", err)
	}
	Cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected temp file removed, got %v", err)
	}
}

func TestBackupSettings(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte("{\"mcpServers\":{}}\n"), 0o644); err != nil {
		t.Fatalf("write mcp: %v", err)
	}
	scope, err := NewScope("claude", root, ScopeConfig{
		BackupSettings: true,
		Env:            map[string]string{"LLMKIT": "1"},
	})
	if err != nil {
		t.Fatalf("NewScope: %v", err)
	}
	defer scope.Restore()

	for _, suffix := range []string{
		filepath.Join(root, ".claude", "settings.json.bak"),
		filepath.Join(root, ".mcp.json.bak"),
	} {
		if _, err := os.Stat(suffix); err != nil {
			t.Fatalf("expected backup file %s: %v", suffix, err)
		}
	}
}
