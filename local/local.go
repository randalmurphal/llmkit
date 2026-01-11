package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/randalmurphal/llmkit/provider"
)

// Client implements provider.Client for local LLM models via a Python sidecar.
type Client struct {
	cfg     Config
	sidecar *Sidecar

	mu      sync.Mutex // Protects sidecar lifecycle
	started bool
}

// NewClient creates a new local model client.
// The sidecar process is not started until the first request.
func NewClient(opts ...Option) *Client {
	c := &Client{
		cfg: DefaultConfig(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewClientWithConfig creates a new local model client from a Config.
func NewClientWithConfig(cfg Config) *Client {
	return &Client{
		cfg: cfg.WithDefaults(),
	}
}

// Complete implements provider.Client.
// Starts the sidecar if not already running.
func (c *Client) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	if err := c.ensureStarted(ctx); err != nil {
		return nil, provider.NewError("local", "complete", err, false)
	}

	proto := c.sidecar.Protocol()
	if proto == nil {
		return nil, provider.NewError("local", "complete", errors.New("sidecar not running"), false)
	}

	// Build RPC params
	params := c.buildCompleteParams(req, false)

	// Make the call with timeout
	callCtx := ctx
	if c.cfg.RequestTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, c.cfg.RequestTimeout)
		defer cancel()
	}

	start := time.Now()
	var result CompleteResult

	// Use goroutine for context cancellation support
	type callResult struct {
		err error
	}
	resultCh := make(chan callResult, 1)

	go func() {
		err := proto.Call("complete", params, &result)
		resultCh <- callResult{err: err}
	}()

	select {
	case <-callCtx.Done():
		return nil, provider.NewError("local", "complete", callCtx.Err(), isRetryableError(callCtx.Err()))
	case r := <-resultCh:
		if r.err != nil {
			return nil, provider.NewError("local", "complete", r.err, isRetryableRPCError(r.err))
		}
	}

	return &provider.Response{
		Content:      result.Content,
		Model:        result.Model,
		FinishReason: result.FinishReason,
		Duration:     time.Since(start),
		Usage: provider.TokenUsage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
			TotalTokens:  result.Usage.TotalTokens,
		},
	}, nil
}

// Stream implements provider.Client.
// Starts the sidecar if not already running.
func (c *Client) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	if err := c.ensureStarted(ctx); err != nil {
		return nil, provider.NewError("local", "stream", err, false)
	}

	proto := c.sidecar.Protocol()
	if proto == nil {
		return nil, provider.NewError("local", "stream", errors.New("sidecar not running"), false)
	}

	// Build RPC params with streaming enabled
	params := c.buildCompleteParams(req, true)

	// Send the initial request
	if err := proto.Notify("stream.start", params); err != nil {
		return nil, provider.NewError("local", "stream", fmt.Errorf("send stream request: %w", err), false)
	}

	// Create output channel
	ch := make(chan provider.StreamChunk)

	// Start goroutine to read stream notifications
	go c.readStreamNotifications(ctx, proto, ch)

	return ch, nil
}

// Provider implements provider.Client.
func (c *Client) Provider() string {
	return "local"
}

// Capabilities implements provider.Client.
func (c *Client) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		Streaming:   true,
		Tools:       false,
		MCP:         true, // MCP via sidecar
		Sessions:    false,
		Images:      false,
		NativeTools: nil,
		ContextFile: "",
	}
}

// Close implements provider.Client.
// Stops the sidecar process if running.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sidecar != nil {
		return c.sidecar.Stop()
	}
	return nil
}

// ensureStarted starts the sidecar if not already running.
// If the sidecar crashed, it will be automatically restarted.
func (c *Client) ensureStarted(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if sidecar is running
	if c.started && c.sidecar != nil && c.sidecar.IsRunning() {
		return nil
	}

	// Check if sidecar crashed and needs restart
	if c.started && c.sidecar != nil {
		// Sidecar was started but is no longer running - it crashed
		exitErr := c.sidecar.ExitError()
		if exitErr != nil {
			// Log the crash and attempt restart
			slog.Warn("sidecar crashed, attempting restart",
				slog.Any("exit_error", exitErr))
		}
		// Clean up the old sidecar
		_ = c.sidecar.Stop()
		c.sidecar = nil
		c.started = false
	}

	// Validate config before starting
	if err := c.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Create and start sidecar
	c.sidecar = NewSidecar(c.cfg)
	if err := c.sidecar.Start(ctx); err != nil {
		return err
	}

	c.started = true
	return nil
}

