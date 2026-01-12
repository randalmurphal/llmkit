package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePluginJSON(t *testing.T) {
	// Create a temporary plugin structure
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	claudePluginDir := filepath.Join(pluginDir, ".claude-plugin")
	commandsDir := filepath.Join(pluginDir, "commands")

	if err := os.MkdirAll(claudePluginDir, 0755); err != nil {
		t.Fatalf("create .claude-plugin dir: %v", err)
	}
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("create commands dir: %v", err)
	}

	// Write plugin.json
	pluginJSON := `{
  "name": "test-plugin",
  "description": "A test plugin",
  "author": {
    "name": "Test Author",
    "email": "test@example.com",
    "url": "https://example.com"
  },
  "homepage": "https://example.com/plugin",
  "keywords": ["test", "example"]
}`
	if err := os.WriteFile(filepath.Join(claudePluginDir, "plugin.json"), []byte(pluginJSON), 0644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}

	// Write a command file
	commandMD := `---
description: "Test command description"
argument-hint: "[ARG]"
---

# Test Command

This is a test command.
`
	if err := os.WriteFile(filepath.Join(commandsDir, "test.md"), []byte(commandMD), 0644); err != nil {
		t.Fatalf("write command file: %v", err)
	}

	// Parse the plugin
	plugin, err := ParsePluginJSON(pluginDir)
	if err != nil {
		t.Fatalf("ParsePluginJSON: %v", err)
	}

	// Verify fields
	if plugin.Name != "test-plugin" {
		t.Errorf("Name = %q, want %q", plugin.Name, "test-plugin")
	}
	if plugin.Description != "A test plugin" {
		t.Errorf("Description = %q, want %q", plugin.Description, "A test plugin")
	}
	if plugin.Author.Name != "Test Author" {
		t.Errorf("Author.Name = %q, want %q", plugin.Author.Name, "Test Author")
	}
	if plugin.Author.Email != "test@example.com" {
		t.Errorf("Author.Email = %q, want %q", plugin.Author.Email, "test@example.com")
	}
	if plugin.Homepage != "https://example.com/plugin" {
		t.Errorf("Homepage = %q, want %q", plugin.Homepage, "https://example.com/plugin")
	}
	if len(plugin.Keywords) != 2 {
		t.Errorf("len(Keywords) = %d, want 2", len(plugin.Keywords))
	}
	if !plugin.HasCommands {
		t.Error("HasCommands = false, want true")
	}
	if len(plugin.Commands) != 1 {
		t.Errorf("len(Commands) = %d, want 1", len(plugin.Commands))
	}
	if len(plugin.Commands) > 0 {
		cmd := plugin.Commands[0]
		if cmd.Name != "test" {
			t.Errorf("Command.Name = %q, want %q", cmd.Name, "test")
		}
		if cmd.Description != "Test command description" {
			t.Errorf("Command.Description = %q, want %q", cmd.Description, "Test command description")
		}
		if cmd.ArgumentHint != "[ARG]" {
			t.Errorf("Command.ArgumentHint = %q, want %q", cmd.ArgumentHint, "[ARG]")
		}
	}
}

func TestParsePluginJSON_MinimalPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "minimal")
	claudePluginDir := filepath.Join(pluginDir, ".claude-plugin")

	if err := os.MkdirAll(claudePluginDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	// Minimal plugin.json
	pluginJSON := `{"name": "minimal", "description": "Minimal plugin"}`
	if err := os.WriteFile(filepath.Join(claudePluginDir, "plugin.json"), []byte(pluginJSON), 0644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}

	plugin, err := ParsePluginJSON(pluginDir)
	if err != nil {
		t.Fatalf("ParsePluginJSON: %v", err)
	}

	if plugin.Name != "minimal" {
		t.Errorf("Name = %q, want %q", plugin.Name, "minimal")
	}
	if plugin.HasCommands {
		t.Error("HasCommands = true, want false (no commands dir)")
	}
}

func TestDiscoverPlugins(t *testing.T) {
	// Create a .claude directory with plugins
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")

	// Create two plugins
	for _, name := range []string{"plugin-a", "plugin-b"} {
		pluginDir := filepath.Join(pluginsDir, name, ".claude-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatalf("create plugin dir: %v", err)
		}
		pluginJSON := `{"name": "` + name + `", "description": "Plugin ` + name + `"}`
		if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0644); err != nil {
			t.Fatalf("write plugin.json: %v", err)
		}
	}

	// Create an invalid directory (no plugin.json)
	invalidDir := filepath.Join(pluginsDir, "invalid")
	if err := os.MkdirAll(invalidDir, 0755); err != nil {
		t.Fatalf("create invalid dir: %v", err)
	}

	plugins, err := DiscoverPlugins(claudeDir)
	if err != nil {
		t.Fatalf("DiscoverPlugins: %v", err)
	}

	if len(plugins) != 2 {
		t.Errorf("len(plugins) = %d, want 2", len(plugins))
	}

	// Should be sorted by name
	if len(plugins) >= 2 {
		if plugins[0].Name != "plugin-a" {
			t.Errorf("plugins[0].Name = %q, want %q", plugins[0].Name, "plugin-a")
		}
		if plugins[1].Name != "plugin-b" {
			t.Errorf("plugins[1].Name = %q, want %q", plugins[1].Name, "plugin-b")
		}
	}
}

