package claudeconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

// PluginService manages plugin configurations across global and project scopes.
type PluginService struct {
	projectRoot string
	globalDir   string
}

// NewPluginService creates a new PluginService.
// projectRoot is the root of the project (where .claude/ might exist).
// If projectRoot is empty, only global plugins are managed.
func NewPluginService(projectRoot string) (*PluginService, error) {
	globalDir, err := GlobalPluginsDir()
	if err != nil {
		return nil, fmt.Errorf("get global plugins dir: %w", err)
	}

	return &PluginService{
		projectRoot: projectRoot,
		globalDir:   globalDir,
	}, nil
}

// List returns all plugins from both global and project scopes.
// Project plugins with the same name override global plugins.
func (s *PluginService) List() ([]PluginInfo, error) {
	// Load settings for enabled status
	var settings *Settings
	if s.projectRoot != "" {
		settings, _ = LoadSettings(s.projectRoot)
	} else {
		settings, _ = LoadGlobalSettings()
	}

	// Discover global plugins
	globalPlugins, err := s.discoverWithScope(s.globalDir, PluginScopeGlobal, settings)
	if err != nil {
		return nil, fmt.Errorf("discover global plugins: %w", err)
	}

	// If no project root, return only global
	if s.projectRoot == "" {
		return pluginsToInfos(globalPlugins), nil
	}

	// Discover project plugins
	projectDir := ProjectPluginsDir(s.projectRoot)
	projectPlugins, err := s.discoverWithScope(projectDir, PluginScopeProject, settings)
	if err != nil {
		return nil, fmt.Errorf("discover project plugins: %w", err)
	}

	// Merge: project overrides global with same name
	merged := make(map[string]*Plugin)
	for _, p := range globalPlugins {
		merged[p.Name] = p
	}
	for _, p := range projectPlugins {
		merged[p.Name] = p
	}

	// Convert to slice
	var result []*Plugin
	for _, p := range merged {
		result = append(result, p)
	}

	return pluginsToInfos(result), nil
}

