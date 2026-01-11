// Package continue provides a client for Continue.dev CLI (cn).
//
// Continue.dev is an open-source coding agent that supports local models via Ollama,
// MCP servers, and agentic file/shell operations. This package wraps the `cn` CLI
// for programmatic use.
//
// # Installation
//
// Install the Continue CLI:
//
//	npm i -g @continuedev/cli
//
// # Usage
//
// Via provider registry:
//
//	import _ "github.com/randalmurphal/llmkit/continue" // Register provider
//
//	client, err := provider.New("continue", provider.Config{
//	    Provider: "continue",
//	    Model:    "llama3.2:latest",
//	    WorkDir:  "/path/to/project",
//	    Options: map[string]any{
//	        "config_path": "~/.continue/config.yaml",
//	    },
//	})
//
// Direct instantiation:
//
//	client := continue.NewContinueCLI(
//	    continue.WithModel("llama3.2:latest"),
//	    continue.WithConfigPath("~/.continue/config.yaml"),
//	    continue.WithWorkdir("/path/to/project"),
//	)
//
// # Configuration
//
// Continue uses config.yaml for model and MCP server configuration.
// See https://docs.continue.dev/reference for full schema.
//
// Example config.yaml:
//
//	name: my-config
//	version: 1.0.0
//	schema: v1
//	models:
//	  - name: Ollama Llama3
//	    provider: ollama
//	    model: llama3.2:latest
//	    apiBase: http://localhost:11434
//	    roles: [chat, edit, apply]
//	mcpServers:
//	  - name: filesystem
//	    command: npx
//	    args: ["@modelcontextprotocol/server-filesystem", "/workspace"]
//
// # Capabilities
//
// Continue provides full agentic capabilities:
//   - File read/write/edit
//   - Shell/bash execution
//   - MCP server integration
//   - Session resumption
//   - Tool permission control
//
// # Tool Permissions
//
// Control tool access with --allow, --ask, and --exclude flags:
//
//	client := continue.NewContinueCLI(
//	    continue.WithAllowedTools([]string{"Write()", "Edit()"}),
//	    continue.WithAskTools([]string{"Bash(curl*)"}),
//	    continue.WithExcludedTools([]string{"Fetch"}),
//	)
package continuedev
