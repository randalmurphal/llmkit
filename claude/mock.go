package claude

import (
	"context"
	"sync"
	"time"
)

// MockClient is a test double for Client.
// It supports fixed responses, sequential responses, and custom handlers.
type MockClient struct {
	mu             sync.Mutex
	responses      []string
	responseIdx    int
	err            error
	completeFunc   func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	streamJSONFunc func(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, *StreamResult, error)

	// Calls tracks all requests for assertions.
	Calls []CompletionRequest
}

// NewMockClient creates a mock that returns a fixed response.
func NewMockClient(response string) *MockClient {
	return &MockClient{responses: []string{response}}
}

// WithResponses configures sequential responses.
// Each call to Complete returns the next response in the list.
// Cycles back to the beginning after exhausting all responses.
func (m *MockClient) WithResponses(responses ...string) *MockClient {
	m.responses = responses
	return m
}

// WithError configures the mock to always return an error.
func (m *MockClient) WithError(err error) *MockClient {
	m.err = err
	return m
}

// WithCompleteFunc sets a custom handler for Complete calls.
// This takes precedence over fixed responses.
func (m *MockClient) WithCompleteFunc(fn func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)) *MockClient {
	m.completeFunc = fn
	return m
}

// WithStreamJSONFunc sets a custom handler for StreamJSON calls.
func (m *MockClient) WithStreamJSONFunc(fn func(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, *StreamResult, error)) *MockClient {
	m.streamJSONFunc = fn
	return m
}

// Complete implements Client.
func (m *MockClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, req)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use custom function if provided
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}

	// Return error if configured
	if m.err != nil {
		return nil, m.err
	}

	// Get response
	response := ""
	if len(m.responses) > 0 {
		response = m.responses[m.responseIdx%len(m.responses)]
		m.responseIdx++
	}

	return &CompletionResponse{
		Content:      response,
		Usage:        TokenUsage{InputTokens: 10, OutputTokens: len(response) / 4, TotalTokens: 10 + len(response)/4},
		FinishReason: "stop",
		Duration:     10 * time.Millisecond,
	}, nil
}

// StreamJSON implements Client.
func (m *MockClient) StreamJSON(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, *StreamResult, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, req)

	// Use custom function if provided
	if m.streamJSONFunc != nil {
		fn := m.streamJSONFunc
		m.mu.Unlock()
		return fn(ctx, req)
	}

	// Return error if configured
	if m.err != nil {
		err := m.err
		m.mu.Unlock()
		return nil, nil, err
	}

	// Get response
	response := ""
	if len(m.responses) > 0 {
		response = m.responses[m.responseIdx%len(m.responses)]
		m.responseIdx++
	}
	m.mu.Unlock()

	events := make(chan StreamEvent)
	result := newStreamResult()

	go func() {
		defer close(events)

		// Check context before sending
		select {
		case <-ctx.Done():
			events <- StreamEvent{Type: StreamEventError, Error: ctx.Err()}
			result.complete(nil, ctx.Err())
			return
		default:
		}

		// Send init event
		events <- StreamEvent{
			Type:      StreamEventInit,
			SessionID: "mock-session-id",
			Init: &InitEvent{
				SessionID: "mock-session-id",
				Model:     "mock-model",
			},
		}

		// Send assistant event with content
		events <- StreamEvent{
			Type:      StreamEventAssistant,
			SessionID: "mock-session-id",
			Assistant: &AssistantEvent{
				MessageID: "mock-msg-id",
				Text:      response,
				Model:     "mock-model",
				Usage: MessageUsage{
					InputTokens:  10,
					OutputTokens: len(response) / 4,
				},
			},
		}

		// Complete with result
		resultEvent := &ResultEvent{
			Subtype:      "success",
			IsError:      false,
			Result:       response,
			SessionID:    "mock-session-id",
			NumTurns:     1,
			TotalCostUSD: 0.001,
			Usage: ResultUsage{
				InputTokens:  10,
				OutputTokens: len(response) / 4,
			},
		}
		result.complete(resultEvent, nil)
	}()

	return events, result, nil
}

// Reset clears the call history and response index.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = nil
	m.responseIdx = 0
}

// CallCount returns the number of times Complete or StreamJSON was called.
func (m *MockClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Calls)
}

// LastCall returns the most recent request, or nil if no calls made.
func (m *MockClient) LastCall() *CompletionRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Calls) == 0 {
		return nil
	}
	req := m.Calls[len(m.Calls)-1]
	return &req
}
