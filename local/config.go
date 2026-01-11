// Package local provides a client for local LLM models via a Python sidecar process.
//
// The local provider communicates with a Python sidecar via JSON-RPC 2.0 over stdio.
// This enables llmkit to work with local model backends like Ollama, llama.cpp, vLLM,
// and HuggingFace transformers without requiring direct Go bindings.
//
// # Architecture
//
// The local client manages a long-running Python sidecar process:
//
//	Go Client <--JSON-RPC/stdio--> Python Sidecar <--Backend API--> Local Model
//
// # Supported Backends
//
//   - ollama: Ollama API server
//   - llama.cpp: llama.cpp server
//   - vllm: vLLM server
//   - transformers: HuggingFace transformers
//
// # Usage
//
//	client, err := provider.New("local", provider.Config{
//	    Model: "llama3.2:latest",
//	    Options: map[string]any{
//	        "backend":      "ollama",
//	        "sidecar_path": "/path/to/sidecar.py",
//	    },
//	})
package local

import (
	"fmt"
	"time"
)

// Backend identifies the local model backend.
type Backend string

// Supported backends.
const (
	BackendOllama       Backend = "ollama"
	BackendLlamaCpp     Backend = "llama.cpp"
	BackendVLLM         Backend = "vllm"
	BackendTransformers Backend = "transformers"
)

// Config holds local provider configuration.
type Config struct {
	// Backend specifies which local model backend to use.
	// Required. One of: "ollama", "llama.cpp", "vllm", "transformers"
	Backend Backend `json:"backend" yaml:"backend"`

	// SidecarPath is the path to the Python sidecar script.
	// Required.
	SidecarPath string `json:"sidecar_path" yaml:"sidecar_path"`

	// Model is the local model name to use.
	// Format depends on backend (e.g., "llama3.2:latest" for Ollama).
	Model string `json:"model" yaml:"model"`

	// Host is the API server address for applicable backends.
	// Default: "localhost:11434" for Ollama, "localhost:8000" for vLLM.
	Host string `json:"host" yaml:"host"`

	// PythonPath is the path to the Python interpreter.
	// Default: "python3"
	PythonPath string `json:"python_path" yaml:"python_path"`

	// StartupTimeout is how long to wait for sidecar to become ready.
	// Default: 30 seconds.
	StartupTimeout time.Duration `json:"startup_timeout" yaml:"startup_timeout"`

	// RequestTimeout is the default timeout for completion requests.
	// Default: 5 minutes.
	RequestTimeout time.Duration `json:"request_timeout" yaml:"request_timeout"`

	// WorkDir is the working directory for the sidecar process.
	WorkDir string `json:"work_dir" yaml:"work_dir"`

	// Env provides additional environment variables for the sidecar.
	Env map[string]string `json:"env" yaml:"env"`

	// MCPServers configures MCP servers to pass through to the sidecar.
	// The sidecar is responsible for connecting to MCP servers.
	MCPServers map[string]MCPServerConfig `json:"mcp_servers" yaml:"mcp_servers"`
}

// MCPServerConfig defines an MCP server to enable.
type MCPServerConfig struct {
	Type    string            `json:"type"`              // "stdio", "http", "sse"
	Command string            `json:"command,omitempty"` // For stdio transport
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"` // For http/sse transport
	Headers []string          `json:"headers,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		PythonPath:     "python3",
		StartupTimeout: 30 * time.Second,
		RequestTimeout: 5 * time.Minute,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Backend == "" {
		return fmt.Errorf("backend is required")
	}

	switch c.Backend {
	case BackendOllama, BackendLlamaCpp, BackendVLLM, BackendTransformers:
		// Valid backend
	default:
		return fmt.Errorf("unknown backend %q, expected one of: ollama, llama.cpp, vllm, transformers", c.Backend)
	}

	if c.SidecarPath == "" {
		return fmt.Errorf("sidecar_path is required")
	}

	if c.Model == "" {
		return fmt.Errorf("model is required")
	}

	if c.StartupTimeout < 0 {
		return fmt.Errorf("startup_timeout must be >= 0")
	}

	if c.RequestTimeout < 0 {
		return fmt.Errorf("request_timeout must be >= 0")
	}

	return nil
}

// WithDefaults returns a copy of the config with defaults applied for unset fields.
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()

	if c.PythonPath == "" {
		c.PythonPath = defaults.PythonPath
	}
	if c.StartupTimeout == 0 {
		c.StartupTimeout = defaults.StartupTimeout
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaults.RequestTimeout
	}
	if c.Host == "" {
		switch c.Backend {
		case BackendOllama:
			c.Host = "localhost:11434"
		case BackendVLLM, BackendLlamaCpp:
			c.Host = "localhost:8000"
		}
	}

	return c
}

// Option configures a local Client.
type Option func(*Client)

// WithBackend sets the backend type.
func WithBackend(backend Backend) Option {
	return func(c *Client) { c.cfg.Backend = backend }
}

// WithSidecarPath sets the path to the sidecar script.
func WithSidecarPath(path string) Option {
	return func(c *Client) { c.cfg.SidecarPath = path }
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(c *Client) { c.cfg.Model = model }
}

// WithHost sets the backend API server address.
func WithHost(host string) Option {
	return func(c *Client) { c.cfg.Host = host }
}

// WithPythonPath sets the Python interpreter path.
func WithPythonPath(path string) Option {
	return func(c *Client) { c.cfg.PythonPath = path }
}

// WithStartupTimeout sets the sidecar startup timeout.
func WithStartupTimeout(d time.Duration) Option {
	return func(c *Client) { c.cfg.StartupTimeout = d }
}

// WithRequestTimeout sets the default request timeout.
func WithRequestTimeout(d time.Duration) Option {
	return func(c *Client) { c.cfg.RequestTimeout = d }
}

// WithWorkDir sets the working directory for the sidecar.
func WithWorkDir(dir string) Option {
	return func(c *Client) { c.cfg.WorkDir = dir }
}

// WithEnv adds environment variables for the sidecar process.
func WithEnv(env map[string]string) Option {
	return func(c *Client) {
		if c.cfg.Env == nil {
			c.cfg.Env = make(map[string]string)
		}
		for k, v := range env {
			c.cfg.Env[k] = v
		}
	}
}

// WithMCPServers configures MCP servers for the sidecar.
func WithMCPServers(servers map[string]MCPServerConfig) Option {
	return func(c *Client) { c.cfg.MCPServers = servers }
}
