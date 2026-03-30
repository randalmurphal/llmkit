package llmkit

import (
	"context"
	"testing"
)

type mockClient struct {
	name string
}

func (m *mockClient) Complete(context.Context, Request) (*Response, error) {
	return &Response{Content: "ok"}, nil
}

func (m *mockClient) Stream(context.Context, Request) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	ch <- StreamChunk{Content: "chunk", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockClient) Provider() string           { return m.name }
func (m *mockClient) Capabilities() Capabilities { return Capabilities{} }
func (m *mockClient) Close() error               { return nil }

func TestRegistryLifecycle(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	Register("test", func(cfg Config) (Client, error) {
		return &mockClient{name: cfg.Provider}, nil
	})
	if !IsRegistered("test") {
		t.Fatal("expected provider to be registered")
	}

	client, err := New("test", Config{Provider: "test"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if client.Provider() != "test" {
		t.Fatalf("Provider = %q", client.Provider())
	}

	available := Available()
	if len(available) != 1 || available[0] != "test" {
		t.Fatalf("Available = %v", available)
	}

	Unregister("test")
	if IsRegistered("test") {
		t.Fatal("expected provider to be unregistered")
	}
}