// ListByScope returns plugins from a specific scope only.
func (s *PluginService) ListByScope(scope PluginScope) ([]PluginInfo, error) {
	var settings *Settings
	if s.projectRoot != "" {
		settings, _ = LoadSettings(s.projectRoot)
	} else {
		settings, _ = LoadGlobalSettings()
	}

	var dir string
	switch scope {
	case PluginScopeGlobal:
		dir = s.globalDir
	case PluginScopeProject:
		if s.projectRoot == "" {
			return nil, fmt.Errorf("no project root configured")
		}
		dir = ProjectPluginsDir(s.projectRoot)
	default:
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	plugins, err := s.discoverWithScope(dir, scope, settings)
	if err != nil {
		return nil, err
	}

	return pluginsToInfos(plugins), nil
}

// discoverWithScope discovers plugins in a directory and sets their scope.
func (s *PluginService) discoverWithScope(dir string, scope PluginScope, settings *Settings) ([]*Plugin, error) {
	claudeDir := filepath.Dir(dir) // plugins is inside .claude
	plugins, err := DiscoverPluginsWithEnabled(claudeDir, settings)
	if err != nil {
		return nil, err
	}

	for _, p := range plugins {
		p.Scope = scope
	}

	return plugins, nil
}

// Get returns a specific plugin by name and scope.
// Uses DiscoverPlugins to find plugins in both simple and cache directory formats.
func (s *PluginService) Get(name string, scope PluginScope) (*Plugin, error) {
	var claudeDir string
	switch scope {
	case PluginScopeGlobal:
		claudeDir = filepath.Dir(s.globalDir) // .claude directory
	case PluginScopeProject:
		if s.projectRoot == "" {
			return nil, fmt.Errorf("no project root configured")
		}
		claudeDir = filepath.Join(s.projectRoot, ".claude")
	default:
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	// Load settings for enabled status
	var settings *Settings
	if scope == PluginScopeProject && s.projectRoot != "" {
		settings, _ = LoadProjectSettings(s.projectRoot)
	} else {
		settings, _ = LoadGlobalSettings()
	}

	// Discover all plugins and find by name
	plugins, err := DiscoverPluginsWithEnabled(claudeDir, settings)
	if err != nil {
		return nil, fmt.Errorf("discover plugins: %w", err)
	}

	for _, p := range plugins {
		if p.Name == name {
			p.Scope = scope
			return p, nil
		}
	}

	return nil, fmt.Errorf("plugin not found: %s", name)
}

// SetEnabled enables or disables a plugin by updating settings.json.
func (s *PluginService) SetEnabled(name string, scope PluginScope, enabled bool) error {
	var settingsPath string
	var settings *Settings
	var err error

	switch scope {
	case PluginScopeGlobal:
		settingsPath, err = GlobalSettingsPath()
		if err != nil {
			return fmt.Errorf("get global settings path: %w", err)
		}
		settings, err = LoadGlobalSettings()
	case PluginScopeProject:
		if s.projectRoot == "" {
			return fmt.Errorf("no project root configured")
		}
		settingsPath = SettingsPath(s.projectRoot)
		settings, err = LoadProjectSettings(s.projectRoot)
	default:
		return fmt.Errorf("unknown scope: %s", scope)
	}

	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	if settings == nil {
		settings = NewSettings()
	}

	if settings.EnabledPlugins == nil {
		settings.EnabledPlugins = make(map[string]bool)
	}

	settings.EnabledPlugins[name] = enabled

	// Save settings
	if scope == PluginScopeProject {
		return SaveProjectSettings(s.projectRoot, settings)
	}

	// For global, we need to save directly
	return saveSettingsFile(settingsPath, settings)
}

// Enable enables a plugin.
func (s *PluginService) Enable(name string, scope PluginScope) error {
	return s.SetEnabled(name, scope, true)
}

// Disable disables a plugin.
func (s *PluginService) Disable(name string, scope PluginScope) error {
	return s.SetEnabled(name, scope, false)
}

// Uninstall removes a plugin from the specified scope.
// Uses discovery to find the plugin's actual path (supports cache directory format).
func (s *PluginService) Uninstall(name string, scope PluginScope) error {
	// Use Get to find the plugin (handles cache directory format)
	plugin, err := s.Get(name, scope)
	if err != nil {
		return err
	}

	if plugin.Path == "" {
		return fmt.Errorf("plugin has no path: %s", name)
	}

	// Remove the plugin directory
	if err := os.RemoveAll(plugin.Path); err != nil {
		return fmt.Errorf("remove plugin directory: %w", err)
	}

	// Remove from enabledPlugins in settings
	_ = s.removeFromSettings(name, scope)

	return nil
}

// removeFromSettings removes a plugin from the enabledPlugins map.
func (s *PluginService) removeFromSettings(name string, scope PluginScope) error {
	var settings *Settings
	var err error

	switch scope {
	case PluginScopeGlobal:
		settings, err = LoadGlobalSettings()
	case PluginScopeProject:
		settings, err = LoadProjectSettings(s.projectRoot)
	}

	if err != nil || settings == nil {
		return nil // No settings to update
	}

	if settings.EnabledPlugins != nil {
		delete(settings.EnabledPlugins, name)
	}

	if scope == PluginScopeProject {
		return SaveProjectSettings(s.projectRoot, settings)
	}

	settingsPath, _ := GlobalSettingsPath()
	return saveSettingsFile(settingsPath, settings)
}

// ListCommands returns all commands for a plugin.
func (s *PluginService) ListCommands(name string, scope PluginScope) ([]PluginCommand, error) {
	plugin, err := s.Get(name, scope)
	if err != nil {
		return nil, err
	}

	return plugin.Commands, nil
}

// pluginsToInfos converts a slice of plugins to plugin infos.
func pluginsToInfos(plugins []*Plugin) []PluginInfo {
	infos := make([]PluginInfo, 0, len(plugins))
	for _, p := range plugins {
		infos = append(infos, p.Info())
	}
	return infos
}

// SaveGlobalSettings saves settings to ~/.claude/settings.json.
func SaveGlobalSettings(settings *Settings) error {
	path, err := GlobalSettingsPath()
	if err != nil {
		return fmt.Errorf("get global settings path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create .claude directory: %w", err)
	}

	return saveSettingsFile(path, settings)
}
