package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettings(t *testing.T) {
	s := NewSettings()

	assert.NotNil(t, s.Env)
	assert.NotNil(t, s.Hooks)
	assert.NotNil(t, s.EnabledPlugins)
	assert.NotNil(t, s.Extensions)
}

func TestSettings_Clone(t *testing.T) {
	original := &Settings{
		Env: map[string]string{
			"FOO": "bar",
		},
		Hooks: map[string][]Hook{
			"PreToolUse": {{Matcher: "Bash", Hooks: []HookEntry{{Type: "command", Command: "echo test"}}}},
		},
		EnabledPlugins: map[string]bool{
			"plugin1": true,
		},
		StatusLine: &StatusLine{
			Type:    "command",
			Command: "echo status",
		},
		Extensions: map[string]any{
			"custom": map[string]any{"key": "value"},
		},
	}

	clone := original.Clone()

	// Verify clone is equal
	assert.Equal(t, original.Env, clone.Env)
	assert.Equal(t, original.EnabledPlugins, clone.EnabledPlugins)
	assert.Equal(t, original.StatusLine.Type, clone.StatusLine.Type)

	// Verify clone is independent
	clone.Env["NEW"] = "value"
	assert.NotContains(t, original.Env, "NEW")
}

func TestSettings_Clone_Nil(t *testing.T) {
	var s *Settings
	assert.Nil(t, s.Clone())
}

func TestSettings_Merge(t *testing.T) {
	global := &Settings{
		Env: map[string]string{
			"GLOBAL": "value",
			"SHARED": "global",
		},
		Hooks: map[string][]Hook{
			"PreToolUse": {{Matcher: "Bash"}},
		},
		EnabledPlugins: map[string]bool{
			"global-plugin": true,
		},
		StatusLine: &StatusLine{
			Command: "global-status",
		},
	}

	project := &Settings{
		Env: map[string]string{
			"PROJECT": "value",
			"SHARED":  "project", // Override global
		},
		Hooks: map[string][]Hook{
			"PreToolUse":  {{Matcher: "Write"}}, // Add to existing
			"PostToolUse": {{Matcher: "Read"}},  // New hook
		},
		EnabledPlugins: map[string]bool{
			"project-plugin": true,
			"global-plugin":  false, // Override global
		},
		StatusLine: &StatusLine{
			Command: "project-status",
		},
	}

	merged := global.Merge(project)

	// Env: project overrides
	assert.Equal(t, "value", merged.Env["GLOBAL"])
	assert.Equal(t, "value", merged.Env["PROJECT"])
	assert.Equal(t, "project", merged.Env["SHARED"])

	// Hooks: merged
	assert.Len(t, merged.Hooks["PreToolUse"], 2)
	assert.Len(t, merged.Hooks["PostToolUse"], 1)

	// Plugins: project overrides
	assert.True(t, merged.EnabledPlugins["project-plugin"])
	assert.False(t, merged.EnabledPlugins["global-plugin"])

	// StatusLine: project overrides
	assert.Equal(t, "project-status", merged.StatusLine.Command)
}

func TestSettings_Merge_NilCases(t *testing.T) {
	settings := &Settings{
		Env: map[string]string{"KEY": "value"},
	}

	// Merge with nil project
	result := settings.Merge(nil)
	assert.Equal(t, "value", result.Env["KEY"])

	// Merge nil with settings
	var nilSettings *Settings
	result = nilSettings.Merge(settings)
	assert.Equal(t, "value", result.Env["KEY"])
}

func TestSettings_GetExtension(t *testing.T) {
	type CustomExt struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	s := &Settings{
		Extensions: map[string]any{
			"custom": map[string]any{
				"field1": "value",
				"field2": float64(42), // JSON numbers are float64
			},
		},
	}

	var ext CustomExt
	err := s.GetExtension("custom", &ext)
	require.NoError(t, err)

	assert.Equal(t, "value", ext.Field1)
	assert.Equal(t, 42, ext.Field2)
}

func TestSettings_GetExtension_NotExists(t *testing.T) {
	s := NewSettings()

	var ext struct{ Field string }
	err := s.GetExtension("nonexistent", &ext)
	require.NoError(t, err)
	assert.Empty(t, ext.Field)
}

