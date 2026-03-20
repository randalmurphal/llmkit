package provider

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// observableMockClient implements Client for testing ObservableClient.
type observableMockClient struct {
	providerName string
	capabilities Capabilities

	completeResp *Response
	completeErr  error

	streamChunks []StreamChunk
	streamErr    error

	closeErr error
}

func (m *observableMockClient) Complete(_ context.Context, _ Request) (*Response, error) {
	if m.completeErr != nil {
		return nil, m.completeErr
	}
	return m.completeResp, nil
}

func (m *observableMockClient) Stream(_ context.Context, _ Request) (<-chan StreamChunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		for _, chunk := range m.streamChunks {
			ch <- chunk
		}
	}()
	return ch, nil
}

func (m *observableMockClient) Provider() string        { return m.providerName }
func (m *observableMockClient) Capabilities() Capabilities { return m.capabilities }
func (m *observableMockClient) Close() error             { return m.closeErr }

// eventCollector collects events in a thread-safe list.
type eventCollector struct {
	mu     sync.Mutex
	events []Event
}

func (c *eventCollector) handler(e Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *eventCollector) list() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]Event, len(c.events))
	copy(result, c.events)
	return result
}

func TestNewObservableClient(t *testing.T) {
	inner := &observableMockClient{providerName: "test"}
	collector := &eventCollector{}

	oc := NewObservableClient(inner, collector.handler)
	if oc == nil {
		t.Fatal("expected non-nil ObservableClient")
	}
}

