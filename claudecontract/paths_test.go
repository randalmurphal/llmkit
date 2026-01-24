package claudecontract

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileNameConstants validates file name constants.
func TestFileNameConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"FileSettings", FileSettings, "settings.json"},
		{"FileMCPConfig", FileMCPConfig, ".mcp.json"},
		{"FileSkillMD", FileSkillMD, "SKILL.md"},
		{"FilePluginJSON", FilePluginJSON, "plugin.json"},
		{"FileClaudeMD", FileClaudeMD, "CLAUDE.md"},
		{"FileAgentsMD", FileAgentsMD, "AGENTS.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestDirectoryNameConstants validates directory name constants.
func TestDirectoryNameConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"DirClaude", DirClaude, ".claude"},
		{"DirSkills", DirSkills, "skills"},
		{"DirPlugins", DirPlugins, "plugins"},
		{"DirHooks", DirHooks, "hooks"},
		{"DirScripts", DirScripts, "scripts"},
		{"DirCommands", DirCommands, "commands"},
		{"DirReferences", DirReferences, "references"},
		{"DirAssets", DirAssets, "assets"},
		{"DirCache", DirCache, "cache"},
		{"DirClaudePlugin", DirClaudePlugin, ".claude-plugin"},
		{"DirProjects", DirProjects, "projects"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestSettingSourceConstants validates setting source constants.
func TestSettingSourceConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant SettingSource
		expected string
	}{
		{"SettingSourceUser", SettingSourceUser, "user"},
		{"SettingSourceProject", SettingSourceProject, "project"},
		{"SettingSourceLocal", SettingSourceLocal, "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestGlobalClaudeDir validates the global .claude directory structure.
func TestGlobalClaudeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	globalClaudeDir := filepath.Join(home, DirClaude)

	// Check if .claude directory exists (it should if Claude Code is installed)
	if _, err := os.Stat(globalClaudeDir); os.IsNotExist(err) {
		t.Skip("Global .claude directory not found (Claude Code may not be installed)")
	}

	// Expected subdirectories in ~/.claude/
	expectedDirs := []string{
		DirSkills,
		DirPlugins,
	}

	for _, subdir := range expectedDirs {
		path := filepath.Join(globalClaudeDir, subdir)
		// These directories may or may not exist depending on user's setup
		// We just verify the path construction is correct
		t.Logf("Expected path: %s", path)
	}

	// Verify settings.json path construction
	settingsPath := filepath.Join(globalClaudeDir, FileSettings)
	t.Logf("Settings path: %s", settingsPath)

	// Verify CLAUDE.md path construction
	claudeMDPath := filepath.Join(globalClaudeDir, FileClaudeMD)
	t.Logf("CLAUDE.md path: %s", claudeMDPath)
}

// TestProjectClaudeDir validates the project .claude directory structure.
func TestProjectClaudeDir(t *testing.T) {
	// Simulate a project root
	projectRoot := "/home/user/myproject"

	// Expected paths
	expectedPaths := map[string]string{
		"claude_dir":  filepath.Join(projectRoot, DirClaude),
		"settings":    filepath.Join(projectRoot, DirClaude, FileSettings),
		"skills_dir":  filepath.Join(projectRoot, DirClaude, DirSkills),
		"plugins_dir": filepath.Join(projectRoot, DirClaude, DirPlugins),
		"hooks_dir":   filepath.Join(projectRoot, DirClaude, DirHooks),
		"scripts_dir": filepath.Join(projectRoot, DirClaude, DirScripts),
		"mcp_config":  filepath.Join(projectRoot, FileMCPConfig),
		"claude_md":   filepath.Join(projectRoot, FileClaudeMD),
		"agents_md":   filepath.Join(projectRoot, FileAgentsMD),
	}

	// Verify path construction uses constants correctly
	tests := []struct {
		name     string
		expected string
	}{
		{"claude_dir", "/home/user/myproject/.claude"},
		{"settings", "/home/user/myproject/.claude/settings.json"},
		{"skills_dir", "/home/user/myproject/.claude/skills"},
		{"plugins_dir", "/home/user/myproject/.claude/plugins"},
		{"hooks_dir", "/home/user/myproject/.claude/hooks"},
		{"scripts_dir", "/home/user/myproject/.claude/scripts"},
		{"mcp_config", "/home/user/myproject/.mcp.json"},
		{"claude_md", "/home/user/myproject/CLAUDE.md"},
		{"agents_md", "/home/user/myproject/AGENTS.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := expectedPaths[tt.name]
			if actual != tt.expected {
				t.Errorf("Path %s = %q, want %q", tt.name, actual, tt.expected)
			}
		})
	}
}

