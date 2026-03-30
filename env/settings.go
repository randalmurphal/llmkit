package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/randalmurphal/llmkit/v2"
	"github.com/randalmurphal/llmkit/v2/claudeconfig"
	"github.com/randalmurphal/llmkit/v2/codexconfig"
)

// Hook is the shared llmkit view of a provider-local hook entry.
type Hook struct {
	Matcher       string            `json:"matcher,omitempty"`
	Type          string            `json:"type,omitempty"`
	Command       string            `json:"command,omitempty"`
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Prompt        string            `json:"prompt,omitempty"`
	Model         string            `json:"model,omitempty"`
	Async         bool              `json:"async,omitempty"`
	Timeout       int               `json:"timeout,omitempty"`
	StatusMessage string            `json:"status_message,omitempty"`
	Once          bool              `json:"once,omitempty"`
}

// Settings is the shared llmkit view of project-local environment settings.
type Settings struct {
	Provider   string                            `json:"provider"`
	Hooks      map[string][]Hook                 `json:"hooks,omitempty"`
	MCPServers map[string]llmkit.MCPServerConfig `json:"mcp_servers,omitempty"`
	Env        map[string]string                 `json:"env,omitempty"`
}

func NewSettings(provider string) *Settings {
	return &Settings{
		Provider:   provider,
		Hooks:      map[string][]Hook{},
		MCPServers: map[string]llmkit.MCPServerConfig{},
		Env:        map[string]string{},
	}
}

func (s *Settings) Clone() *Settings {
	if s == nil {
		return nil
	}
	clone := NewSettings(s.Provider)
	maps.Copy(clone.Env, s.Env)
	for event, hooks := range s.Hooks {
		clone.Hooks[event] = append([]Hook(nil), hooks...)
	}
	for name, server := range s.MCPServers {
		clone.MCPServers[name] = cloneMCPServer(server)
	}
	return clone
}

func (s *Settings) GetHooks(event string) []Hook {
	if s == nil {
		return nil
	}
	return append([]Hook(nil), s.Hooks[event]...)
}

func (s *Settings) AddHook(event string, hook Hook) {
	if s.Hooks == nil {
		s.Hooks = map[string][]Hook{}
	}
	s.Hooks[event] = append(s.Hooks[event], hook)
}

