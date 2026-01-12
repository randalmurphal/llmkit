// Package claudeconfig provides utilities for parsing Claude Code's native configuration formats.
package claudeconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Plugin represents a Claude Code plugin parsed from plugin.json.
// Plugins are directories containing a .claude-plugin/plugin.json manifest.
type Plugin struct {
	// Fields from plugin.json
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Author      PluginAuthor `json:"author,omitempty"`
	Homepage    string       `json:"homepage,omitempty"`
	Keywords    []string     `json:"keywords,omitempty"`

	// Metadata (not from plugin.json)
	Path        string      `json:"-"` // Directory containing .claude-plugin/
	Scope       PluginScope `json:"-"` // global or project
	Enabled     bool        `json:"-"` // From settings.json enabledPlugins
	Version     string      `json:"-"` // From version field or installed_plugins.json
	InstalledAt time.Time   `json:"-"` // Installation timestamp
	UpdatedAt   time.Time   `json:"-"` // Last update timestamp

	// Resource flags indicate presence of subdirectories
	HasCommands bool `json:"-"`
	HasHooks    bool `json:"-"`
	HasScripts  bool `json:"-"`

	// Discovered commands
	Commands []PluginCommand `json:"-"`
}

// PluginAuthor represents the author information in plugin.json.
type PluginAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// PluginScope represents where a plugin is installed.
type PluginScope string

const (
	// PluginScopeGlobal indicates a plugin installed in ~/.claude/plugins/
	PluginScopeGlobal PluginScope = "global"
	// PluginScopeProject indicates a plugin installed in .claude/plugins/
	PluginScopeProject PluginScope = "project"
)

// PluginInfo provides summary information for listing plugins.
type PluginInfo struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Author       string      `json:"author"`
	Path         string      `json:"path"`
	Scope        PluginScope `json:"scope"`
	Enabled      bool        `json:"enabled"`
	Version      string      `json:"version,omitempty"`
	HasCommands  bool        `json:"has_commands"`
	CommandCount int         `json:"command_count"`
}

// PluginCommand represents a slash command provided by a plugin.
type PluginCommand struct {
	Name         string `json:"name"`          // Command name (filename without .md)
	Description  string `json:"description"`   // From YAML frontmatter
	ArgumentHint string `json:"argument_hint"` // From YAML frontmatter
	FilePath     string `json:"file_path"`     // Full path to command file
}

// Info returns summary information for this plugin.
func (p *Plugin) Info() PluginInfo {
	authorName := p.Author.Name
	return PluginInfo{
		Name:         p.Name,
		Description:  p.Description,
		Author:       authorName,
		Path:         p.Path,
		Scope:        p.Scope,
		Enabled:      p.Enabled,
		Version:      p.Version,
		HasCommands:  p.HasCommands,
		CommandCount: len(p.Commands),
	}
}

// Validate checks that the plugin has required fields.
func (p *Plugin) Validate() error {
	if p.Name == "" {
		return errors.New("plugin name is required")
	}
	if p.Description == "" {
		return errors.New("plugin description is required")
	}
	return nil
}

// ParsePluginJSON reads and parses a plugin.json file.
// The path can be:
// - The plugin.json file itself
// - The .claude-plugin directory containing plugin.json
// - The plugin directory containing .claude-plugin/plugin.json
func ParsePluginJSON(path string) (*Plugin, error) {
	// Determine the actual file path
	filePath := path
	dirPath := path

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		// Check if this is the .claude-plugin directory
		if filepath.Base(path) == ".claude-plugin" {
			filePath = filepath.Join(path, "plugin.json")
			dirPath = filepath.Dir(path)
		} else {
			// Assume this is the plugin root directory
			filePath = filepath.Join(path, ".claude-plugin", "plugin.json")
			dirPath = path
		}
	} else {
		// It's a file, get the plugin root (parent of .claude-plugin)
		dirPath = filepath.Dir(filepath.Dir(path))
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read plugin.json: %w", err)
	}

	// Parse JSON
	plugin := &Plugin{}
	if err := json.Unmarshal(data, plugin); err != nil {
		return nil, fmt.Errorf("parse plugin.json: %w", err)
	}

	plugin.Path = dirPath

	// Check for resource subdirectories
	plugin.HasCommands = dirExists(filepath.Join(dirPath, "commands"))
	plugin.HasHooks = dirExists(filepath.Join(dirPath, "hooks"))
	plugin.HasScripts = dirExists(filepath.Join(dirPath, "scripts"))

	// Discover commands if present
	if plugin.HasCommands {
		commands, err := discoverPluginCommands(filepath.Join(dirPath, "commands"))
		if err == nil {
			plugin.Commands = commands
		}
	}

	return plugin, nil
}

