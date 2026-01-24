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

	"github.com/randalmurphal/llmkit/claudecontract"
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

	// Discovered resources
	Commands   []PluginCommand   `json:"-"`
	MCPServers []PluginMCPServer `json:"-"`
	Hooks      []PluginHook      `json:"-"`
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

// PluginMCPServer represents an MCP server provided by a plugin.
// Parsed from .mcp.json in the plugin directory.
type PluginMCPServer struct {
	Name    string            `json:"name"`              // Server name (key in .mcp.json)
	Command string            `json:"command"`           // Command to run
	Args    []string          `json:"args,omitempty"`    // Command arguments
	Env     map[string]string `json:"env,omitempty"`     // Environment variables
	URL     string            `json:"url,omitempty"`     // URL for HTTP-based servers
	Type    string            `json:"type,omitempty"`    // Server type (stdio, sse, http)
}

// PluginHook represents a hook provided by a plugin.
// Parsed from hooks/hooks.json in the plugin directory.
type PluginHook struct {
	Event       string   `json:"event"`                 // Hook event (Stop, PreToolUse, etc.)
	Type        string   `json:"type"`                  // Hook type (command)
	Command     string   `json:"command"`               // Command to execute
	Matcher     string   `json:"matcher,omitempty"`     // Tool matcher pattern
	Description string   `json:"description,omitempty"` // Hook description
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
		if filepath.Base(path) == claudecontract.DirClaudePlugin {
			filePath = filepath.Join(path, claudecontract.FilePluginJSON)
			dirPath = filepath.Dir(path)
		} else {
			// Assume this is the plugin root directory
			filePath = filepath.Join(path, claudecontract.DirClaudePlugin, claudecontract.FilePluginJSON)
			dirPath = path
		}
	} else {
		// It's a file, get the plugin root (parent of .claude-plugin)
		dirPath = filepath.Dir(filepath.Dir(path))
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", claudecontract.FilePluginJSON, err)
	}

	// Parse JSON
	plugin := &Plugin{}
	if err := json.Unmarshal(data, plugin); err != nil {
		return nil, fmt.Errorf("parse %s: %w", claudecontract.FilePluginJSON, err)
	}

	plugin.Path = dirPath

	// Check for resource subdirectories
	plugin.HasCommands = dirExists(filepath.Join(dirPath, claudecontract.DirCommands))
	plugin.HasHooks = dirExists(filepath.Join(dirPath, claudecontract.DirHooks))
	plugin.HasScripts = dirExists(filepath.Join(dirPath, claudecontract.DirScripts))

	// Discover commands if present
	if plugin.HasCommands {
		commands, err := discoverPluginCommands(filepath.Join(dirPath, claudecontract.DirCommands))
		if err == nil {
			plugin.Commands = commands
		}
	}

	// Discover MCP servers from .mcp.json
	mcpServers, err := discoverPluginMCPServers(dirPath)
	if err == nil {
		plugin.MCPServers = mcpServers
	}

	// Discover hooks from hooks/hooks.json
	if plugin.HasHooks {
		hooks, err := discoverPluginHooks(filepath.Join(dirPath, claudecontract.DirHooks))
		if err == nil {
			plugin.Hooks = hooks
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

// discoverPluginMCPServers parses .mcp.json from the plugin directory.
func discoverPluginMCPServers(pluginDir string) ([]PluginMCPServer, error) {
	mcpPath := filepath.Join(pluginDir, claudecontract.FileMCPConfig)
	if !fileExists(mcpPath) {
		return nil, nil
	}

	data, err := os.ReadFile(mcpPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", claudecontract.FileMCPConfig, err)
	}

	// .mcp.json format: {"server-name": {"command": "...", "args": [...]}}
	var mcpConfig map[string]struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Type    string            `json:"type"`
	}

	if err := json.Unmarshal(data, &mcpConfig); err != nil {
		return nil, fmt.Errorf("parse %s: %w", claudecontract.FileMCPConfig, err)
	}

	var servers []PluginMCPServer
	for name, config := range mcpConfig {
		server := PluginMCPServer{
			Name:    name,
			Command: config.Command,
			Args:    config.Args,
			Env:     config.Env,
			URL:     config.URL,
			Type:    config.Type,
		}
		// Infer type if not specified
		if server.Type == "" {
			if server.URL != "" {
				server.Type = "sse"
			} else {
				server.Type = "stdio"
			}
		}
		servers = append(servers, server)
	}

	// Sort by name for consistent ordering
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers, nil
}

// discoverPluginHooks parses hooks/hooks.json from the plugin directory.
func discoverPluginHooks(hooksDir string) ([]PluginHook, error) {
	hooksPath := filepath.Join(hooksDir, "hooks.json")
	if !fileExists(hooksPath) {
		return nil, nil
	}

	data, err := os.ReadFile(hooksPath)
	if err != nil {
		return nil, fmt.Errorf("read hooks.json: %w", err)
	}

	// hooks.json format:
	// {
	//   "description": "...",
	//   "hooks": {
	//     "Stop": [{"hooks": [{"type": "command", "command": "..."}]}],
	//     "PreToolUse": [{"hooks": [...], "matcher": "Edit|Write"}]
	//   }
	// }
	var hooksConfig struct {
		Description string `json:"description"`
		Hooks       map[string][]struct {
			Hooks []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
			Matcher string `json:"matcher"`
		} `json:"hooks"`
	}

	if err := json.Unmarshal(data, &hooksConfig); err != nil {
		return nil, fmt.Errorf("parse hooks.json: %w", err)
	}

	var hooks []PluginHook
	for event, hookConfigs := range hooksConfig.Hooks {
		for _, config := range hookConfigs {
			for _, hook := range config.Hooks {
				hooks = append(hooks, PluginHook{
					Event:       event,
					Type:        hook.Type,
					Command:     hook.Command,
					Matcher:     config.Matcher,
					Description: hooksConfig.Description,
				})
			}
		}
	}

	// Sort by event then command for consistent ordering
	sort.Slice(hooks, func(i, j int) bool {
		if hooks[i].Event != hooks[j].Event {
			return hooks[i].Event < hooks[j].Event
		}
		return hooks[i].Command < hooks[j].Command
	})

	return hooks, nil
}

// DiscoverPlugins finds all plugins in the given .claude directory.
// It searches in multiple locations:
// 1. plugins/{name}/.claude-plugin/plugin.json (simple format)
// 2. plugins/cache/{marketplace}/{name}/{version}/.claude-plugin/plugin.json (Claude Code format)
func DiscoverPlugins(claudeDir string) ([]*Plugin, error) {
	pluginsDir := filepath.Join(claudeDir, claudecontract.DirPlugins)

	// Check if plugins directory exists
	if !dirExists(pluginsDir) {
		return nil, nil // No plugins directory, return empty list
	}

	var plugins []*Plugin
	seen := make(map[string]bool) // Track seen plugins by name to avoid duplicates

	// 1. Scan direct plugins (plugins/{name}/.claude-plugin/plugin.json)
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == claudecontract.DirCache {
			continue
		}

		pluginPath := filepath.Join(pluginsDir, entry.Name())
		pluginJSONPath := filepath.Join(pluginPath, claudecontract.DirClaudePlugin, claudecontract.FilePluginJSON)

		if !fileExists(pluginJSONPath) {
			continue
		}

		plugin, err := ParsePluginJSON(pluginPath)
		if err != nil {
			continue
		}

		if !seen[plugin.Name] {
			plugins = append(plugins, plugin)
			seen[plugin.Name] = true
		}
	}

	// 2. Scan cache directory (plugins/cache/{marketplace}/{name}/{version}/.claude-plugin/plugin.json)
	cacheDir := filepath.Join(pluginsDir, claudecontract.DirCache)
	if dirExists(cacheDir) {
		marketplaces, err := os.ReadDir(cacheDir)
		if err == nil {
			for _, marketplace := range marketplaces {
				if !marketplace.IsDir() {
					continue
				}

				marketplacePath := filepath.Join(cacheDir, marketplace.Name())
				pluginDirs, err := os.ReadDir(marketplacePath)
				if err != nil {
					continue
				}

				for _, pluginDir := range pluginDirs {
					if !pluginDir.IsDir() {
						continue
					}

					// Scan version directories
					pluginPath := filepath.Join(marketplacePath, pluginDir.Name())
					versions, err := os.ReadDir(pluginPath)
					if err != nil {
						continue
					}

					// Find the latest version (or any version with plugin.json)
					for _, version := range versions {
						if !version.IsDir() {
							continue
						}

						versionPath := filepath.Join(pluginPath, version.Name())
						pluginJSONPath := filepath.Join(versionPath, claudecontract.DirClaudePlugin, claudecontract.FilePluginJSON)

						if !fileExists(pluginJSONPath) {
							continue
						}

						plugin, err := ParsePluginJSON(versionPath)
						if err != nil {
							continue
						}

						// Set version from directory name if not in plugin.json
						if plugin.Version == "" {
							plugin.Version = version.Name()
						}

						// Only add if not already seen (first version found wins)
						if !seen[plugin.Name] {
							plugins = append(plugins, plugin)
							seen[plugin.Name] = true
						}
						break // Only take first valid version found
					}
				}
			}
		}
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
	return filepath.Join(home, claudecontract.DirClaude, claudecontract.DirPlugins), nil
}

// ProjectPluginsDir returns the path to the project plugins directory.
func ProjectPluginsDir(projectRoot string) string {
	return filepath.Join(projectRoot, claudecontract.DirClaude, claudecontract.DirPlugins)
}
