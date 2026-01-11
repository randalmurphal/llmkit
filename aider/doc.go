// Package aider provides a client for the Aider CLI.
//
// Aider is a git-aware AI coding assistant that supports local models via Ollama.
// This package wraps the `aider` CLI for programmatic use, particularly suited
// for code editing workflows that integrate with git.
//
// # Installation
//
// Install Aider:
//
//	pip install aider-chat
//
// # Usage
//
// Via provider registry:
//
//	import _ "github.com/randalmurphal/llmkit/aider" // Register provider
//
//	client, err := provider.New("aider", provider.Config{
//	    Provider: "aider",
//	    Model:    "ollama_chat/llama3.2:latest",
//	    WorkDir:  "/path/to/project",
//	    Options: map[string]any{
//	        "editable_files": []string{"src/main.go"},
//	        "yes_always":     true,
//	    },
//	})
//
// Direct instantiation:
//
//	client := aider.NewAiderCLI(
//	    aider.WithModel("ollama_chat/llama3.2:latest"),
//	    aider.WithWorkdir("/path/to/project"),
//	    aider.WithEditableFiles([]string{"src/main.go"}),
//	    aider.WithYesAlways(),
//	)
//
// # Ollama Configuration
//
// Use the ollama_chat/ prefix for Ollama models:
//
//	client := aider.NewAiderCLI(
//	    aider.WithModel("ollama_chat/llama3.2:latest"),
//	    aider.WithOllamaAPIBase("http://localhost:11434"),
//	)
//
// Note: Ollama defaults to 2k context window which is small for Aider.
// Set OLLAMA_CONTEXT_LENGTH=8192 (or higher) when running ollama serve.
//
// # Capabilities
//
// Aider provides git-centric code editing:
//   - File editing with git integration
//   - Shell command suggestions
//   - Automatic commits (can be disabled)
//
// Aider does NOT currently support:
//   - MCP servers (PR pending)
//   - Session persistence
//   - Image inputs
//   - JSON output format (text parsing required)
//
// # Git Integration
//
// Aider is deeply integrated with git. Control git behavior with:
//
//	client := aider.NewAiderCLI(
//	    aider.WithNoGit(),        // Disable git entirely
//	    aider.WithNoAutoCommits(), // Disable auto-commits
//	)
package aider
