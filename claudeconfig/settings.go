package claudeconfig

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// Settings represents Claude Code's settings.json structure.
type Settings struct {
	Env            map[string]string    `json:"env,omitempty"`
	Hooks          map[string][]Hook    `json:"hooks,omitempty"`
	EnabledPlugins map[string]bool      `json:"enabledPlugins,omitempty"`
	StatusLine     *StatusLine          `json:"statusLine,omitempty"`
	Extensions     map[string]any       `json:"extensions,omitempty"` // For custom extensions like "orc"
}

// Hook represents a hook entry in settings.json.
type Hook struct {
	Matcher string      `json:"matcher"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry represents a single hook action.
type HookEntry struct {
	Type    string `json:"type"`    // "command"
	Command string `json:"command"` // Script or command to execute
}

// StatusLine represents the status line configuration.
type StatusLine struct {
	Type    string `json:"type,omitempty"`
	Command string `json:"command,omitempty"`
}

// HookEvent represents valid hook event types.
type HookEvent string

const (
	HookPreToolUse  HookEvent = "PreToolUse"
	HookPostToolUse HookEvent = "PostToolUse"
	HookPreCompact  HookEvent = "PreCompact"
	HookPrePrompt   HookEvent = "PrePrompt"
	HookStop        HookEvent = "Stop"
)

// ValidHookEvents returns all valid hook event types.
func ValidHookEvents() []HookEvent {
	return []HookEvent{
		HookPreToolUse,
		HookPostToolUse,
		HookPreCompact,
		HookPrePrompt,
		HookStop,
	}
}

// ToolPermissions defines allow/deny lists for Claude Code tools.
type ToolPermissions struct {
	Allow []string `json:"allow,omitempty"` // Whitelist: only these tools allowed
	Deny  []string `json:"deny,omitempty"`  // Blacklist: these tools blocked
}

// IsEmpty returns true if no permissions are configured.
func (p *ToolPermissions) IsEmpty() bool {
	return p == nil || (len(p.Allow) == 0 && len(p.Deny) == 0)
}

// Merge combines two ToolPermissions, with override taking precedence.
func (p *ToolPermissions) Merge(override *ToolPermissions) *ToolPermissions {
	if override == nil || override.IsEmpty() {
		return p
	}
	if p == nil || p.IsEmpty() {
		return override
	}
	// Override takes full precedence
	return override
}

// NewSettings creates an empty Settings struct with initialized maps.
func NewSettings() *Settings {
	return &Settings{
		Env:            make(map[string]string),
		Hooks:          make(map[string][]Hook),
		EnabledPlugins: make(map[string]bool),
		Extensions:     make(map[string]any),
	}
}

// Clone creates a deep copy of the settings.
func (s *Settings) Clone() *Settings {
	if s == nil {
		return nil
	}

	clone := NewSettings()

	maps.Copy(clone.Env, s.Env)

	for k, v := range s.Hooks {
		hooks := make([]Hook, len(v))
		copy(hooks, v)
		clone.Hooks[k] = hooks
	}

	maps.Copy(clone.EnabledPlugins, s.EnabledPlugins)

	if s.StatusLine != nil {
		clone.StatusLine = &StatusLine{
			Type:    s.StatusLine.Type,
			Command: s.StatusLine.Command,
		}
	}

	maps.Copy(clone.Extensions, s.Extensions)

	return clone
}

// Merge combines global settings with project settings.
// Project settings override global settings.
func (s *Settings) Merge(project *Settings) *Settings {
	if s == nil {
		return project.Clone()
	}
	if project == nil {
		return s.Clone()
	}

	result := s.Clone()

	// Env: project overrides global
	maps.Copy(result.Env, project.Env)

	// Hooks: merge by event type (project hooks added to global)
	for event, hooks := range project.Hooks {
		if existing, ok := result.Hooks[event]; ok {
			result.Hooks[event] = append(existing, hooks...)
		} else {
			result.Hooks[event] = hooks
		}
	}

	// Plugins: union of both (project can enable/disable)
	maps.Copy(result.EnabledPlugins, project.EnabledPlugins)

	// StatusLine: project overrides global
	if project.StatusLine != nil {
		result.StatusLine = &StatusLine{
			Type:    project.StatusLine.Type,
			Command: project.StatusLine.Command,
		}
	}

	// Extensions: project overrides global
	maps.Copy(result.Extensions, project.Extensions)

	return result
}

// GetExtension retrieves a custom extension section and unmarshals it into the target.
func (s *Settings) GetExtension(name string, target any) error {
	if s == nil || s.Extensions == nil {
		return nil
	}

	ext, ok := s.Extensions[name]
	if !ok {
		return nil
	}

	// Re-marshal and unmarshal to convert map[string]any to target struct
	data, err := json.Marshal(ext)
	if err != nil {
		return fmt.Errorf("marshal extension: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal extension: %w", err)
	}

	return nil
}

// SetExtension sets a custom extension section.
func (s *Settings) SetExtension(name string, value any) {
	if s.Extensions == nil {
		s.Extensions = make(map[string]any)
	}
	s.Extensions[name] = value
}

// GetHooks returns hooks for the given event type.
func (s *Settings) GetHooks(event HookEvent) []Hook {
	if s == nil || s.Hooks == nil {
		return nil
	}
	return s.Hooks[string(event)]
}

// AddHook adds a hook for the given event type.
func (s *Settings) AddHook(event HookEvent, hook Hook) {
	if s.Hooks == nil {
		s.Hooks = make(map[string][]Hook)
	}
	s.Hooks[string(event)] = append(s.Hooks[string(event)], hook)
}

// RemoveHook removes a hook with the given matcher from the event.
func (s *Settings) RemoveHook(event HookEvent, matcher string) bool {
	if s.Hooks == nil {
		return false
	}

	hooks := s.Hooks[string(event)]
	for i, h := range hooks {
		if h.Matcher == matcher {
			s.Hooks[string(event)] = append(hooks[:i], hooks[i+1:]...)
			return true
		}
	}
	return false
}

// LoadSettings loads and merges global and project settings.
// The projectRoot is the root of the project (where .claude/ might exist).
func LoadSettings(projectRoot string) (*Settings, error) {
	global, err := LoadGlobalSettings()
	if err != nil {
		return nil, fmt.Errorf("load global settings: %w", err)
	}

	project, err := LoadProjectSettings(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load project settings: %w", err)
	}

	return global.Merge(project), nil
}

// LoadGlobalSettings loads settings from ~/.claude/settings.json.
func LoadGlobalSettings() (*Settings, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	path := filepath.Join(home, ".claude", "settings.json")
	return loadSettingsFile(path)
}

// LoadProjectSettings loads settings from {projectRoot}/.claude/settings.json.
func LoadProjectSettings(projectRoot string) (*Settings, error) {
	path := filepath.Join(projectRoot, ".claude", "settings.json")
	return loadSettingsFile(path)
}

// loadSettingsFile loads settings from a specific file path.
// Returns empty settings if the file doesn't exist.
func loadSettingsFile(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSettings(), nil
		}
		return nil, fmt.Errorf("read settings file: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse settings JSON: %w", err)
	}

	// Initialize nil maps
	if settings.Env == nil {
		settings.Env = make(map[string]string)
	}
	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]Hook)
	}
	if settings.EnabledPlugins == nil {
		settings.EnabledPlugins = make(map[string]bool)
	}
	if settings.Extensions == nil {
		settings.Extensions = make(map[string]any)
	}

	return &settings, nil
}

// SaveProjectSettings saves settings to {projectRoot}/.claude/settings.json.
func SaveProjectSettings(projectRoot string, settings *Settings) error {
	claudeDir := filepath.Join(projectRoot, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("create .claude directory: %w", err)
	}

	path := filepath.Join(claudeDir, "settings.json")
	return saveSettingsFile(path, settings)
}

// saveSettingsFile saves settings to a specific file path.
func saveSettingsFile(path string, settings *Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}

	return nil
}

// SettingsPath returns the path to the project settings file.
func SettingsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "settings.json")
}

// GlobalSettingsPath returns the path to the global settings file.
func GlobalSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}
