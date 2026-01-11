package provider

import (
	"context"
	"testing"
)

// mockClient implements Client for testing.
type mockClient struct {
	name string
}

func (m *mockClient) Complete(ctx context.Context, req Request) (*Response, error) {
	return &Response{Content: "mock response"}, nil
}

func (m *mockClient) Stream(ctx context.Context, req Request) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	ch <- StreamChunk{Content: "mock chunk", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockClient) Provider() string { return m.name }

func (m *mockClient) Capabilities() Capabilities {
	return Capabilities{
		Streaming:   true,
		Tools:       true,
		NativeTools: []string{"TestTool"},
	}
}

func (m *mockClient) Close() error { return nil }

func TestRegister(t *testing.T) {
	// Clear registry for clean test
	ClearRegistry()
	defer ClearRegistry()

	// Register a test provider
	Register("test", func(cfg Config) (Client, error) {
		return &mockClient{name: "test"}, nil
	})

	// Verify it's registered
	if !IsRegistered("test") {
		t.Error("expected 'test' to be registered")
	}
}

func TestRegister_Panic(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	// Register once
	Register("duplicate", func(cfg Config) (Client, error) {
		return &mockClient{name: "duplicate"}, nil
	})

	// Second registration should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	Register("duplicate", func(cfg Config) (Client, error) {
		return &mockClient{name: "duplicate2"}, nil
	})
}

func TestNew(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	Register("test", func(cfg Config) (Client, error) {
		return &mockClient{name: "test"}, nil
	})

	client, err := New("test", Config{Provider: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Provider() != "test" {
		t.Errorf("expected provider 'test', got %q", client.Provider())
	}
}

func TestNew_UnknownProvider(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	_, err := New("unknown", Config{Provider: "unknown"})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestMustNew(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	Register("test", func(cfg Config) (Client, error) {
		return &mockClient{name: "test"}, nil
	})

	client := MustNew("test", Config{Provider: "test"})
	if client.Provider() != "test" {
		t.Errorf("expected provider 'test', got %q", client.Provider())
	}
}

func TestMustNew_Panics(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown provider")
		}
	}()
	MustNew("unknown", Config{Provider: "unknown"})
}

func TestAvailable(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	Register("alpha", func(cfg Config) (Client, error) {
		return &mockClient{name: "alpha"}, nil
	})
	Register("beta", func(cfg Config) (Client, error) {
		return &mockClient{name: "beta"}, nil
	})

	available := Available()
	if len(available) != 2 {
		t.Errorf("expected 2 providers, got %d", len(available))
	}
	// Should be sorted
	if available[0] != "alpha" || available[1] != "beta" {
		t.Errorf("expected [alpha, beta], got %v", available)
	}
}

func TestUnregister(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	Register("test", func(cfg Config) (Client, error) {
		return &mockClient{name: "test"}, nil
	})

	if !IsRegistered("test") {
		t.Error("expected 'test' to be registered")
	}

	Unregister("test")

	if IsRegistered("test") {
		t.Error("expected 'test' to be unregistered")
	}
}
