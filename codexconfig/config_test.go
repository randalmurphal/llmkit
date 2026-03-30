package codexconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectConfigRoundTrip(t *testing.T) {
	root := t.TempDir()
	cfg := &ConfigFile{
		ModelInstructionsFile:       "docs/agent.md",
		ProjectDocFallbackFilenames: []string{"TEAM_AGENTS.md"},
		ProjectDocMaxBytes:          4096,
		MCPServers: map[string]MCPServer{
			"docs": {
				URL:  "https://developers.openai.com/mcp",
				Type: "streamable_http",
			},
		},
		Skills: SkillsSettings{
			Config: []SkillToggle{{
				Path:    "/tmp/skill/SKILL.md",
				Enabled: false,
			}},
		},
		Agents: AgentsSettings{
			MaxThreads: 4,
			MaxDepth:   1,
		},
	}

	if err := SaveProjectConfig(root, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	loaded, err := LoadProjectConfig(root)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if loaded.ModelInstructionsFile != "docs/agent.md" {
		t.Fatalf("ModelInstructionsFile = %q", loaded.ModelInstructionsFile)
	}
	if loaded.MCPServers["docs"].URL != "https://developers.openai.com/mcp" {
		t.Fatalf("MCP URL = %q", loaded.MCPServers["docs"].URL)
	}
	enabled, ok := loaded.SkillEnabled("/tmp/skill/SKILL.md")
	if !ok || enabled {
		t.Fatalf("SkillEnabled = %v, %v; want false, true", enabled, ok)
	}
	if loaded.Agents.MaxThreads != 4 {
		t.Fatalf("Agents.MaxThreads = %d", loaded.Agents.MaxThreads)
	}
}

func TestConfigPreservesUnknownTopLevelFields(t *testing.T) {
	root := t.TempDir()
	path := ProjectConfigPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("experimental_feature = true\n[mcp_servers.docs]\nurl = \"https://example.com\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadProjectConfig(root)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	cfg.ProjectDocMaxBytes = 4096
	if err := SaveProjectConfig(root, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "experimental_feature = true") {
		t.Fatalf("expected unknown top-level key preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "project_doc_max_bytes = 4096") {
		t.Fatalf("expected known key written, got:\n%s", content)
	}
}
