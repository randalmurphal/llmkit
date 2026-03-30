package codexconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ConfigFile struct {
	Profiles                    map[string]Profile   `toml:"profiles,omitempty"`
	MCPServers                  map[string]MCPServer `toml:"mcp_servers,omitempty"`
	Skills                      SkillsSettings       `toml:"skills,omitempty"`
	Agents                      AgentsSettings       `toml:"agents,omitempty"`
	ModelInstructionsFile       string               `toml:"model_instructions_file,omitempty"`
	ProjectDocFallbackFilenames []string             `toml:"project_doc_fallback_filenames,omitempty"`
	ProjectDocMaxBytes          int                  `toml:"project_doc_max_bytes,omitempty"`
	ProjectRootMarkers          []string             `toml:"project_root_markers,omitempty"`

	raw map[string]any `toml:"-"`
}

type Profile struct {
	Model                       string               `toml:"model,omitempty"`
	ModelReasoningEffort        string               `toml:"model_reasoning_effort,omitempty"`
	SandboxMode                 string               `toml:"sandbox_mode,omitempty"`
	ApprovalPolicy              string               `toml:"approval_policy,omitempty"`
	MCPServers                  map[string]MCPServer `toml:"mcp_servers,omitempty"`
	ProjectDocFallbackFilenames []string             `toml:"project_doc_fallback_filenames,omitempty"`
}

type SkillsSettings struct {
	Config []SkillToggle `toml:"config,omitempty"`
}

type AgentsSettings struct {
	MaxThreads           int `toml:"max_threads,omitempty"`
	MaxDepth             int `toml:"max_depth,omitempty"`
	JobMaxRuntimeSeconds int `toml:"job_max_runtime_seconds,omitempty"`
}

type MCPServer struct {
	Command  string            `toml:"command,omitempty" json:"command,omitempty"`
	Args     []string          `toml:"args,omitempty" json:"args,omitempty"`
	Env      map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	URL      string            `toml:"url,omitempty" json:"url,omitempty"`
	Type     string            `toml:"type,omitempty" json:"type,omitempty"`
	Headers  map[string]string `toml:"headers,omitempty" json:"headers,omitempty"`
	Disabled bool              `toml:"disabled,omitempty" json:"disabled,omitempty"`
}

func LoadProjectConfig(projectRoot string) (*ConfigFile, error) {
	return loadConfig(ProjectConfigPath(projectRoot))
}

func LoadUserConfig() (*ConfigFile, error) {
	path, err := UserConfigPath()
	if err != nil {
		return nil, err
	}
	return loadConfig(path)
}

func SaveProjectConfig(projectRoot string, cfg *ConfigFile) error {
	return saveConfig(ProjectConfigPath(projectRoot), cfg)
}

func SaveUserConfig(cfg *ConfigFile) error {
	path, err := UserConfigPath()
	if err != nil {
		return err
	}
	return saveConfig(path, cfg)
}

func loadConfig(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ConfigFile{}, nil
		}
		return nil, fmt.Errorf("read config.toml: %w", err)
	}

	var cfg ConfigFile
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.toml: %w", err)
	}
	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, fmt.Errorf("decode config.toml: %w", err)
	}
	cfg.raw = raw
	return &cfg, nil
}

func saveConfig(path string, cfg *ConfigFile) error {
	if cfg == nil {
		cfg = &ConfigFile{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := MarshalConfig(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func MarshalConfig(cfg *ConfigFile) ([]byte, error) {
	if cfg == nil {
		cfg = &ConfigFile{}
	}
	doc := cloneAnyMap(cfg.raw)
	if doc == nil {
		doc = map[string]any{}
	}

	known, err := knownConfigMap(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode known config: %w", err)
	}
	for _, key := range []string{
		"profiles",
		"mcp_servers",
		"skills",
		"agents",
		"model_instructions_file",
		"project_doc_fallback_filenames",
		"project_doc_max_bytes",
		"project_root_markers",
	} {
		value, ok := known[key]
		if !ok {
			delete(doc, key)
			continue
		}
		if existing, ok := doc[key].(map[string]any); ok {
			if incoming, ok := value.(map[string]any); ok {
				if key == "profiles" || key == "mcp_servers" {
					doc[key] = replaceAnyMap(existing, incoming)
					continue
				}
				doc[key] = mergeAnyMap(existing, incoming)
				continue
			}
		}
		doc[key] = value
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(doc); err != nil {
		return nil, fmt.Errorf("write config.toml: %w", err)
	}
	return buf.Bytes(), nil
}

func knownConfigMap(cfg *ConfigFile) (map[string]any, error) {
	typed := struct {
		Profiles                    map[string]Profile   `toml:"profiles,omitempty"`
		MCPServers                  map[string]MCPServer `toml:"mcp_servers,omitempty"`
		Skills                      SkillsSettings       `toml:"skills,omitempty"`
		Agents                      AgentsSettings       `toml:"agents,omitempty"`
		ModelInstructionsFile       string               `toml:"model_instructions_file,omitempty"`
		ProjectDocFallbackFilenames []string             `toml:"project_doc_fallback_filenames,omitempty"`
		ProjectDocMaxBytes          int                  `toml:"project_doc_max_bytes,omitempty"`
		ProjectRootMarkers          []string             `toml:"project_root_markers,omitempty"`
	}{
		Profiles:                    cfg.Profiles,
		MCPServers:                  cfg.MCPServers,
		Skills:                      cfg.Skills,
		Agents:                      cfg.Agents,
		ModelInstructionsFile:       cfg.ModelInstructionsFile,
		ProjectDocFallbackFilenames: cfg.ProjectDocFallbackFilenames,
		ProjectDocMaxBytes:          cfg.ProjectDocMaxBytes,
		ProjectRootMarkers:          cfg.ProjectRootMarkers,
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(typed); err != nil {
		return nil, err
	}
	var out map[string]any
	if _, err := toml.Decode(buf.String(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if child, ok := value.(map[string]any); ok {
			out[key] = cloneAnyMap(child)
			continue
		}
		out[key] = value
	}
	return out
}

func mergeAnyMap(existing, incoming map[string]any) map[string]any {
	out := cloneAnyMap(existing)
	if out == nil {
		out = map[string]any{}
	}
	for key, value := range incoming {
		if existingChild, ok := out[key].(map[string]any); ok {
			if incomingChild, ok := value.(map[string]any); ok {
				out[key] = mergeAnyMap(existingChild, incomingChild)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func replaceAnyMap(existing, incoming map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range incoming {
		if existingChild, ok := existing[key].(map[string]any); ok {
			if incomingChild, ok := value.(map[string]any); ok {
				out[key] = mergeAnyMap(existingChild, incomingChild)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func (c *ConfigFile) SkillEnabled(path string) (bool, bool) {
	if c == nil {
		return true, false
	}
	for _, entry := range c.Skills.Config {
		if cleanPath(entry.Path) == cleanPath(path) {
			return entry.Enabled, true
		}
	}
	return true, false
}

func (c *ConfigFile) SetSkillEnabled(path string, enabled bool) {
	if c == nil {
		return
	}
	for i := range c.Skills.Config {
		if cleanPath(c.Skills.Config[i].Path) == cleanPath(path) {
			c.Skills.Config[i].Enabled = enabled
			return
		}
	}
	c.Skills.Config = append(c.Skills.Config, SkillToggle{Path: path, Enabled: enabled})
}