func TestSettings_SetExtension(t *testing.T) {
	s := NewSettings()

	type CustomExt struct {
		Value string `json:"value"`
	}

	s.SetExtension("custom", CustomExt{Value: "test"})

	var retrieved CustomExt
	err := s.GetExtension("custom", &retrieved)
	require.NoError(t, err)
	assert.Equal(t, "test", retrieved.Value)
}

func TestSettings_Hooks(t *testing.T) {
	s := NewSettings()

	// Add hooks
	s.AddHook(HookPreToolUse, Hook{
		Matcher: "Bash",
		Hooks:   []HookEntry{{Type: "command", Command: "echo test"}},
	})
	s.AddHook(HookPreToolUse, Hook{
		Matcher: "Write",
		Hooks:   []HookEntry{{Type: "command", Command: "echo write"}},
	})

	hooks := s.GetHooks(HookPreToolUse)
	assert.Len(t, hooks, 2)

	// Remove hook
	removed := s.RemoveHook(HookPreToolUse, "Bash")
	assert.True(t, removed)

	hooks = s.GetHooks(HookPreToolUse)
	assert.Len(t, hooks, 1)
	assert.Equal(t, "Write", hooks[0].Matcher)

	// Remove non-existent
	removed = s.RemoveHook(HookPreToolUse, "NonExistent")
	assert.False(t, removed)
}

func TestLoadProjectSettings(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	settingsJSON := `{
		"env": {
			"MY_VAR": "my_value"
		},
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "Bash",
					"hooks": [{"type": "command", "command": "echo test"}]
				}
			]
		},
		"enabledPlugins": {
			"my-plugin": true
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(settingsJSON), 0644))

	settings, err := LoadProjectSettings(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "my_value", settings.Env["MY_VAR"])
	assert.Len(t, settings.Hooks["PreToolUse"], 1)
	assert.True(t, settings.EnabledPlugins["my-plugin"])
}

func TestLoadProjectSettings_NotExists(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadProjectSettings(tmpDir)
	require.NoError(t, err)

	// Should return empty settings, not error
	assert.NotNil(t, settings)
	assert.Empty(t, settings.Env)
}

func TestSaveProjectSettings(t *testing.T) {
	tmpDir := t.TempDir()

	settings := &Settings{
		Env: map[string]string{
			"SAVED_VAR": "saved_value",
		},
		EnabledPlugins: map[string]bool{
			"saved-plugin": true,
		},
	}

	err := SaveProjectSettings(tmpDir, settings)
	require.NoError(t, err)

	// Verify file exists
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	assert.FileExists(t, settingsPath)

	// Read back
	loaded, err := LoadProjectSettings(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "saved_value", loaded.Env["SAVED_VAR"])
	assert.True(t, loaded.EnabledPlugins["saved-plugin"])
}

func TestToolPermissions_IsEmpty(t *testing.T) {
	assert.True(t, (*ToolPermissions)(nil).IsEmpty())
	assert.True(t, (&ToolPermissions{}).IsEmpty())
	assert.False(t, (&ToolPermissions{Allow: []string{"Read"}}).IsEmpty())
	assert.False(t, (&ToolPermissions{Deny: []string{"Bash"}}).IsEmpty())
}

func TestToolPermissions_Merge(t *testing.T) {
	base := &ToolPermissions{Allow: []string{"Read", "Write"}}

	// Override takes precedence
	override := &ToolPermissions{Allow: []string{"Read"}}
	result := base.Merge(override)
	assert.Equal(t, []string{"Read"}, result.Allow)

	// Nil override returns base
	result = base.Merge(nil)
	assert.Equal(t, base, result)

	// Empty override returns base
	result = base.Merge(&ToolPermissions{})
	assert.Equal(t, base, result)

	// Nil base with override returns override
	result = (*ToolPermissions)(nil).Merge(override)
	assert.Equal(t, override, result)
}

func TestValidHookEvents(t *testing.T) {
	events := ValidHookEvents()
	assert.Contains(t, events, HookPreToolUse)
	assert.Contains(t, events, HookPostToolUse)
	assert.Contains(t, events, HookPreCompact)
	assert.Contains(t, events, HookPrePrompt)
	assert.Contains(t, events, HookStop)
}

func TestSettingsPath(t *testing.T) {
	path := SettingsPath("/home/user/project")
	assert.Equal(t, "/home/user/project/.claude/settings.json", path)
}
