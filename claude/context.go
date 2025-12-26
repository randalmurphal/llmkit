package claude

import "context"

type contextKey struct{ name string }

var clientContextKey = &contextKey{"claude-client"}

// ContextWithClient adds a Client to a context.
// Use ClientFromContext to retrieve it.
//
// Example:
//
//	client := claude.NewFromConfig(cfg)
//	ctx := claude.ContextWithClient(context.Background(), client)
//	// Pass ctx to functions that need the client
func ContextWithClient(ctx context.Context, c Client) context.Context {
	return context.WithValue(ctx, clientContextKey, c)
}

// ClientFromContext retrieves a Client from a context.
// Returns nil if no Client is present.
//
// Example:
//
//	func processRequest(ctx context.Context, prompt string) (string, error) {
//	    client := claude.ClientFromContext(ctx)
//	    if client == nil {
//	        return "", errors.New("claude client not found in context")
//	    }
//	    resp, err := client.Complete(ctx, claude.CompletionRequest{
//	        Messages: []claude.Message{{Role: claude.RoleUser, Content: prompt}},
//	    })
//	    if err != nil {
//	        return "", err
//	    }
//	    return resp.Content, nil
//	}
func ClientFromContext(ctx context.Context) Client {
	if c, ok := ctx.Value(clientContextKey).(Client); ok {
		return c
	}
	return nil
}

// MustClientFromContext retrieves a Client or panics.
// Use when client is required and missing is a programming error.
//
// Example:
//
//	func handler(ctx context.Context) {
//	    client := claude.MustClientFromContext(ctx)
//	    // Use client - will panic if not present
//	}
func MustClientFromContext(ctx context.Context) Client {
	c := ClientFromContext(ctx)
	if c == nil {
		panic("claude.Client not found in context")
	}
	return c
}