// buildCompleteParams converts a provider.Request to RPC CompleteParams.
func (c *Client) buildCompleteParams(req provider.Request, stream bool) CompleteParams {
	params := CompleteParams{
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Stream:       stream,
		Options:      req.Options,
	}

	// Use client's model if not specified in request
	if params.Model == "" {
		params.Model = c.cfg.Model
	}

	// Convert messages
	params.Messages = make([]MessageParam, len(req.Messages))
	for i, m := range req.Messages {
		params.Messages[i] = MessageParam{
			Role:    string(m.Role),
			Content: m.GetText(), // Use GetText() to handle multimodal
			Name:    m.Name,
		}
	}

	return params
}

// readStreamNotifications reads stream notifications from the protocol.
func (c *Client) readStreamNotifications(ctx context.Context, proto *Protocol, ch chan<- provider.StreamChunk) {
	defer close(ch)

	for {
		select {
		case <-ctx.Done():
			ch <- provider.StreamChunk{Error: ctx.Err()}
			return
		default:
		}

		// Read next message
		data, err := proto.ReadMessage()
		if err != nil {
			ch <- provider.StreamChunk{Error: provider.NewError("local", "stream", fmt.Errorf("read message: %w", err), false)}
			return
		}

		// Parse as notification
		notif, err := ParseNotification(data)
		if err != nil {
			ch <- provider.StreamChunk{Error: provider.NewError("local", "stream", fmt.Errorf("parse notification: %w", err), false)}
			return
		}

		// Skip non-notifications (shouldn't happen during streaming)
		if notif == nil {
			continue
		}

		// Handle notification by method
		switch notif.Method {
		case "stream.chunk":
			paramsData, err := json.Marshal(notif.Params)
			if err != nil {
				ch <- provider.StreamChunk{Error: provider.NewError("local", "stream", fmt.Errorf("marshal chunk params: %w", err), false)}
				return
			}
			chunk, err := ParseStreamChunk(paramsData)
			if err != nil {
				ch <- provider.StreamChunk{Error: provider.NewError("local", "stream", fmt.Errorf("parse chunk: %w", err), false)}
				return
			}

			ch <- provider.StreamChunk{
				Content: chunk.Content,
				Done:    chunk.Done,
			}

			if chunk.Done {
				return
			}

		case "stream.done":
			paramsData, err := json.Marshal(notif.Params)
			if err != nil {
				ch <- provider.StreamChunk{Error: provider.NewError("local", "stream", fmt.Errorf("marshal done params: %w", err), false)}
				return
			}
			done, err := ParseStreamDone(paramsData)
			if err != nil {
				ch <- provider.StreamChunk{Error: provider.NewError("local", "stream", fmt.Errorf("parse done: %w", err), false)}
				return
			}

			ch <- provider.StreamChunk{
				Done: true,
				Usage: &provider.TokenUsage{
					InputTokens:  done.Usage.InputTokens,
					OutputTokens: done.Usage.OutputTokens,
					TotalTokens:  done.Usage.TotalTokens,
				},
			}
			return

		case "stream.error":
			// Parse error notification
			var errParams struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			paramsData, _ := json.Marshal(notif.Params)
			if err := json.Unmarshal(paramsData, &errParams); err == nil {
				ch <- provider.StreamChunk{
					Error: provider.NewError("local", "stream", fmt.Errorf("stream error %d: %s", errParams.Code, errParams.Message), false),
				}
			} else {
				ch <- provider.StreamChunk{
					Error: provider.NewError("local", "stream", errors.New("unknown stream error"), false),
				}
			}
			return

		default:
			// Ignore unknown notifications
		}
	}
}

// isRetryableError checks if a standard error is retryable.
func isRetryableError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// isRetryableRPCError checks if an RPC error is retryable.
func isRetryableRPCError(err error) bool {
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		if rpcErr.Code == CodeConnectionError {
			return true
		}
	}
	return false
}