// discoverPluginCommands finds all command files in a plugin's commands directory.
func discoverPluginCommands(commandsDir string) ([]PluginCommand, error) {
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		return nil, fmt.Errorf("read commands directory: %w", err)
	}

	var commands []PluginCommand
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		cmdName := strings.TrimSuffix(name, ".md")
		cmdPath := filepath.Join(commandsDir, name)

		// Parse frontmatter for description and argument-hint
		desc, argHint := parseCommandFrontmatter(cmdPath)

		commands = append(commands, PluginCommand{
			Name:         cmdName,
			Description:  desc,
			ArgumentHint: argHint,
			FilePath:     cmdPath,
		})
	}

	// Sort by name
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	return commands, nil
}

// parseCommandFrontmatter extracts description and argument-hint from command file.
func parseCommandFrontmatter(path string) (description, argumentHint string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return "", ""
	}

	// Find end of frontmatter
	endIdx := strings.Index(content[3:], "---")
	if endIdx == -1 {
		return "", ""
	}

	frontmatter := content[3 : endIdx+3]
	lines := strings.Split(frontmatter, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			description = strings.Trim(description, `"'`)
		} else if strings.HasPrefix(line, "argument-hint:") {
			argumentHint = strings.TrimSpace(strings.TrimPrefix(line, "argument-hint:"))
			argumentHint = strings.Trim(argumentHint, `"'`)
		}
	}

	return description, argumentHint
}

// DiscoverPlugins finds all plugins in the given .claude directory.
// It searches in the plugins/ subdirectory.
func DiscoverPlugins(claudeDir string) ([]*Plugin, error) {
	pluginsDir := filepath.Join(claudeDir, "plugins")

	// Check if plugins directory exists
	if !dirExists(pluginsDir) {
		return nil, nil // No plugins directory, return empty list
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("read plugins directory: %w", err)
	}

	var plugins []*Plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(pluginsDir, entry.Name())
		pluginJSONPath := filepath.Join(pluginPath, ".claude-plugin", "plugin.json")

		// Check if plugin.json exists in this directory
		if !fileExists(pluginJSONPath) {
			continue
		}

		plugin, err := ParsePluginJSON(pluginPath)
		if err != nil {
			// Log warning but continue discovering other plugins
			continue
		}

		plugins = append(plugins, plugin)
	}

	// Sort by name for consistent ordering
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins, nil
}

// DiscoverPluginsWithEnabled finds all plugins and sets their Enabled status
// based on the provided settings.
func DiscoverPluginsWithEnabled(claudeDir string, settings *Settings) ([]*Plugin, error) {
	plugins, err := DiscoverPlugins(claudeDir)
	if err != nil {
		return nil, err
	}

	if settings != nil && settings.EnabledPlugins != nil {
		for _, plugin := range plugins {
			// Check both simple name and namespaced name
			if enabled, ok := settings.EnabledPlugins[plugin.Name]; ok {
				plugin.Enabled = enabled
			}
		}
	}

	return plugins, nil
}

// PluginKey returns the key used in settings.json enabledPlugins map.
// Format: "name" for local plugins, "name@namespace" for marketplace plugins.
func PluginKey(name, namespace string) string {
	if namespace == "" {
		return name
	}
	return name + "@" + namespace
}

// ParsePluginKey extracts name and namespace from a plugin key.
func ParsePluginKey(key string) (name, namespace string) {
	if idx := strings.LastIndex(key, "@"); idx != -1 {
		return key[:idx], key[idx+1:]
	}
	return key, ""
}

// GlobalPluginsDir returns the path to the global plugins directory.
func GlobalPluginsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "plugins"), nil
}

// ProjectPluginsDir returns the path to the project plugins directory.
func ProjectPluginsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "plugins")
}
