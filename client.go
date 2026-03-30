package llmkit

import "context"

// Client is the unified interface for Claude and Codex clients.
// Implementations must be safe for concurrent use.
type Client interface {
	Complete(ctx context.Context, req Request) (*Response, error)
	Stream(ctx context.Context, req Request) (<-chan StreamChunk, error)
	Provider() string
	Capabilities() Capabilities
	Close() error
}

// RuntimeCapabilities describes direct execution features available from a provider.
type RuntimeCapabilities struct {
	Streaming   bool     `json:"streaming"`
	Tools       bool     `json:"tools"`
	Sessions    bool     `json:"sessions"`
	Images      bool     `json:"images"`
	NativeTools []string `json:"native_tools,omitempty"`
}

// EnvironmentCapabilities describes provider-local ecosystem support.
type EnvironmentCapabilities struct {
	ConfigFile   string `json:"config_file,omitempty"`
	ContextFile  string `json:"context_file,omitempty"`
	Hooks        bool   `json:"hooks"`
	MCP          bool   `json:"mcp"`
	Skills       bool   `json:"skills"`
	Plugins      bool   `json:"plugins"`
	Instructions bool   `json:"instructions"`
	CustomAgents bool   `json:"custom_agents"`
	Rules        bool   `json:"rules"`
}

// Capabilities describes what a provider supports at runtime and in its local ecosystem.
type Capabilities struct {
	Runtime     RuntimeCapabilities     `json:"runtime"`
	Environment EnvironmentCapabilities `json:"environment"`
}

// HasTool checks if a native runtime tool is available by name.
func (c Capabilities) HasTool(name string) bool {
	for _, t := range c.Runtime.NativeTools {
		if t == name {
			return true
		}
	}
	return false
}

var (
	ClaudeCapabilities = Capabilities{
		Runtime: RuntimeCapabilities{
			Streaming:   true,
			Tools:       true,
			Sessions:    true,
			Images:      true,
			NativeTools: []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash", "Task", "TodoWrite", "WebFetch", "WebSearch", "AskUserQuestion", "NotebookEdit", "LSP", "Skill", "EnterPlanMode", "ExitPlanMode", "KillShell", "TaskOutput"},
		},
		Environment: EnvironmentCapabilities{
			ContextFile:  "CLAUDE.md",
			Hooks:        true,
			MCP:          true,
			Skills:       true,
			Plugins:      true,
			Instructions: true,
			CustomAgents: true,
		},
	}

	CodexCapabilities = Capabilities{
		Runtime: RuntimeCapabilities{
			Streaming:   true,
			Tools:       true,
			Sessions:    true,
			Images:      true,
			NativeTools: []string{"shell", "apply_patch", "read_file", "list_dir", "web_search"},
		},
		Environment: EnvironmentCapabilities{
			ConfigFile:   "config.toml",
			ContextFile:  "AGENTS.md",
			Hooks:        true,
			MCP:          true,
			Skills:       true,
			Plugins:      true,
			Instructions: true,
			CustomAgents: true,
			Rules:        true,
		},
	}
)
