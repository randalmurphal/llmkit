package claudeconfig

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/randalmurphal/llmkit/claudecontract"
)

// MCPServer represents an MCP server configuration.
// Supports stdio, http, and sse transport types.
type MCPServer struct {
	// Type specifies the transport type: "stdio", "http", or "sse"
	// If empty, defaults to "stdio" for servers with Command set
	Type string `json:"type,omitempty"`

	// Command is the executable to run (for stdio transport)
	Command string `json:"command,omitempty"`

	// Args are command-line arguments (for stdio transport)
	Args []string `json:"args,omitempty"`

	// Env contains environment variables for the server
	// Supports ${VAR} and ${VAR:-default} syntax
	Env map[string]string `json:"env,omitempty"`

	// URL is the server endpoint (for http/sse transport)
	URL string `json:"url,omitempty"`

	// Headers are HTTP headers (for http/sse transport)
	// Format: ["Authorization: Bearer ${TOKEN}"]
	Headers []string `json:"headers,omitempty"`

	// Disabled indicates if the server should be skipped
	Disabled bool `json:"disabled,omitempty"`
}

// MCPConfig represents the .mcp.json file structure.
type MCPConfig struct {
	MCPServers map[string]*MCPServer `json:"mcpServers,omitempty"`
}

// NewMCPConfig creates an empty MCP config with initialized maps.
func NewMCPConfig() *MCPConfig {
	return &MCPConfig{
		MCPServers: make(map[string]*MCPServer),
	}
}

// Clone creates a deep copy of the MCP config.
func (c *MCPConfig) Clone() *MCPConfig {
	if c == nil {
		return nil
	}

	clone := NewMCPConfig()
	for name, server := range c.MCPServers {
		clone.MCPServers[name] = server.Clone()
	}
	return clone
}

// Clone creates a deep copy of the MCP server.
func (s *MCPServer) Clone() *MCPServer {
	if s == nil {
		return nil
	}

	clone := &MCPServer{
		Type:     s.Type,
		Command:  s.Command,
		URL:      s.URL,
		Disabled: s.Disabled,
	}

	if s.Args != nil {
		clone.Args = make([]string, len(s.Args))
		copy(clone.Args, s.Args)
	}

	if s.Env != nil {
		clone.Env = make(map[string]string, len(s.Env))
		maps.Copy(clone.Env, s.Env)
	}

	if s.Headers != nil {
		clone.Headers = make([]string, len(s.Headers))
		copy(clone.Headers, s.Headers)
	}

	return clone
}

// GetTransportType returns the effective transport type.
// Returns "stdio" if Type is empty and Command is set.
func (s *MCPServer) GetTransportType() string {
	if s.Type != "" {
		return s.Type
	}
	if s.Command != "" {
		return "stdio"
	}
	if s.URL != "" {
		return "http"
	}
	return "stdio"
}

// IsValid checks if the server configuration is valid.
func (s *MCPServer) IsValid() error {
	transport := s.GetTransportType()

	switch transport {
	case "stdio":
		if s.Command == "" {
			return fmt.Errorf("stdio transport requires command")
		}
	case "http", "sse":
		if s.URL == "" {
			return fmt.Errorf("%s transport requires url", transport)
		}
	default:
		return fmt.Errorf("unsupported transport type: %s", transport)
	}

	return nil
}

// AddServer adds or updates an MCP server in the config.
func (c *MCPConfig) AddServer(name string, server *MCPServer) error {
	if name == "" {
		return fmt.Errorf("server name is required")
	}
	if server == nil {
		return fmt.Errorf("server config is required")
	}
	if err := server.IsValid(); err != nil {
		return fmt.Errorf("invalid server config: %w", err)
	}

	if c.MCPServers == nil {
		c.MCPServers = make(map[string]*MCPServer)
	}
	c.MCPServers[name] = server
	return nil
}

