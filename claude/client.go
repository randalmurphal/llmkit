// Package claude provides interfaces and implementations for the Claude Code CLI.
//
// This package can be used directly via NewClaudeCLI, or through the unified
// provider interface via provider.New("claude", cfg).
//
// # Direct Usage
//
//	client := claude.NewClaudeCLI(
//	    claude.WithModel("claude-sonnet-4-20250514"),
//	    claude.WithWorkdir("/path/to/project"),
//	)
//	resp, err := client.Complete(ctx, claude.CompletionRequest{
//	    Messages: []claude.Message{{Role: claude.RoleUser, Content: "Hello"}},
//	})
//
// # Provider Interface Usage
//
//	import _ "github.com/randalmurphal/llmkit/claude" // Register provider
//
//	client, err := provider.New("claude", provider.Config{
//	    Provider: "claude",
//	    Model:    "claude-sonnet-4-20250514",
//	})
//	resp, err := client.Complete(ctx, provider.Request{
//	    Messages: []provider.Message{{Role: provider.RoleUser, Content: "Hello"}},
//	})
package claude

import "context"

// Client is the interface for LLM providers.
// Implementations must be safe for concurrent use.
type Client interface {
	// Complete sends a request and returns the full response.
	// The context controls cancellation and timeouts.
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// StreamJSON sends a request and returns streaming events plus a result future.
	// Events include init (session_id), assistant (per-message content/usage), result (final totals).
	// The channel closes when streaming completes. Use result.Wait() for the final ResultEvent.
	StreamJSON(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, *StreamResult, error)
}
