// Package codex provides a Go wrapper for the OpenAI Codex CLI.
//
// The wrapper is designed for headless/non-interactive execution using
// `codex exec --json` and includes adaptive parsing for modern JSONL events
// like thread/turn/item lifecycle messages.
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
//	    Messages: []codex.Message{{Role: codex.RoleUser, Content: "Summarize this repo"}},
//	})
//
// # Headless Configuration
//
//	client := codex.NewCodexCLI(
//	    codex.WithProfile("ci"),
//	    codex.WithSandboxMode(codex.SandboxWorkspaceWrite),
//	    codex.WithApprovalMode(codex.ApprovalNever),
//	    codex.WithWebSearchMode(codex.WebSearchCached),
//	    codex.WithConfigOverride("model_reasoning_effort", "medium"),
//	    codex.WithEnabledFeatures([]string{"project_doc"}),
//	)
//
// # Session Resume
//
// Resume a specific session or the most recent session via WithSessionID:
//
//	client := codex.NewCodexCLI(codex.WithSessionID("thread_abc123"))
//	resp, err := client.Complete(ctx, codex.CompletionRequest{Messages: ...})
//
// The session ID is passed through buildExecArgs which emits `exec resume <id>`.
// Flags unsupported by `exec resume` are automatically filtered out.
//
// # Unified llmkit Interface
//
// The codex package registers itself with the llmkit registry:
//
//	import (
//	    "github.com/randalmurphal/llmkit/v2"
//	    _ "github.com/randalmurphal/llmkit/v2/codex"
//	)
//
//	client, err := llmkit.New("codex", llmkit.Config{
//	    Provider: "codex",
//	    Model: "gpt-5-codex",
//	})
package codex
