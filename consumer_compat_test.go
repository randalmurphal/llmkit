package llmkit_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/randalmurphal/llmkit/v2/claude"
	claudesession "github.com/randalmurphal/llmkit/v2/claude/session"
	"github.com/randalmurphal/llmkit/v2/claudeconfig"
	"github.com/randalmurphal/llmkit/v2/codex"
	codexsession "github.com/randalmurphal/llmkit/v2/codex/session"
	"github.com/randalmurphal/llmkit/v2/codexcontract"
)

func TestOrcConsumerSurfaceCompiles(t *testing.T) {
	_ = claude.NewClaudeCLI(
		claude.WithResume("session-123"),
		claude.WithSessionID("session-123"),
		claude.WithSystemPrompt("system"),
		claude.WithAllowedTools([]string{"Read"}),
		claude.WithDisallowedTools([]string{"Bash"}),
		claude.WithJSONSchema(`{"type":"object"}`),
		claude.WithMCPServers(map[string]claude.MCPServerConfig{
			"local": {
				Command: "mcp-server",
				Args:    []string{"--stdio"},
			},
		}),
	)

	_ = claude.CompletionRequest{
		Messages:   []claude.Message{{Role: claude.RoleUser, Content: "prompt"}},
		JSONSchema: `{"type":"object"}`,
	}

	_ = codex.NewCodexCLI(
		codex.WithSessionID("last"),
		codex.WithReasoningEffort("medium"),
		codex.WithOutputSchema("/tmp/schema.json"),
	)

	_ = codex.CompletionRequest{
		Messages:         []codex.Message{{Role: codex.RoleUser, Content: "prompt"}},
		OutputSchemaPath: "/tmp/schema.json",
	}

	var (
		_ = claudeconfig.ParseSkillMD
		_ = claudeconfig.WriteSkillMD
		_ = claudeconfig.DiscoverSkills
		_ = claudeconfig.ListSkillResources
		_ = claudeconfig.LoadSettings
		_ = claudeconfig.LoadGlobalSettings
		_ = claudeconfig.LoadProjectSettings
		_ = claudeconfig.SaveProjectSettings
		_ = claudeconfig.ValidHookEvents
		_ = claudeconfig.LoadClaudeMDHierarchy
		_ = claudeconfig.LoadProjectClaudeMD
		_ = claudeconfig.SaveProjectClaudeMD
		_ = claudeconfig.AvailableTools
		_ = claudeconfig.ToolsByCategory
		_ = claudeconfig.GetTool
		_ = claudeconfig.ToolCategories
		_ = claudeconfig.NewAgentService
		_ = claudeconfig.NewScriptService
		_ = claudeconfig.ParsePluginJSON
		_ = claudeconfig.DiscoverPlugins
		_ = claudeconfig.NewPluginService
		_ = claudeconfig.GlobalPluginsDir
		_ = claudeconfig.ProjectPluginsDir
		_ = claudeconfig.LoadProjectMCPConfig
		_ = claudeconfig.SaveProjectMCPConfig
	)
}

func TestHerdingLlamasConsumerSurfaceCompiles(t *testing.T) {
	_ = claudesession.NewManager
	_ = codexsession.NewManager

	claudeOpts := []claudesession.SessionOption{
		claudesession.WithWorkdir("/tmp/project"),
		claudesession.WithPermissions(true),
		claudesession.WithSystemPrompt("system"),
		claudesession.WithModel("claude-sonnet-4-20250514"),
		claudesession.WithEffort("high"),
	}
	if len(claudeOpts) == 0 {
		t.Fatal("expected Claude session options to compile")
	}

	codexOpts := []codexsession.SessionOption{
		codexsession.WithWorkdir("/tmp/project"),
		codexsession.WithFullAuto(),
		codexsession.WithSystemPrompt("system"),
		codexsession.WithModel("gpt-5.4"),
		codexsession.WithReasoningEffort("high"),
	}
	if len(codexOpts) == 0 {
		t.Fatal("expected Codex session options to compile")
	}

	cfg := codexcontract.HookConfig{
		Hooks: map[string][]codexcontract.HookMatcher{
			string(codexcontract.HookStop): {
				{
					Matcher: "herdingllamas-stop-hook",
					Hooks: []codexcontract.HookEntry{
						{
							Type:    "command",
							Command: "/tmp/hook.sh",
							Timeout: 10,
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal codex hook config: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected codex hook config JSON")
	}
}

func TestClaudeConfigProjectSettingsRoundTripForConsumerUsage(t *testing.T) {
	dir := t.TempDir()

	settings := &claudeconfig.Settings{}
	settings.AddHook(claudeconfig.HookStop, claudeconfig.Hook{
		Matcher: "compat-stop-hook",
		Hooks: []claudeconfig.HookEntry{
			{Type: "command", Command: "/tmp/hook.sh", Timeout: 10},
		},
	})

	if err := claudeconfig.SaveProjectSettings(dir, settings); err != nil {
		t.Fatalf("save project settings: %v", err)
	}

	loaded, err := claudeconfig.LoadProjectSettings(dir)
	if err != nil {
		t.Fatalf("load project settings: %v", err)
	}

	stopHooks := loaded.Hooks[string(claudeconfig.HookStop)]
	if len(stopHooks) != 1 || stopHooks[0].Matcher != "compat-stop-hook" {
		t.Fatalf("unexpected stop hooks after round trip: %+v", stopHooks)
	}

	if _, err := filepath.Abs(dir); err != nil {
		t.Fatalf("abs project dir: %v", err)
	}
}
