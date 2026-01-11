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

	// Stream sends a request and returns a channel of response chunks.
	// The channel is closed when streaming completes (check chunk.Done).
	// Errors during streaming are returned via chunk.Error.
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
