package provider

import (
	"fmt"
	"sort"
	"sync"
)

// Factory creates a new Client from the given configuration.
// Each provider registers its own factory function.
type Factory func(cfg Config) (Client, error)

// registry stores registered provider factories.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a provider factory to the registry.
// Providers should call this in their init() function.
// Panics if a provider with the same name is already registered.
//
// Example:
//
//	func init() {
//	    provider.Register("claude", func(cfg provider.Config) (provider.Client, error) {
//	        return NewClaudeCLI(cfg)
//	    })
//	}
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("provider %q already registered", name))
	}
	registry[name] = factory
}

// New creates a new Client using the named provider.
// Returns ErrUnknownProvider if the provider is not registered.
//
// Example:
//
//	client, err := provider.New("claude", provider.Config{
//	    Model:   "claude-sonnet-4-20250514",
//	    WorkDir: "/path/to/project",
//	})
func New(name string, cfg Config) (Client, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, name)
	}
	return factory(cfg)
}

// MustNew creates a new Client, panicking on error.
// Use only when provider availability is guaranteed (e.g., in tests).
func MustNew(name string, cfg Config) Client {
	client, err := New(name, cfg)
	if err != nil {
		panic(fmt.Sprintf("provider.MustNew(%q): %v", name, err))
	}
	return client
}

// Available returns the names of all registered providers.
// The list is sorted alphabetically for consistent ordering.
func Available() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsRegistered checks if a provider is registered.
func IsRegistered(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	_, ok := registry[name]
	return ok
}

// Unregister removes a provider from the registry.
// This is primarily useful for testing.
func Unregister(name string) {
	registryMu.Lock()
	defer registryMu.Unlock()

	delete(registry, name)
}

// ClearRegistry removes all registered providers.
// This is primarily useful for testing.
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry = make(map[string]Factory)
}
