package local

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// JSON-RPC 2.0 protocol types for sidecar communication.

const jsonrpcVersion = "2.0"

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      int64  `json:"id"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int64           `json:"id"`
}

// Notification is a JSON-RPC 2.0 notification (no ID, no response expected).
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	if len(e.Data) > 0 {
		return fmt.Sprintf("RPC error %d: %s (data: %s)", e.Code, e.Message, string(e.Data))
	}
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Application-specific error codes (range -32000 to -32099).
const (
	CodeBackendError    = -32000 // Backend API error
	CodeModelNotFound   = -32001 // Model not found/loaded
	CodeStreamError     = -32002 // Streaming error
	CodeConnectionError = -32003 // Backend connection failed
)

// CompleteParams are the parameters for the "complete" RPC method.
type CompleteParams struct {
	Messages     []MessageParam `json:"messages"`
	Model        string         `json:"model,omitempty"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	MaxTokens    int            `json:"max_tokens,omitempty"`
	Temperature  float64        `json:"temperature,omitempty"`
	Stream       bool           `json:"stream,omitempty"`
	Options      map[string]any `json:"options,omitempty"`
}

// MessageParam is a message in the conversation.
type MessageParam struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"` // For tool results
}

// CompleteResult is the result of a "complete" RPC call.
type CompleteResult struct {
	Content      string      `json:"content"`
	Model        string      `json:"model,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Usage        UsageResult `json:"usage"`
}

// UsageResult tracks token usage.
type UsageResult struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// StreamChunkParams are the parameters for stream.chunk notifications.
type StreamChunkParams struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// StreamDoneParams are the parameters for stream.done notifications.
type StreamDoneParams struct {
	Usage        UsageResult `json:"usage"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Model        string      `json:"model,omitempty"`
}

// InitParams are the parameters for the "init" RPC method.
type InitParams struct {
	Backend    string                     `json:"backend"`
	Model      string                     `json:"model"`
	Host       string                     `json:"host,omitempty"`
	MCPServers map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
	Options    map[string]any             `json:"options,omitempty"`
}

// InitResult is the result of an "init" RPC call.
type InitResult struct {
	Ready   bool   `json:"ready"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

// ShutdownResult is the result of a "shutdown" RPC call.
type ShutdownResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Protocol handles JSON-RPC encoding/decoding over stdio.
type Protocol struct {
	reader   *bufio.Reader
	writer   io.Writer
	enc      *json.Encoder
	writeMu  sync.Mutex // Protects writer
	readMu   sync.Mutex // Protects reader for concurrent reads
	nextID   int64
}

// NewProtocol creates a new JSON-RPC protocol handler.
func NewProtocol(r io.Reader, w io.Writer) *Protocol {
	return &Protocol{
		reader: bufio.NewReader(r),
		writer: w,
		enc:    json.NewEncoder(w),
	}
}

// Call sends a request and waits for a response.
// The result is unmarshaled into the provided value.
// This method is safe for concurrent use, but callers waiting for responses
// will be serialized.
func (p *Protocol) Call(method string, params, result any) error {
	id := atomic.AddInt64(&p.nextID, 1)

	req := Request{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
		ID:      id,
	}

	// Send request
	if err := p.send(req); err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	// Read response - may need to skip notifications
	// Hold the read lock for the entire read loop to ensure we get our response
	p.readMu.Lock()
	defer p.readMu.Unlock()

	for {
		line, err := p.reader.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		// Check if this is a notification (no ID field)
		var msg struct {
			ID *int64 `json:"id"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			return fmt.Errorf("parse message: %w", err)
		}

		// Skip notifications while waiting for our response
		if msg.ID == nil {
			continue
		}

		// Parse as response
		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}

		// Verify ID matches
		if resp.ID != id {
			// Skip responses with wrong ID (shouldn't happen with single-threaded calls)
			continue
		}

		// Check for RPC error
		if resp.Error != nil {
			return resp.Error
		}

		// Unmarshal result
		if result != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("unmarshal result: %w", err)
			}
		}

		return nil
	}
}

// Notify sends a notification (no response expected).
func (p *Protocol) Notify(method string, params any) error {
	notif := Notification{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
	}
	return p.send(notif)
}

// send marshals and writes a message.
func (p *Protocol) send(msg any) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Append newline for line-based protocol
	data = append(data, '\n')

	_, err = p.writer.Write(data)
	return err
}

// ReadMessage reads a single message (response or notification).
// Returns the raw JSON for further processing.
// This method is safe for concurrent use.
func (p *Protocol) ReadMessage() ([]byte, error) {
	p.readMu.Lock()
	defer p.readMu.Unlock()
	return p.reader.ReadBytes('\n')
}

// ParseNotification attempts to parse a message as a notification.
// Returns nil, nil if the message is not a notification.
func ParseNotification(data []byte) (*Notification, error) {
	// Check if this has an ID (response) or not (notification)
	var msg struct {
		ID *int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	// Has ID - not a notification
	if msg.ID != nil {
		return nil, nil
	}

	var notif Notification
	if err := json.Unmarshal(data, &notif); err != nil {
		return nil, err
	}

	return &notif, nil
}

// ParseStreamChunk parses a stream.chunk notification payload.
func ParseStreamChunk(data json.RawMessage) (*StreamChunkParams, error) {
	var params StreamChunkParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}
	return &params, nil
}

// ParseStreamDone parses a stream.done notification payload.
func ParseStreamDone(data json.RawMessage) (*StreamDoneParams, error) {
	var params StreamDoneParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}
	return &params, nil
}

// IsNotification checks if a message is a notification (no ID).
func IsNotification(data []byte) bool {
	var msg struct {
		ID *int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return false
	}
	return msg.ID == nil
}
