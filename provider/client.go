// Package provider defines the unified interface for LLM CLI providers.
//
// This package enables seamless switching between different AI coding CLI tools
// (Claude Code, Gemini CLI, Codex CLI, OpenCode, local models) while maintaining
// a consistent API. All providers support MCP (Model Context Protocol) as the
// universal extension mechanism for custom tools.
//
// # Usage
//
// Create a client using the registry:
//
//	client, err := provider.New("claude", provider.Config{
//	    Model:   "claude-sonnet-4-20250514",
//	    WorkDir: "/path/to/project",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Check capabilities before using provider-specific features
//	caps := client.Capabilities()
//	if caps.HasTool("Glob") {
//	    // Use native glob search
//	} else if caps.MCP {
//	    // Fall back to MCP filesystem server
//	}
//
// # Available Providers
//
//   - "claude": Claude Code CLI (Anthropic)
//   - "gemini": Gemini CLI (Google)
//   - "codex": Codex CLI (OpenAI)
//   - "opencode": OpenCode CLI (Open Source)
//   - "local": Local model sidecar (Ollama, llama.cpp, vLLM)
//
// # Tool Support
//
// Each provider has different native tools. Use Capabilities() to discover what's
// available, and MCP for cross-provider custom tools. See the Capabilities struct
// documentation for details on each provider's native capabilities.
package provider

import "context"

// Client is the unified interface for LLM CLI providers.
// Implementations must be safe for concurrent use.
type Client interface {
	// Complete sends a request and returns the full response.
	// The context controls cancellation and timeouts.
	Complete(ctx context.Context, req Request) (*Response, error)

	// Stream sends a request and returns a channel of response chunks.
	// The channel is closed when streaming completes (check chunk.Done).
	// Errors during streaming are returned via chunk.Error.
	Stream(ctx context.Context, req Request) (<-chan StreamChunk, error)

	// Provider returns the provider name (e.g., "claude", "gemini", "codex").
	Provider() string

	// Capabilities returns what this provider natively supports.
	// Use this to check for specific tools before attempting to use them.
	Capabilities() Capabilities

	// Close releases any resources held by the client.
	// For CLI-based providers, this may terminate any running sessions.
	Close() error
}

// Capabilities describes what a provider natively supports.
// Use this to make informed decisions about feature availability
// before attempting operations that may not be supported.
type Capabilities struct {
	// Streaming indicates if the provider supports streaming responses.
	Streaming bool `json:"streaming"`

	// Tools indicates if the provider supports tool/function calling.
	Tools bool `json:"tools"`

	// MCP indicates if the provider supports MCP (Model Context Protocol) servers.
	// If true, custom tools can be added via MCP regardless of native tool support.
	MCP bool `json:"mcp"`

	// Sessions indicates if the provider supports multi-turn conversation sessions.
	Sessions bool `json:"sessions"`

	// Images indicates if the provider supports image inputs.
	Images bool `json:"images"`

	// NativeTools lists the provider's built-in tools by name.
	// Tool names are provider-specific (e.g., "Read" for Claude, "read_file" for Gemini).
	NativeTools []string `json:"native_tools"`

	// ContextFile is the filename for project-specific context (e.g., "CLAUDE.md", "GEMINI.md").
	// Empty string if the provider doesn't support context files.
	ContextFile string `json:"context_file,omitempty"`
}

// HasTool checks if a native tool is available by name.
// Tool names are case-sensitive and provider-specific.
func (c Capabilities) HasTool(name string) bool {
	for _, t := range c.NativeTools {
		if t == name {
			return true
		}
	}
	return false
}

// Pre-defined capability sets for known providers.
// These are exported for documentation purposes; actual capabilities
// are returned by each provider's Capabilities() method.
var (
	// ClaudeCapabilities describes Claude Code CLI's native capabilities.
	ClaudeCapabilities = Capabilities{
		Streaming:   true,
		Tools:       true,
		MCP:         true,
		Sessions:    true,
		Images:      true,
		NativeTools: []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash", "Task", "TodoWrite", "WebFetch", "WebSearch", "AskUserQuestion", "NotebookEdit", "LSP", "Skill", "EnterPlanMode", "ExitPlanMode", "KillShell", "TaskOutput"},
		ContextFile: "CLAUDE.md",
	}

	// GeminiCapabilities describes Gemini CLI's native capabilities.
	GeminiCapabilities = Capabilities{
		Streaming:   true,
		Tools:       true,
		MCP:         true,
		Sessions:    false,
		Images:      true,
		NativeTools: []string{"read_file", "write_file", "run_shell_command", "web_fetch", "google_web_search", "save_memory", "write_todos"},
		ContextFile: "GEMINI.md",
	}

	// CodexCapabilities describes Codex CLI's native capabilities.
	CodexCapabilities = Capabilities{
		Streaming:   true,
		Tools:       true,
		MCP:         true,
		Sessions:    true,
		Images:      true,
		NativeTools: []string{"file_read", "file_write", "shell", "web_search"},
		ContextFile: "",
	}

	// OpenCodeCapabilities describes OpenCode CLI's native capabilities.
	OpenCodeCapabilities = Capabilities{
		Streaming:   true,
		Tools:       true,
		MCP:         true,
		Sessions:    false,
		Images:      false,
		NativeTools: []string{"write", "edit", "bash", "WebFetch", "Task"},
		ContextFile: "",
	}

	// LocalCapabilities describes local model sidecar capabilities.
	// Local models have no native tools; all tools must be provided via MCP.
	LocalCapabilities = Capabilities{
		Streaming:   true,
		Tools:       false,
		MCP:         true,
		Sessions:    false,
		Images:      false,
		NativeTools: nil,
		ContextFile: "",
	}
)
