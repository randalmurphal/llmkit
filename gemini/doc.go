// Package gemini provides a Go wrapper for the Gemini CLI binary.
//
// The Gemini CLI is Google's AI coding assistant available via npm:
//
//	npm install -g @google/gemini-cli
//
// This package implements the provider.Client interface, enabling
// seamless switching between different AI coding CLI tools while
// maintaining a consistent API.
//
// # Basic Usage
//
//	import "github.com/randalmurphal/llmkit/gemini"
//
//	client := gemini.NewGeminiCLI(
//	    gemini.WithModel("gemini-2.5-pro"),
//	    gemini.WithTimeout(5*time.Minute),
//	)
//
//	resp, err := client.Complete(ctx, gemini.CompletionRequest{
//	    SystemPrompt: "You are a helpful assistant.",
//	    Messages: []gemini.Message{
//	        {Role: gemini.RoleUser, Content: "Hello!"},
//	    },
//	})
//
// # Provider Registry Usage
//
//	import (
//	    "github.com/randalmurphal/llmkit/provider"
//	    _ "github.com/randalmurphal/llmkit/gemini" // Register provider
//	)
//
//	client, err := provider.New("gemini", provider.Config{
//	    Provider: "gemini",
//	    Model:    "gemini-2.5-pro",
//	})
//
// # Native Tools
//
// Gemini CLI provides the following native tools:
//   - read_file: Read file contents
//   - write_file: Write content to files
//   - run_shell_command: Execute shell commands
//   - web_fetch: Fetch content from URLs
//   - google_web_search: Search the web
//   - save_memory: Persist information across sessions
//   - write_todos: Manage todo lists
//
// # Context File
//
// Gemini CLI reads project-specific instructions from GEMINI.md files,
// similar to Claude's CLAUDE.md. Place this file in your project root
// to provide context to the model.
//
// # MCP Support
//
// Gemini CLI supports the Model Context Protocol (MCP) for extending
// capabilities with custom tools. Configure MCP servers via:
//   - WithMCPConfig: Path to MCP configuration file
//   - WithMCPServers: Inline server definitions
//
// # Sandbox Modes
//
// Gemini supports different execution environments:
//   - "host": Direct execution on host machine (default)
//   - "docker": Execute commands in Docker container
//   - "remote-execution": Execute on remote infrastructure
//
// Use WithSandbox to configure the desired mode.
//
// # Non-Interactive Mode
//
// For automation and scripting, use WithYolo to auto-approve all actions
// without prompting for confirmation. Use with caution in trusted environments.
package gemini
