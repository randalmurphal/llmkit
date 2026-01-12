// Package claudeconfig provides utilities for parsing Claude Code's native configuration formats.
package claudeconfig

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Agent represents a Claude Code agent parsed from an agent .md file.
// The file format uses YAML frontmatter followed by markdown content.
// This is similar to skills but stored in the agents/ directory.
type Agent struct {
	// Frontmatter fields (from YAML between --- delimiters)
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Tools       string `yaml:"tools,omitempty" json:"tools,omitempty"` // Comma-separated tool names

	// Content is the markdown body after the frontmatter
	Content string `yaml:"-" json:"content"`

	// Path is the file path of the agent .md file
	Path string `yaml:"-" json:"path"`
}

// AgentInfo provides summary information for listing agents.
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tools       string `json:"tools,omitempty"`
	Path        string `json:"path"`
}

// Info returns summary information for this agent.
func (a *Agent) Info() AgentInfo {
	return AgentInfo{
		Name:        a.Name,
		Description: a.Description,
		Tools:       a.Tools,
		Path:        a.Path,
	}
}

// ToolsList returns the tools as a slice of strings.
func (a *Agent) ToolsList() []string {
	if a.Tools == "" {
		return nil
	}
	parts := strings.Split(a.Tools, ",")
	tools := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tools = append(tools, t)
		}
	}
	return tools
}

// Validate checks that the agent has required fields.
func (a *Agent) Validate() error {
	if a.Name == "" {
		return errors.New("agent name is required")
	}
	if a.Description == "" {
		return errors.New("agent description is required")
	}
	return nil
}

// ParseAgentMD reads and parses an agent .md file from the given path.
func ParseAgentMD(path string) (*Agent, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent file: %w", err)
	}

	// Parse the content
	agent, err := parseAgentContent(data)
	if err != nil {
		return nil, fmt.Errorf("parse agent file: %w", err)
	}

	agent.Path = path

	return agent, nil
}

// parseAgentContent parses the agent .md content with YAML frontmatter.
func parseAgentContent(data []byte) (*Agent, error) {
	// Check for frontmatter delimiter
	if !bytes.HasPrefix(data, []byte("---")) {
		return nil, errors.New("agent file must start with YAML frontmatter (---)")
	}

	// Find the end of frontmatter
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var frontmatterLines []string
	var contentLines []string
	inFrontmatter := false
	foundEnd := false

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if lineNum == 1 && line == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter && line == "---" {
			inFrontmatter = false
			foundEnd = true
			continue
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if foundEnd {
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan content: %w", err)
	}

	if !foundEnd {
		return nil, errors.New("agent file frontmatter not closed (missing ---)")
	}

	// Parse YAML frontmatter
	agent := &Agent{}
	frontmatter := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(frontmatter), agent); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	// Set content (trim leading/trailing blank lines)
	agent.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))

	return agent, nil
}

// DiscoverAgents finds all agent .md files in the given .claude directory.
// It searches in the agents/ subdirectory for any .md files with YAML frontmatter.
func DiscoverAgents(claudeDir string) ([]*Agent, error) {
	agentsDir := filepath.Join(claudeDir, "agents")

	// Check if agents directory exists
	if !dirExists(agentsDir) {
		return nil, nil // No agents directory, return empty list
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("read agents directory: %w", err)
	}

	var agents []*Agent
	for _, entry := range entries {
		// Skip directories, only process .md files
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		agentPath := filepath.Join(agentsDir, entry.Name())

		agent, err := ParseAgentMD(agentPath)
		if err != nil {
			// Log warning but continue discovering other agents
			continue
		}

		agents = append(agents, agent)
	}

	// Sort by name for consistent ordering
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	return agents, nil
}

// WriteAgentMD writes an agent to an .md file in the given directory.
func WriteAgentMD(agent *Agent, dir string) error {
	if err := agent.Validate(); err != nil {
		return fmt.Errorf("validate agent: %w", err)
	}

	// Create directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Build frontmatter
	frontmatter := struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Tools       string `yaml:"tools,omitempty"`
	}{
		Name:        agent.Name,
		Description: agent.Description,
		Tools:       agent.Tools,
	}

	fmBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("marshal frontmatter: %w", err)
	}

	// Build full content
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	buf.WriteString("---\n\n")
	buf.WriteString(agent.Content)
	if !strings.HasSuffix(agent.Content, "\n") {
		buf.WriteString("\n")
	}

	// Write file - use sanitized name for filename
	filename := strings.ReplaceAll(agent.Name, " ", "-") + ".md"
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write agent file: %w", err)
	}

	return nil
}