func TestObservableClient_Complete_Success(t *testing.T) {
	inner := &observableMockClient{
		providerName: "test-provider",
		completeResp: &Response{
			Content:   "Hello, world!",
			Model:     "test-model",
			SessionID: "sess-1",
			Usage:     TokenUsage{InputTokens: 10, OutputTokens: 5},
		},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	resp, err := oc.Complete(context.Background(), Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got %q", resp.Content)
	}

	events := collector.list()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Type != EventDone {
		t.Errorf("expected EventDone, got %v", e.Type)
	}
	if e.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %q", e.Text)
	}
	if e.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", e.Model)
	}
	if e.SessionID != "sess-1" {
		t.Errorf("expected session ID 'sess-1', got %q", e.SessionID)
	}
	if !e.Done {
		t.Error("expected Done to be true")
	}
	if e.Usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if e.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", e.Usage.InputTokens)
	}
	if e.Provider != "test-provider" {
		t.Errorf("expected provider 'test-provider', got %q", e.Provider)
	}
	if e.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestObservableClient_Complete_Error(t *testing.T) {
	testErr := errors.New("api error")
	inner := &observableMockClient{
		providerName: "test-provider",
		completeErr:  testErr,
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	resp, err := oc.Complete(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected error")
	}
	if resp != nil {
		t.Error("expected nil response on error")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("expected error %v, got %v", testErr, err)
	}

	events := collector.list()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Type != EventError {
		t.Errorf("expected EventError, got %v", e.Type)
	}
	if !errors.Is(e.Error, testErr) {
		t.Errorf("expected error %v in event, got %v", testErr, e.Error)
	}
}

func TestObservableClient_Stream_Success(t *testing.T) {
	usage := &TokenUsage{InputTokens: 20, OutputTokens: 10}
	inner := &observableMockClient{
		providerName: "test-provider",
		streamChunks: []StreamChunk{
			{Content: "Hello", SessionID: "sess-1"},
			{Content: " World", SessionID: "sess-1"},
			{Done: true, SessionID: "sess-1", Usage: usage},
		},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	ch, err := oc.Stream(context.Background(), Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drain the channel.
	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[0].Content != "Hello" {
		t.Errorf("expected first chunk content 'Hello', got %q", chunks[0].Content)
	}
	if chunks[1].Content != " World" {
		t.Errorf("expected second chunk content ' World', got %q", chunks[1].Content)
	}
	if !chunks[2].Done {
		t.Error("expected third chunk to be done")
	}

	// Check emitted events.
	events := collector.list()

	// Should have: EventText("Hello"), EventText(" World"), EventUsage, EventDone
	var textEvents, usageEvents, doneEvents int
	for _, e := range events {
		switch e.Type {
		case EventText:
			textEvents++
		case EventUsage:
			usageEvents++
		case EventDone:
			doneEvents++
		}
	}

	if textEvents != 2 {
		t.Errorf("expected 2 EventText, got %d", textEvents)
	}
	if usageEvents != 1 {
		t.Errorf("expected 1 EventUsage, got %d", usageEvents)
	}
	if doneEvents != 1 {
		t.Errorf("expected 1 EventDone, got %d", doneEvents)
	}
}

func TestObservableClient_Stream_EmitsTextForContentChunks(t *testing.T) {
	inner := &observableMockClient{
		providerName: "test-provider",
		streamChunks: []StreamChunk{
			{Content: "chunk1", SessionID: "s"},
		},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	ch, err := oc.Stream(context.Background(), Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for range ch {
	}

	events := collector.list()
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	if events[0].Type != EventText {
		t.Errorf("expected EventText, got %v", events[0].Type)
	}
	if events[0].Text != "chunk1" {
		t.Errorf("expected text 'chunk1', got %q", events[0].Text)
	}
	if events[0].SessionID != "s" {
		t.Errorf("expected session ID 's', got %q", events[0].SessionID)
	}
}

func TestObservableClient_Stream_EmitsErrorForErrorChunks(t *testing.T) {
	testErr := errors.New("stream broken")
	inner := &observableMockClient{
		providerName: "test-provider",
		streamChunks: []StreamChunk{
			{Error: testErr, SessionID: "s"},
		},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	ch, err := oc.Stream(context.Background(), Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for range ch {
	}

	events := collector.list()
	var errorEvents int
	for _, e := range events {
		if e.Type == EventError {
			errorEvents++
			if !errors.Is(e.Error, testErr) {
				t.Errorf("expected error %v, got %v", testErr, e.Error)
			}
		}
	}
	if errorEvents != 1 {
		t.Errorf("expected 1 EventError, got %d", errorEvents)
	}
}

func TestObservableClient_Stream_EmitsDoneForDoneChunks(t *testing.T) {
	inner := &observableMockClient{
		providerName: "test-provider",
		streamChunks: []StreamChunk{
			{Done: true, SessionID: "s"},
		},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	ch, err := oc.Stream(context.Background(), Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for range ch {
	}

	events := collector.list()
	var doneEvents int
	for _, e := range events {
		if e.Type == EventDone {
			doneEvents++
			if !e.Done {
				t.Error("expected Done flag to be true")
			}
		}
	}
	if doneEvents != 1 {
		t.Errorf("expected 1 EventDone, got %d", doneEvents)
	}
}

func TestObservableClient_Stream_EmitsToolCallEvents(t *testing.T) {
	tc := ToolCall{ID: "tc-1", Name: "Read", Arguments: []byte(`{"path":"/test"}`)}
	inner := &observableMockClient{
		providerName: "test-provider",
		streamChunks: []StreamChunk{
			{ToolCalls: []ToolCall{tc}, SessionID: "s"},
		},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	ch, err := oc.Stream(context.Background(), Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for range ch {
	}

	events := collector.list()
	var toolEvents int
	for _, e := range events {
		if e.Type == EventToolCall {
			toolEvents++
			if e.ToolCall == nil {
				t.Error("expected non-nil ToolCall in event")
			} else if e.ToolCall.Name != "Read" {
				t.Errorf("expected tool name 'Read', got %q", e.ToolCall.Name)
			}
		}
	}
	if toolEvents != 1 {
		t.Errorf("expected 1 EventToolCall, got %d", toolEvents)
	}
}

func TestObservableClient_Stream_InitError(t *testing.T) {
	testErr := errors.New("init failed")
	inner := &observableMockClient{
		providerName: "test-provider",
		streamErr:    testErr,
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	ch, err := oc.Stream(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected error")
	}
	if ch != nil {
		t.Error("expected nil channel on error")
	}

	events := collector.list()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventError {
		t.Errorf("expected EventError, got %v", events[0].Type)
	}
}

func TestObservableClient_Stream_ClosesOuterChannelWhenInnerCloses(t *testing.T) {
	inner := &observableMockClient{
		providerName: "test-provider",
		streamChunks: []StreamChunk{
			{Content: "only"},
		},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	ch, err := oc.Stream(context.Background(), Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drain all chunks.
	count := 0
	for range ch {
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 chunk, got %d", count)
	}
	// Channel is now closed - verify by trying to read again.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}
}

func TestObservableClient_Stream_ContextCancellation(t *testing.T) {
	// Create a stream that blocks forever.
	inner := &observableMockClient{
		providerName: "test-provider",
	}
	// Override Stream to send chunks slowly.
	blockingInner := &blockingStreamClient{providerName: "test-provider"}
	collector := &eventCollector{}
	oc := NewObservableClient(blockingInner, collector.handler)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := oc.Stream(ctx, Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = inner // suppress unused warning for the replaced mock

	// Cancel context, which should cause the goroutine to exit.
	cancel()

	// Drain whatever is left - the channel should close.
	for range ch {
	}
}

// blockingStreamClient sends one chunk then blocks until context cancels.
type blockingStreamClient struct {
	providerName string
}

func (b *blockingStreamClient) Complete(_ context.Context, _ Request) (*Response, error) {
	return nil, errors.New("not implemented")
}

func (b *blockingStreamClient) Stream(_ context.Context, _ Request) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	go func() {
		// Send one chunk, then keep the channel open forever.
		// The ObservableClient's goroutine should exit when context cancels.
		ch <- StreamChunk{Content: "start"}
		// Never close - the test relies on context cancellation.
	}()
	return ch, nil
}

func (b *blockingStreamClient) Provider() string        { return b.providerName }
func (b *blockingStreamClient) Capabilities() Capabilities { return Capabilities{} }
func (b *blockingStreamClient) Close() error             { return nil }

func TestObservableClient_Provider(t *testing.T) {
	inner := &observableMockClient{providerName: "my-provider"}
	oc := NewObservableClient(inner, func(Event) {})

	if oc.Provider() != "my-provider" {
		t.Errorf("expected 'my-provider', got %q", oc.Provider())
	}
}

func TestObservableClient_Capabilities(t *testing.T) {
	caps := Capabilities{
		Streaming:   true,
		Tools:       true,
		NativeTools: []string{"Read", "Write"},
	}
	inner := &observableMockClient{capabilities: caps}
	oc := NewObservableClient(inner, func(Event) {})

	got := oc.Capabilities()
	if !got.Streaming {
		t.Error("expected Streaming to be true")
	}
	if !got.Tools {
		t.Error("expected Tools to be true")
	}
	if len(got.NativeTools) != 2 {
		t.Errorf("expected 2 native tools, got %d", len(got.NativeTools))
	}
}

func TestObservableClient_Close(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inner := &observableMockClient{}
		oc := NewObservableClient(inner, func(Event) {})
		if err := oc.Close(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		closeErr := errors.New("close failed")
		inner := &observableMockClient{closeErr: closeErr}
		oc := NewObservableClient(inner, func(Event) {})
		err := oc.Close()
		if !errors.Is(err, closeErr) {
			t.Errorf("expected error %v, got %v", closeErr, err)
		}
	})
}

func TestObservableClient_Emit_FillsProviderWhenEmpty(t *testing.T) {
	inner := &observableMockClient{
		providerName: "auto-fill-provider",
		completeResp: &Response{Content: "ok"},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	_, _ = oc.Complete(context.Background(), Request{})

	events := collector.list()
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	if events[0].Provider != "auto-fill-provider" {
		t.Errorf("expected provider 'auto-fill-provider', got %q", events[0].Provider)
	}
}

func TestObservableClient_Emit_FillsTimestamp(t *testing.T) {
	inner := &observableMockClient{
		providerName: "test",
		completeResp: &Response{Content: "ok"},
	}
	collector := &eventCollector{}
	oc := NewObservableClient(inner, collector.handler)

	_, _ = oc.Complete(context.Background(), Request{})

	events := collector.list()
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	if events[0].Timestamp.IsZero() {
		t.Error("expected non-zero timestamp to be auto-filled")
	}
}