// RemoveServer removes an MCP server from the config.
func (c *MCPConfig) RemoveServer(name string) bool {
	if c.MCPServers == nil {
		return false
	}
	if _, exists := c.MCPServers[name]; !exists {
		return false
	}
	delete(c.MCPServers, name)
	return true
}

// GetServer returns an MCP server by name.
func (c *MCPConfig) GetServer(name string) *MCPServer {
	if c.MCPServers == nil {
		return nil
	}
	return c.MCPServers[name]
}

// ListServers returns all server names.
func (c *MCPConfig) ListServers() []string {
	if c.MCPServers == nil {
		return nil
	}
	names := make([]string, 0, len(c.MCPServers))
	for name := range c.MCPServers {
		names = append(names, name)
	}
	return names
}

// LoadProjectMCPConfig loads MCP config from {projectRoot}/.mcp.json.
// Returns empty config if file doesn't exist.
func LoadProjectMCPConfig(projectRoot string) (*MCPConfig, error) {
	path := MCPConfigPath(projectRoot)
	return loadMCPConfigFile(path)
}

// loadMCPConfigFile loads MCP config from a specific file path.
func loadMCPConfigFile(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewMCPConfig(), nil
		}
		return nil, fmt.Errorf("read MCP config file: %w", err)
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse MCP config JSON: %w", err)
	}

	// Initialize nil map
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]*MCPServer)
	}

	return &config, nil
}

// SaveProjectMCPConfig saves MCP config to {projectRoot}/.mcp.json.
func SaveProjectMCPConfig(projectRoot string, config *MCPConfig) error {
	path := MCPConfigPath(projectRoot)
	return saveMCPConfigFile(path, config)
}

// saveMCPConfigFile saves MCP config to a specific file path.
func saveMCPConfigFile(path string, config *MCPConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal MCP config: %w", err)
	}

	// Add trailing newline
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write MCP config file: %w", err)
	}

	return nil
}

// MCPConfigPath returns the path to the project MCP config file.
func MCPConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, claudecontract.FileMCPConfig)
}

// MCPConfigExists checks if a project MCP config file exists.
func MCPConfigExists(projectRoot string) bool {
	path := MCPConfigPath(projectRoot)
	_, err := os.Stat(path)
	return err == nil
}

// MCPServerInfo provides summary information about an MCP server.
type MCPServerInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Command   string `json:"command,omitempty"`
	URL       string `json:"url,omitempty"`
	Disabled  bool   `json:"disabled"`
	HasEnv    bool   `json:"has_env"`
	EnvCount  int    `json:"env_count"`
	ArgsCount int    `json:"args_count"`
}

// GetServerInfo returns summary information about a server.
func (c *MCPConfig) GetServerInfo(name string) *MCPServerInfo {
	server := c.GetServer(name)
	if server == nil {
		return nil
	}

	return &MCPServerInfo{
		Name:      name,
		Type:      server.GetTransportType(),
		Command:   server.Command,
		URL:       server.URL,
		Disabled:  server.Disabled,
		HasEnv:    len(server.Env) > 0,
		EnvCount:  len(server.Env),
		ArgsCount: len(server.Args),
	}
}

// ListServerInfos returns summary information for all servers.
func (c *MCPConfig) ListServerInfos() []*MCPServerInfo {
	if c.MCPServers == nil {
		return nil
	}

	infos := make([]*MCPServerInfo, 0, len(c.MCPServers))
	for name := range c.MCPServers {
		if info := c.GetServerInfo(name); info != nil {
			infos = append(infos, info)
		}
	}
	return infos
}

// Merge combines two MCP configs, with override taking precedence.
func (c *MCPConfig) Merge(override *MCPConfig) *MCPConfig {
	if c == nil {
		return override.Clone()
	}
	if override == nil {
		return c.Clone()
	}

	result := c.Clone()
	for name, server := range override.MCPServers {
		result.MCPServers[name] = server.Clone()
	}
	return result
}