func TestDiscoverPlugins_NoPluginsDir(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")

	// No plugins directory exists
	plugins, err := DiscoverPlugins(claudeDir)
	if err != nil {
		t.Fatalf("DiscoverPlugins: %v", err)
	}

	if plugins != nil && len(plugins) != 0 {
		t.Errorf("len(plugins) = %d, want 0", len(plugins))
	}
}

func TestDiscoverPluginsWithEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")

	// Create a plugin
	pluginDir := filepath.Join(pluginsDir, "test-plugin", ".claude-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	pluginJSON := `{"name": "test-plugin", "description": "Test"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}

	// Create settings with plugin enabled
	settings := &Settings{
		EnabledPlugins: map[string]bool{
			"test-plugin": true,
		},
	}

	plugins, err := DiscoverPluginsWithEnabled(claudeDir, settings)
	if err != nil {
		t.Fatalf("DiscoverPluginsWithEnabled: %v", err)
	}

	if len(plugins) != 1 {
		t.Fatalf("len(plugins) = %d, want 1", len(plugins))
	}

	if !plugins[0].Enabled {
		t.Error("plugin.Enabled = false, want true")
	}
}

func TestPluginValidate(t *testing.T) {
	tests := []struct {
		name    string
		plugin  Plugin
		wantErr bool
	}{
		{
			name:    "valid plugin",
			plugin:  Plugin{Name: "test", Description: "Test plugin"},
			wantErr: false,
		},
		{
			name:    "missing name",
			plugin:  Plugin{Description: "Test plugin"},
			wantErr: true,
		},
		{
			name:    "missing description",
			plugin:  Plugin{Name: "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPluginInfo(t *testing.T) {
	plugin := &Plugin{
		Name:        "test",
		Description: "Test plugin",
		Author:      PluginAuthor{Name: "Author"},
		Path:        "/path/to/plugin",
		Scope:       PluginScopeProject,
		Enabled:     true,
		Version:     "1.0.0",
		HasCommands: true,
		Commands:    []PluginCommand{{Name: "cmd1"}, {Name: "cmd2"}},
	}

	info := plugin.Info()

	if info.Name != "test" {
		t.Errorf("info.Name = %q, want %q", info.Name, "test")
	}
	if info.Author != "Author" {
		t.Errorf("info.Author = %q, want %q", info.Author, "Author")
	}
	if info.Scope != PluginScopeProject {
		t.Errorf("info.Scope = %q, want %q", info.Scope, PluginScopeProject)
	}
	if !info.Enabled {
		t.Error("info.Enabled = false, want true")
	}
	if info.CommandCount != 2 {
		t.Errorf("info.CommandCount = %d, want 2", info.CommandCount)
	}
}

func TestPluginKey(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      string
	}{
		{"plugin", "", "plugin"},
		{"plugin", "official", "plugin@official"},
		{"my-plugin", "marketplace", "my-plugin@marketplace"},
	}

	for _, tt := range tests {
		got := PluginKey(tt.name, tt.namespace)
		if got != tt.want {
			t.Errorf("PluginKey(%q, %q) = %q, want %q", tt.name, tt.namespace, got, tt.want)
		}
	}
}

func TestParsePluginKey(t *testing.T) {
	tests := []struct {
		key           string
		wantName      string
		wantNamespace string
	}{
		{"plugin", "plugin", ""},
		{"plugin@official", "plugin", "official"},
		{"my-plugin@marketplace", "my-plugin", "marketplace"},
		{"complex@name@space", "complex@name", "space"},
	}

	for _, tt := range tests {
		name, namespace := ParsePluginKey(tt.key)
		if name != tt.wantName {
			t.Errorf("ParsePluginKey(%q) name = %q, want %q", tt.key, name, tt.wantName)
		}
		if namespace != tt.wantNamespace {
			t.Errorf("ParsePluginKey(%q) namespace = %q, want %q", tt.key, namespace, tt.wantNamespace)
		}
	}
}

func TestPluginService_List(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project .claude/plugins structure
	projectPluginsDir := filepath.Join(tmpDir, ".claude", "plugins")
	pluginDir := filepath.Join(projectPluginsDir, "project-plugin", ".claude-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	pluginJSON := `{"name": "project-plugin", "description": "Project plugin"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}

	svc, err := NewPluginService(tmpDir)
	if err != nil {
		t.Fatalf("NewPluginService: %v", err)
	}

	// Override global dir to temp location to avoid using real global plugins
	svc.globalDir = filepath.Join(tmpDir, "global-plugins")

	infos, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(infos) != 1 {
		t.Errorf("len(infos) = %d, want 1", len(infos))
	}

	if len(infos) > 0 && infos[0].Name != "project-plugin" {
		t.Errorf("infos[0].Name = %q, want %q", infos[0].Name, "project-plugin")
	}
}

func TestPluginService_SetEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("create .claude dir: %v", err)
	}

	svc, err := NewPluginService(tmpDir)
	if err != nil {
		t.Fatalf("NewPluginService: %v", err)
	}

	// Enable a plugin
	if err := svc.SetEnabled("test-plugin", PluginScopeProject, true); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}

	// Verify settings were written
	settings, err := LoadProjectSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectSettings: %v", err)
	}

	if !settings.EnabledPlugins["test-plugin"] {
		t.Error("plugin not enabled in settings")
	}

	// Disable the plugin
	if err := svc.SetEnabled("test-plugin", PluginScopeProject, false); err != nil {
		t.Fatalf("SetEnabled(false): %v", err)
	}

	settings, _ = LoadProjectSettings(tmpDir)
	if settings.EnabledPlugins["test-plugin"] {
		t.Error("plugin still enabled after disable")
	}
}