func (s *Settings) RemoveHook(event, matcher string) bool {
	if s == nil {
		return false
	}
	hooks := s.Hooks[event]
	for i, hook := range hooks {
		if hook.Matcher == matcher {
			s.Hooks[event] = append(hooks[:i], hooks[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Settings) GetMCPServers() map[string]llmkit.MCPServerConfig {
	if s == nil {
		return nil
	}
	out := make(map[string]llmkit.MCPServerConfig, len(s.MCPServers))
	for name, server := range s.MCPServers {
		out[name] = cloneMCPServer(server)
	}
	return out
}

func (s *Settings) AddMCPServer(name string, cfg llmkit.MCPServerConfig) {
	if s.MCPServers == nil {
		s.MCPServers = map[string]llmkit.MCPServerConfig{}
	}
	s.MCPServers[name] = cloneMCPServer(cfg)
}

func (s *Settings) RemoveMCPServer(name string) bool {
	if s == nil {
		return false
	}
	if _, ok := s.MCPServers[name]; !ok {
		return false
	}
	delete(s.MCPServers, name)
	return true
}

// LoadSettings loads the project-local settings for the selected provider.
func LoadSettings(provider, workDir string) (*Settings, error) {
	store, err := openProjectStore(provider, workDir)
	if err != nil {
		return nil, err
	}
	return store.snapshot(), nil
}

// SaveSettings replaces the provider-local project settings represented by the shared view.
func SaveSettings(provider, workDir string, settings *Settings) error {
	store, err := openProjectStore(provider, workDir)
	if err != nil {
		return err
	}
	if err := store.replace(settings); err != nil {
		return err
	}
	return store.save()
}

type projectStore interface {
	snapshot() *Settings
	replace(*Settings) error
	addHook(event string, hook Hook) error
	removeHookIfMatches(event string, hook Hook) bool
	setEnv(key, value string) error
	removeEnvIfMatches(key, value string) bool
	setMCP(name string, cfg llmkit.MCPServerConfig) error
	removeMCPIfMatches(name string, cfg llmkit.MCPServerConfig) bool
	save() error
	paths() []string
}

func openProjectStore(provider, workDir string) (projectStore, error) {
	switch provider {
	case "claude":
		settings, err := claudeconfig.LoadProjectSettings(workDir)
		if err != nil {
			return nil, err
		}
		mcp, err := claudeconfig.LoadProjectMCPConfig(workDir)
		if err != nil {
			return nil, err
		}
		return &claudeStore{workDir: workDir, settings: settings, mcp: mcp}, nil
	case "codex":
		cfg, err := codexconfig.LoadProjectConfig(workDir)
		if err != nil {
			return nil, err
		}
		hooks, err := codexconfig.LoadHooks(workDir)
		if err != nil {
			return nil, err
		}
		return &codexStore{workDir: workDir, config: cfg, hooks: hooks}, nil
	default:
		return nil, fmt.Errorf("%w: %s", llmkit.ErrUnknownProvider, provider)
	}
}

type claudeStore struct {
	workDir  string
	settings *claudeconfig.Settings
	mcp      *claudeconfig.MCPConfig
}

func (s *claudeStore) snapshot() *Settings {
	out := NewSettings("claude")
	maps.Copy(out.Env, s.settings.Env)
	out.Hooks = flattenClaudeHooks(s.settings.Hooks)
	for name, server := range s.mcp.MCPServers {
		out.MCPServers[name] = llmkit.MCPServerConfig{
			Type:     server.Type,
			Command:  server.Command,
			Args:     append([]string(nil), server.Args...),
			Env:      cloneStringMap(server.Env),
			URL:      server.URL,
			Headers:  sliceHeadersToMap(server.Headers),
			Disabled: server.Disabled,
		}
	}
	return out
}

func (s *claudeStore) replace(settings *Settings) error {
	if settings == nil {
		settings = NewSettings("claude")
	}
	s.settings.Env = cloneStringMap(settings.Env)
	s.settings.Hooks = expandClaudeHooks(settings.Hooks)
	s.mcp.MCPServers = map[string]*claudeconfig.MCPServer{}
	for name, server := range settings.MCPServers {
		s.mcp.MCPServers[name] = &claudeconfig.MCPServer{
			Type:     server.Type,
			Command:  server.Command,
			Args:     append([]string(nil), server.Args...),
			Env:      cloneStringMap(server.Env),
			URL:      server.URL,
			Headers:  mapHeadersToSlice(server.Headers),
			Disabled: server.Disabled,
		}
	}
	return nil
}

func (s *claudeStore) addHook(event string, hook Hook) error {
	if s.settings.Hooks == nil {
		s.settings.Hooks = map[string][]claudeconfig.Hook{}
	}
	entry := claudeHookEntry(hook)
	hooks := s.settings.Hooks[event]
	for i := range hooks {
		if hooks[i].Matcher == hook.Matcher {
			hooks[i].Hooks = append(hooks[i].Hooks, entry)
			s.settings.Hooks[event] = hooks
			return nil
		}
	}
	s.settings.Hooks[event] = append(s.settings.Hooks[event], claudeconfig.Hook{
		Matcher: hook.Matcher,
		Hooks:   []claudeconfig.HookEntry{entry},
	})
	return nil
}

func (s *claudeStore) removeHookIfMatches(event string, hook Hook) bool {
	groups := s.settings.Hooks[event]
	for i := range groups {
		for j := range groups[i].Hooks {
			if groups[i].Matcher == hook.Matcher && reflect.DeepEqual(groups[i].Hooks[j], claudeHookEntry(hook)) {
				groups[i].Hooks = append(groups[i].Hooks[:j], groups[i].Hooks[j+1:]...)
				if len(groups[i].Hooks) == 0 {
					s.settings.Hooks[event] = append(groups[:i], groups[i+1:]...)
				} else {
					s.settings.Hooks[event] = groups
				}
				return true
			}
		}
	}
	return false
}

func (s *claudeStore) setEnv(key, value string) error {
	if s.settings.Env == nil {
		s.settings.Env = map[string]string{}
	}
	s.settings.Env[key] = value
	return nil
}

func (s *claudeStore) removeEnvIfMatches(key, value string) bool {
	if current, ok := s.settings.Env[key]; ok && current == value {
		delete(s.settings.Env, key)
		return true
	}
	return false
}

func (s *claudeStore) setMCP(name string, cfg llmkit.MCPServerConfig) error {
	if s.mcp.MCPServers == nil {
		s.mcp.MCPServers = map[string]*claudeconfig.MCPServer{}
	}
	s.mcp.MCPServers[name] = &claudeconfig.MCPServer{
		Type:     cfg.Type,
		Command:  cfg.Command,
		Args:     append([]string(nil), cfg.Args...),
		Env:      cloneStringMap(cfg.Env),
		URL:      cfg.URL,
		Headers:  mapHeadersToSlice(cfg.Headers),
		Disabled: cfg.Disabled,
	}
	return nil
}

func (s *claudeStore) removeMCPIfMatches(name string, cfg llmkit.MCPServerConfig) bool {
	current, ok := s.mcp.MCPServers[name]
	if !ok {
		return false
	}
	want := &claudeconfig.MCPServer{
		Type:     cfg.Type,
		Command:  cfg.Command,
		Args:     append([]string(nil), cfg.Args...),
		Env:      cloneStringMap(cfg.Env),
		URL:      cfg.URL,
		Headers:  mapHeadersToSlice(cfg.Headers),
		Disabled: cfg.Disabled,
	}
	if !reflect.DeepEqual(current, want) {
		return false
	}
	delete(s.mcp.MCPServers, name)
	return true
}

func (s *claudeStore) save() error {
	if err := writeJSONAtomic(claudeconfig.SettingsPath(s.workDir), s.settings); err != nil {
		return err
	}
	return writeJSONAtomic(claudeconfig.MCPConfigPath(s.workDir), s.mcp)
}

func (s *claudeStore) paths() []string {
	return []string{
		claudeconfig.SettingsPath(s.workDir),
		claudeconfig.MCPConfigPath(s.workDir),
	}
}

type codexStore struct {
	workDir string
	config  *codexconfig.ConfigFile
	hooks   *codexconfig.HookConfig
}

func (s *codexStore) snapshot() *Settings {
	out := NewSettings("codex")
	out.Hooks = flattenCodexHooks(s.hooks.Hooks)
	for name, server := range s.config.MCPServers {
		out.MCPServers[name] = llmkit.MCPServerConfig{
			Type:     server.Type,
			Command:  server.Command,
			Args:     append([]string(nil), server.Args...),
			Env:      cloneStringMap(server.Env),
			URL:      server.URL,
			Headers:  cloneStringMap(server.Headers),
			Disabled: server.Disabled,
		}
	}
	return out
}

func (s *codexStore) replace(settings *Settings) error {
	if settings == nil {
		settings = NewSettings("codex")
	}
	if len(settings.Env) > 0 {
		return fmt.Errorf("%w: codex project env overrides", llmkit.ErrCapabilityNotSupported)
	}
	s.config.MCPServers = map[string]codexconfig.MCPServer{}
	for name, server := range settings.MCPServers {
		s.config.MCPServers[name] = codexconfig.MCPServer{
			Type:     server.Type,
			Command:  server.Command,
			Args:     append([]string(nil), server.Args...),
			Env:      cloneStringMap(server.Env),
			URL:      server.URL,
			Headers:  cloneStringMap(server.Headers),
			Disabled: server.Disabled,
		}
	}
	s.hooks.Hooks = expandCodexHooks(settings.Hooks)
	return nil
}

func (s *codexStore) addHook(event string, hook Hook) error {
	if s.hooks.Hooks == nil {
		s.hooks.Hooks = map[string][]codexconfig.HookMatcher{}
	}
	entry := codexHookEntry(hook)
	matchers := s.hooks.Hooks[event]
	for i := range matchers {
		if matchers[i].Matcher == hook.Matcher {
			matchers[i].Hooks = append(matchers[i].Hooks, entry)
			s.hooks.Hooks[event] = matchers
			return nil
		}
	}
	s.hooks.Hooks[event] = append(matchers, codexconfig.HookMatcher{
		Matcher: hook.Matcher,
		Hooks:   []codexconfig.HookEntry{entry},
	})
	return nil
}

func (s *codexStore) removeHookIfMatches(event string, hook Hook) bool {
	matchers := s.hooks.Hooks[event]
	for i := range matchers {
		for j := range matchers[i].Hooks {
			if matchers[i].Matcher == hook.Matcher && reflect.DeepEqual(matchers[i].Hooks[j], codexHookEntry(hook)) {
				matchers[i].Hooks = append(matchers[i].Hooks[:j], matchers[i].Hooks[j+1:]...)
				if len(matchers[i].Hooks) == 0 {
					s.hooks.Hooks[event] = append(matchers[:i], matchers[i+1:]...)
				} else {
					s.hooks.Hooks[event] = matchers
				}
				return true
			}
		}
	}
	return false
}

func (s *codexStore) setEnv(_, _ string) error {
	return fmt.Errorf("%w: codex project env overrides", llmkit.ErrCapabilityNotSupported)
}

func (s *codexStore) removeEnvIfMatches(_, _ string) bool { return false }

func (s *codexStore) setMCP(name string, cfg llmkit.MCPServerConfig) error {
	if s.config.MCPServers == nil {
		s.config.MCPServers = map[string]codexconfig.MCPServer{}
	}
	s.config.MCPServers[name] = codexconfig.MCPServer{
		Type:     cfg.Type,
		Command:  cfg.Command,
		Args:     append([]string(nil), cfg.Args...),
		Env:      cloneStringMap(cfg.Env),
		URL:      cfg.URL,
		Headers:  cloneStringMap(cfg.Headers),
		Disabled: cfg.Disabled,
	}
	return nil
}

func (s *codexStore) removeMCPIfMatches(name string, cfg llmkit.MCPServerConfig) bool {
	current, ok := s.config.MCPServers[name]
	if !ok {
		return false
	}
	want := codexconfig.MCPServer{
		Type:     cfg.Type,
		Command:  cfg.Command,
		Args:     append([]string(nil), cfg.Args...),
		Env:      cloneStringMap(cfg.Env),
		URL:      cfg.URL,
		Headers:  cloneStringMap(cfg.Headers),
		Disabled: cfg.Disabled,
	}
	if !reflect.DeepEqual(current, want) {
		return false
	}
	delete(s.config.MCPServers, name)
	return true
}

func (s *codexStore) save() error {
	if err := writeTOMLAtomic(codexconfig.ProjectConfigPath(s.workDir), s.config); err != nil {
		return err
	}
	return writeJSONAtomic(codexconfig.HooksPath(s.workDir), s.hooks)
}

func (s *codexStore) paths() []string {
	return []string{
		codexconfig.ProjectConfigPath(s.workDir),
		codexconfig.HooksPath(s.workDir),
	}
}

func flattenClaudeHooks(in map[string][]claudeconfig.Hook) map[string][]Hook {
	out := map[string][]Hook{}
	for event, groups := range in {
		for _, group := range groups {
			for _, entry := range group.Hooks {
				out[event] = append(out[event], Hook{
					Matcher:       group.Matcher,
					Type:          entry.Type,
					Command:       entry.Command,
					URL:           entry.URL,
					Headers:       cloneStringMap(entry.Headers),
					Prompt:        entry.Prompt,
					Model:         entry.Model,
					Async:         entry.Async,
					Timeout:       entry.Timeout,
					StatusMessage: entry.StatusMessage,
					Once:          entry.Once,
				})
			}
		}
	}
	return out
}

func expandClaudeHooks(in map[string][]Hook) map[string][]claudeconfig.Hook {
	out := map[string][]claudeconfig.Hook{}
	for event, hooks := range in {
		for _, hook := range hooks {
			out[event] = append(out[event], claudeconfig.Hook{
				Matcher: hook.Matcher,
				Hooks:   []claudeconfig.HookEntry{claudeHookEntry(hook)},
			})
		}
	}
	return out
}

func claudeHookEntry(h Hook) claudeconfig.HookEntry {
	return claudeconfig.HookEntry{
		Type:          h.Type,
		Command:       h.Command,
		URL:           h.URL,
		Headers:       cloneStringMap(h.Headers),
		Prompt:        h.Prompt,
		Model:         h.Model,
		Async:         h.Async,
		Timeout:       h.Timeout,
		StatusMessage: h.StatusMessage,
		Once:          h.Once,
	}
}

func flattenCodexHooks(in map[string][]codexconfig.HookMatcher) map[string][]Hook {
	out := map[string][]Hook{}
	for event, groups := range in {
		for _, group := range groups {
			for _, entry := range group.Hooks {
				out[event] = append(out[event], Hook{
					Matcher:       group.Matcher,
					Type:          entry.Type,
					Command:       entry.Command,
					Timeout:       entry.Timeout,
					StatusMessage: entry.StatusMessage,
				})
			}
		}
	}
	return out
}

func expandCodexHooks(in map[string][]Hook) map[string][]codexconfig.HookMatcher {
	out := map[string][]codexconfig.HookMatcher{}
	for event, hooks := range in {
		for _, hook := range hooks {
			out[event] = append(out[event], codexconfig.HookMatcher{
				Matcher: hook.Matcher,
				Hooks:   []codexconfig.HookEntry{codexHookEntry(hook)},
			})
		}
	}
	return out
}

func codexHookEntry(h Hook) codexconfig.HookEntry {
	return codexconfig.HookEntry{
		Type:          h.Type,
		Command:       h.Command,
		Timeout:       h.Timeout,
		StatusMessage: h.StatusMessage,
	}
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data)
}

func writeTOMLAtomic(path string, value any) error {
	if cfg, ok := value.(*codexconfig.ConfigFile); ok {
		data, err := codexconfig.MarshalConfig(cfg)
		if err != nil {
			return err
		}
		return writeFileAtomic(path, data)
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}
	return writeFileAtomic(path, buf.Bytes())
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".llmkit-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}

func cloneMCPServer(in llmkit.MCPServerConfig) llmkit.MCPServerConfig {
	return llmkit.MCPServerConfig{
		Type:     in.Type,
		Command:  in.Command,
		Args:     append([]string(nil), in.Args...),
		Env:      cloneStringMap(in.Env),
		URL:      in.URL,
		Headers:  cloneStringMap(in.Headers),
		Disabled: in.Disabled,
	}
}

func mapHeadersToSlice(in map[string]string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for key, value := range in {
		out = append(out, key+": "+value)
	}
	return out
}

func sliceHeadersToMap(in []string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for _, item := range in {
		parts := bytes.SplitN([]byte(item), []byte(":"), 2)
		if len(parts) != 2 {
			continue
		}
		out[string(bytes.TrimSpace(parts[0]))] = string(bytes.TrimSpace(parts[1]))
	}
	return out
}
