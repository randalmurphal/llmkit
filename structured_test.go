package llmkit

import (
	"context"
	"encoding/json"
	"testing"
)

type typedMockClient struct {
	resp *Response
	err  error
	req  Request
}

func (m *typedMockClient) Complete(_ context.Context, req Request) (*Response, error) {
	m.req = req
	return m.resp, m.err
}
func (m *typedMockClient) Stream(context.Context, Request) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}
func (m *typedMockClient) Provider() string           { return "mock" }
func (m *typedMockClient) Capabilities() Capabilities { return Capabilities{} }
func (m *typedMockClient) Close() error               { return nil }

func TestCompleteTyped(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	mock := &typedMockClient{
		resp: &Response{Content: `{"name":"demo"}`},
	}
	out, err := CompleteTyped[payload](context.Background(), mock, Request{})
	if err != nil {
		t.Fatalf("CompleteTyped: %v", err)
	}
	if out.Value.Name != "demo" {
		t.Fatalf("Value.Name = %q", out.Value.Name)
	}
	if len(mock.req.JSONSchema) == 0 {
		t.Fatal("expected generated schema")
	}
	var schema map[string]any
	if err := json.Unmarshal(mock.req.JSONSchema, &schema); err != nil {
		t.Fatalf("schema unmarshal: %v", err)
	}
}

func TestCompleteTypedRejectsEmptyContent(t *testing.T) {
	_, err := CompleteTyped[map[string]any](context.Background(), &typedMockClient{
		resp: &Response{Content: ""},
	}, Request{})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCompleteTypedExtractsLastJSONValue(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	out, err := CompleteTyped[payload](context.Background(), &typedMockClient{
		resp: &Response{Content: "Thinking...\n{\"name\":\"demo\"}"},
	}, Request{})
	if err != nil {
		t.Fatalf("CompleteTyped: %v", err)
	}
	if out.Value.Name != "demo" {
		t.Fatalf("Value.Name = %q", out.Value.Name)
	}
}

func TestCompleteTypedRejectsUnknownFields(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	_, err := CompleteTyped[payload](context.Background(), &typedMockClient{
		resp: &Response{Content: `{"name":"demo","extra":"nope"}`},
	}, Request{})
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}
