package claudeconfig

import (
	"errors"
	"fmt"
	"sort"
)

// SubAgent errors
var (
	ErrSubAgentNameRequired        = errors.New("sub-agent name is required")
	ErrSubAgentDescriptionRequired = errors.New("sub-agent description is required")
	ErrSubAgentNotFound            = errors.New("sub-agent not found")
	ErrSubAgentAlreadyExists       = errors.New("sub-agent already exists")
)

// SubAgent defines a reusable agent configuration for Claude Code.
// Sub-agents can be invoked during tasks to delegate work with specific configurations.
type SubAgent struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Model       string           `json:"model,omitempty"`      // Default: inherit from parent
	Tools       *ToolPermissions `json:"tools,omitempty"`      // Tool restrictions for this agent
	Prompt      string           `json:"prompt,omitempty"`     // System prompt override
	WorkDir     string           `json:"work_dir,omitempty"`   // Working directory override
	SkillRefs   []string         `json:"skill_refs,omitempty"` // Skills to load for this agent
	Timeout     string           `json:"timeout,omitempty"`    // Execution timeout (e.g., "5m")
}

// Validate checks that the sub-agent has required fields.
func (a *SubAgent) Validate() error {
	if a.Name == "" {
		return ErrSubAgentNameRequired
	}
	if a.Description == "" {
		return ErrSubAgentDescriptionRequired
	}
	return nil
}

// AgentService manages sub-agent configurations stored in .claude/settings.json.
type AgentService struct {
	projectRoot   string
	extensionName string // Key in settings.json extensions (default: "agents")
}

// AgentServiceOption configures the AgentService.
type AgentServiceOption func(*AgentService)

// WithAgentExtensionName sets a custom extension name for storing agents.
func WithAgentExtensionName(name string) AgentServiceOption {
	return func(s *AgentService) {
		s.extensionName = name
	}
}

// NewAgentService creates a new agent service for the given project root.
func NewAgentService(projectRoot string, opts ...AgentServiceOption) *AgentService {
	s := &AgentService{
		projectRoot:   projectRoot,
		extensionName: "agents",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// List returns all registered sub-agents.
func (s *AgentService) List() ([]SubAgent, error) {
	agents, err := s.loadAgents()
	if err != nil {
		return nil, fmt.Errorf("load agents: %w", err)
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	return agents, nil
}

// Get returns a sub-agent by name.
func (s *AgentService) Get(name string) (*SubAgent, error) {
	agents, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, agent := range agents {
		if agent.Name == name {
			return &agent, nil
		}
	}

	return nil, ErrSubAgentNotFound
}

// Create adds a new sub-agent.
func (s *AgentService) Create(agent SubAgent) error {
	if err := agent.Validate(); err != nil {
		return fmt.Errorf("validate agent: %w", err)
	}

	agents, err := s.loadAgents()
	if err != nil {
		return fmt.Errorf("load agents: %w", err)
	}

	// Check for duplicates
	for _, existing := range agents {
		if existing.Name == agent.Name {
			return ErrSubAgentAlreadyExists
		}
	}

	agents = append(agents, agent)

	if err := s.saveAgents(agents); err != nil {
		return fmt.Errorf("save agents: %w", err)
	}

	return nil
}

// Update modifies an existing sub-agent.
func (s *AgentService) Update(name string, agent SubAgent) error {
	if err := agent.Validate(); err != nil {
		return fmt.Errorf("validate agent: %w", err)
	}

	agents, err := s.loadAgents()
	if err != nil {
		return fmt.Errorf("load agents: %w", err)
	}

	found := false
	for i, existing := range agents {
		if existing.Name == name {
			// Check for rename conflicts
			if agent.Name != name {
				for _, other := range agents {
					if other.Name == agent.Name {
						return ErrSubAgentAlreadyExists
					}
				}
			}
			agents[i] = agent
			found = true
			break
		}
	}

	if !found {
		return ErrSubAgentNotFound
	}

	if err := s.saveAgents(agents); err != nil {
		return fmt.Errorf("save agents: %w", err)
	}

	return nil
}

// Delete removes a sub-agent by name.
func (s *AgentService) Delete(name string) error {
	agents, err := s.loadAgents()
	if err != nil {
		return fmt.Errorf("load agents: %w", err)
	}

	found := false
	result := make([]SubAgent, 0, len(agents))
	for _, agent := range agents {
		if agent.Name == name {
			found = true
			continue
		}
		result = append(result, agent)
	}

	if !found {
		return ErrSubAgentNotFound
	}

	if err := s.saveAgents(result); err != nil {
		return fmt.Errorf("save agents: %w", err)
	}

	return nil
}

// Exists checks if a sub-agent with the given name exists.
func (s *AgentService) Exists(name string) bool {
	_, err := s.Get(name)
	return err == nil
}

// loadAgents loads sub-agents from settings.json extension.
func (s *AgentService) loadAgents() ([]SubAgent, error) {
	settings, err := LoadProjectSettings(s.projectRoot)
	if err != nil {
		return nil, err
	}

	var agents []SubAgent
	if err := settings.GetExtension(s.extensionName, &agents); err != nil {
		return nil, fmt.Errorf("get %s extension: %w", s.extensionName, err)
	}

	if agents == nil {
		agents = []SubAgent{}
	}

	return agents, nil
}

// saveAgents saves sub-agents to settings.json extension.
func (s *AgentService) saveAgents(agents []SubAgent) error {
	settings, err := LoadProjectSettings(s.projectRoot)
	if err != nil {
		return err
	}

	settings.SetExtension(s.extensionName, agents)

	if err := SaveProjectSettings(s.projectRoot, settings); err != nil {
		return fmt.Errorf("save project settings: %w", err)
	}

	return nil
}
