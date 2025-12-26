package claude

import (
	"sync"
)

// Default client management for applications that want a global client.
// This is optional - applications can manage their own client lifecycle.

var (
	defaultClient     Client
	defaultClientOnce sync.Once
	defaultClientMu   sync.RWMutex
	defaultConfig     = DefaultConfig()
)

// SetDefaultConfig sets the configuration for the default client.
// Must be called before GetDefaultClient is first called.
// Not thread-safe with concurrent GetDefaultClient calls during initialization.
//
// Example:
//
//	// In main.go or init
//	claude.SetDefaultConfig(claude.Config{
//	    Model:    "claude-opus-4-5-20251101",
//	    MaxTurns: 10,
//	})
//
//	// Anywhere else in the application
//	client := claude.GetDefaultClient()
func SetDefaultConfig(cfg Config) {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultConfig = cfg
}

// GetDefaultClient returns a singleton default client.
// Creates the client lazily on first call using the default config.
// Thread-safe for concurrent access after initialization.
//
// Example:
//
//	client := claude.GetDefaultClient()
//	resp, err := client.Complete(ctx, req)
func GetDefaultClient() Client {
	// Fast path: already initialized
	defaultClientMu.RLock()
	if defaultClient != nil {
		c := defaultClient
		defaultClientMu.RUnlock()
		return c
	}
	defaultClientMu.RUnlock()

	// Slow path: initialize using Once to ensure single initialization
	// Capture config under lock to avoid race with SetDefaultConfig
	defaultClientMu.Lock()
	cfg := defaultConfig
	defaultClientMu.Unlock()

	defaultClientOnce.Do(func() {
		client := NewFromConfig(cfg)
		defaultClientMu.Lock()
		defaultClient = client
		defaultClientMu.Unlock()
	})

	defaultClientMu.RLock()
	c := defaultClient
	defaultClientMu.RUnlock()
	return c
}

// SetDefaultClient sets the singleton client directly.
// Useful for testing or when you want to manage the client lifecycle.
// Note: If GetDefaultClient was already called, call ResetDefaultClient first.
//
// Example:
//
//	// In tests
//	mock := claude.NewMockClient("test response")
//	claude.SetDefaultClient(mock)
//	defer claude.ResetDefaultClient()
func SetDefaultClient(c Client) {
	if c == nil {
		panic("SetDefaultClient: client cannot be nil (use ResetDefaultClient instead)")
	}
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient = c
	// Mark as initialized so GetDefaultClient won't recreate
	defaultClientOnce.Do(func() {})
}

// ResetDefaultClient clears the singleton client.
// Useful for testing to ensure clean state between tests.
// After reset, GetDefaultClient will create a new client.
//
// Example:
//
//	func TestSomething(t *testing.T) {
//	    mock := claude.NewMockClient("response")
//	    claude.SetDefaultClient(mock)
//	    defer claude.ResetDefaultClient()
//
//	    // Test code that uses GetDefaultClient()
//	}
func ResetDefaultClient() {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient = nil
	defaultClientOnce = sync.Once{}
}
