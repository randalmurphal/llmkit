// Package claude provides a Go wrapper for the Claude CLI.
//
// This package enables programmatic interaction with Claude through the
// official CLI binary. It supports both synchronous completions and
// streaming responses.
//
// # Basic Usage
//
//	client := claude.NewCLI()
//	resp, err := client.Complete(ctx, claude.CompletionRequest{
//	    Messages: []claude.Message{
//	        {Role: claude.RoleUser, Content: "Hello!"},
//	    },
//	})
//
// # Streaming
//
//	stream, err := client.Stream(ctx, req)
//	for chunk := range stream {
//	    fmt.Print(chunk.Content)
//	}
//
// # Container Support
//
// For containerized environments, credentials can be loaded from a mounted
// directory:
//
//	client := claude.NewCLI(
//	    claude.WithHomeDir("/home/worker"),
//	    claude.WithDangerouslySkipPermissions(),
//	)
//
// # Testing
//
// Use MockClient for testing without the actual CLI:
//
//	mock := &claude.MockClient{
//	    CompleteFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
//	        return &CompletionResponse{Content: "test"}, nil
//	    },
//	}
package claude
