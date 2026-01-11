// Package codex provides a Go wrapper for the OpenAI Codex CLI.
//
// The Codex CLI is an AI coding assistant that provides file operations,
// shell execution, and web search capabilities. This package wraps the CLI
// to provide a programmatic Go interface.
//
// # Installation
//
// The Codex CLI must be installed separately:
//
//	npm install -g @openai/codex
//
// # Basic Usage
//
//	client := codex.NewCodexCLI(
//	    codex.WithModel("gpt-5-codex"),
//	    codex.WithTimeout(5*time.Minute),
//	)
//
//	resp, err := client.Complete(ctx, codex.CompletionRequest{
//	    Messages: []codex.Message{
//	        {Role: codex.RoleUser, Content: "What files are in this directory?"},
//	    },
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # Sandbox Modes
//
// Codex supports three sandbox modes that control file system access:
//
//   - SandboxReadOnly: Only read operations allowed
//   - SandboxWorkspaceWrite: Can write to workspace directory (default)
//   - SandboxDangerFullAccess: Full file system access
//
// Example:
//
//	client := codex.NewCodexCLI(
//	    codex.WithSandboxMode(codex.SandboxReadOnly),
//	)
//
// # Approval Modes
//
// Control when Codex asks for user approval:
//
//   - ApprovalUntrusted: Ask for all operations (most restrictive)
//   - ApprovalOnFailure: Ask only when operations fail
//   - ApprovalOnRequest: Ask only when explicitly requested
//   - ApprovalNever: Never ask for approval (least restrictive)
//
// For non-interactive usage, use WithFullAuto():
//
//	client := codex.NewCodexCLI(
//	    codex.WithFullAuto(),
//	)
//
// # Session Resume
//
// Resume a previous session:
//
//	client := codex.NewCodexCLI()
//	resp, err := client.Resume(ctx, "session-id", "Continue with the task")
//
// # Image Attachments
//
// Attach images to requests:
//
//	client := codex.NewCodexCLI(
//	    codex.WithImages([]string{"/path/to/screenshot.png"}),
//	)
//
// # Provider Interface
//
// The codex package registers itself with the provider registry:
//
//	import (
//	    "github.com/randalmurphal/llmkit/provider"
//	    _ "github.com/randalmurphal/llmkit/codex" // Register provider
//	)
//
//	client, err := provider.New("codex", provider.Config{
//	    Model:   "gpt-5-codex",
//	    WorkDir: "/path/to/project",
//	    Options: map[string]any{
//	        "sandbox":          "workspace-write",
//	        "ask_for_approval": "never",
//	    },
//	})
package codex