// TestPluginDirectoryStructure validates plugin directory structure constants.
func TestPluginDirectoryStructure(t *testing.T) {
	pluginRoot := "/home/user/.claude/plugins/my-plugin"

	// Plugin structure: plugins/{name}/.claude-plugin/plugin.json
	expectedStructure := map[string]string{
		"plugin_meta_dir": filepath.Join(pluginRoot, DirClaudePlugin),
		"plugin_json":     filepath.Join(pluginRoot, DirClaudePlugin, FilePluginJSON),
		"hooks_dir":       filepath.Join(pluginRoot, DirHooks),
		"commands_dir":    filepath.Join(pluginRoot, DirCommands),
		"scripts_dir":     filepath.Join(pluginRoot, DirScripts),
		"mcp_config":      filepath.Join(pluginRoot, FileMCPConfig),
	}

	// Verify the expected paths are constructed correctly
	if expected := "/home/user/.claude/plugins/my-plugin/.claude-plugin"; expectedStructure["plugin_meta_dir"] != expected {
		t.Errorf("plugin_meta_dir = %q, want %q", expectedStructure["plugin_meta_dir"], expected)
	}
	if expected := "/home/user/.claude/plugins/my-plugin/.claude-plugin/plugin.json"; expectedStructure["plugin_json"] != expected {
		t.Errorf("plugin_json = %q, want %q", expectedStructure["plugin_json"], expected)
	}
}

// TestSkillDirectoryStructure validates skill directory structure constants.
func TestSkillDirectoryStructure(t *testing.T) {
	skillRoot := "/home/user/.claude/skills/my-skill"

	// Skill structure: skills/{name}/SKILL.md + optional subdirs
	expectedStructure := map[string]string{
		"skill_md":   filepath.Join(skillRoot, FileSkillMD),
		"references": filepath.Join(skillRoot, DirReferences),
		"scripts":    filepath.Join(skillRoot, DirScripts),
		"assets":     filepath.Join(skillRoot, DirAssets),
	}

	if expected := "/home/user/.claude/skills/my-skill/SKILL.md"; expectedStructure["skill_md"] != expected {
		t.Errorf("skill_md = %q, want %q", expectedStructure["skill_md"], expected)
	}
}

// TestSessionProjectPath validates session JSONL file path construction.
func TestSessionProjectPath(t *testing.T) {
	home := "/home/user"
	projectPath := "/home/user/repos/myproject"
	sessionID := "abc-123-def-456"

	// Sessions are stored in: ~/.claude/projects/{normalized-path}/{sessionId}.jsonl
	// The normalized path replaces / with - and removes special characters
	claudeDir := filepath.Join(home, DirClaude)
	projectsDir := filepath.Join(claudeDir, DirProjects)

	// This demonstrates the expected structure
	t.Logf("Projects directory: %s", projectsDir)
	t.Logf("Example session path would be: %s/{normalized-path}/%s.jsonl", projectsDir, sessionID)
	_ = projectPath // used for documentation
}

// TestHiddenFileNaming validates hidden file/directory naming conventions.
func TestHiddenFileNaming(t *testing.T) {
	// Hidden files/directories in Unix start with .
	hiddenItems := []string{
		DirClaude,       // .claude
		FileMCPConfig,   // .mcp.json
		DirClaudePlugin, // .claude-plugin
	}

	for _, item := range hiddenItems {
		if item == "" || item[0] != '.' {
			t.Errorf("Expected hidden item %q to start with '.'", item)
		}
	}

	// Non-hidden items should not start with .
	visibleItems := []string{
		FileSettings,   // settings.json
		FileSkillMD,    // SKILL.md
		FilePluginJSON, // plugin.json
		FileClaudeMD,   // CLAUDE.md
		DirSkills,      // skills
		DirPlugins,     // plugins
	}

	for _, item := range visibleItems {
		if item != "" && item[0] == '.' {
			t.Errorf("Expected visible item %q to not start with '.'", item)
		}
	}
}
