package codexconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type CustomAgent struct {
	Name                  string               `toml:"name" json:"name"`
	Description           string               `toml:"description" json:"description"`
	DeveloperInstructions string               `toml:"developer_instructions" json:"developer_instructions"`
	NicknameCandidates    []string             `toml:"nickname_candidates,omitempty" json:"nickname_candidates,omitempty"`
	Model                 string               `toml:"model,omitempty" json:"model,omitempty"`
	ModelReasoningEffort  string               `toml:"model_reasoning_effort,omitempty" json:"model_reasoning_effort,omitempty"`
	SandboxMode           string               `toml:"sandbox_mode,omitempty" json:"sandbox_mode,omitempty"`
	MCPServers            map[string]MCPServer `toml:"mcp_servers,omitempty" json:"mcp_servers,omitempty"`
	Skills                SkillsSettings       `toml:"skills,omitempty" json:"skills,omitempty"`
	Path                  string               `toml:"-" json:"path"`
}

func (a *CustomAgent) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if a.Description == "" {
		return fmt.Errorf("agent description is required")
	}
	if a.DeveloperInstructions == "" {
		return fmt.Errorf("developer_instructions is required")
	}
	return nil
}

func ParseCustomAgent(path string) (*CustomAgent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read custom agent: %w", err)
	}
	var agent CustomAgent
	if err := toml.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("parse custom agent: %w", err)
	}
	agent.Path = path
	return &agent, nil
}

func DiscoverCustomAgents(projectRoot string) ([]*CustomAgent, error) {
	var agents []*CustomAgent
	userDir, err := UserAgentsDir()
	if err == nil {
		found, err := discoverAgentsInDir(userDir)
		if err != nil {
			return nil, err
		}
		agents = append(agents, found...)
	}
	found, err := discoverAgentsInDir(ProjectAgentsDir(projectRoot))
	if err != nil {
		return nil, err
	}
	agents = append(agents, found...)

	sort.Slice(agents, func(i, j int) bool {
		if agents[i].Name == agents[j].Name {
			return agents[i].Path < agents[j].Path
		}
		return agents[i].Name < agents[j].Name
	})
	return agents, nil
}

func discoverAgentsInDir(dir string) ([]*CustomAgent, error) {
	if !dirExists(dir) {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read agents dir: %w", err)
	}
	var agents []*CustomAgent
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		agent, err := ParseCustomAgent(path)
		if err != nil {
			continue
		}
		agents = append(agents, agent)
	}
	return agents, nil
}
