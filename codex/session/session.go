package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/randalmurphal/llmkit/codexcontract"
)

// Session manages a long-running Codex app-server process with JSON-RPC 2.0 I/O.
type Session interface {
	// ID returns the session identifier (same as ThreadID for Codex sessions).
	ID() string

	// ThreadID returns the Codex thread UUID assigned by the app-server.
	// Empty until the thread/start handshake completes.
	ThreadID() string

	// Send sends a user message as a turn/start JSON-RPC request.
	// Returns immediately; responses arrive via the Output channel.
	Send(ctx context.Context, msg UserMessage) error

	// Steer injects input into an actively running turn via turn/steer.
	// This is a Codex-specific capability not available in Claude sessions.
	Steer(ctx context.Context, msg UserMessage) error

	// Output returns a channel of parsed output notifications from the app-server.
	// The channel is closed when the session ends.
	Output() <-chan OutputMessage

	// Close terminates the session and releases resources.
	Close() error

	// Status returns the current session state.
	Status() SessionStatus

	// Info returns session metadata.
	Info() SessionInfo

	// Wait blocks until the session completes.
	Wait() error

	// WaitForInit blocks until the thread/start handshake completes and
	// the thread ID is available.
	WaitForInit(ctx context.Context) error
}

// session implements Session.
type session struct {
	id     string // thread ID, set after handshake
	config sessionConfig

	// Process management
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cancel context.CancelFunc

	// JSON-RPC request ID counter
	nextID atomic.Int64

	// Pending request waiters: request ID -> response channel.
	// The readOutput goroutine delivers responses here.
	pending   map[int64]chan *JSONRPCResponse
	pendingMu sync.Mutex

	// Output handling
	outputCh chan OutputMessage

	// State
	status       atomic.Value // SessionStatus
	createdAt    time.Time
	lastActivity atomic.Value // time.Time
	turnCount    atomic.Int32

	// Active turn tracking for turn/steer support.
	// Updated by readOutput when it sees turn.started/turn.completed notifications.
	activeTurnID atomic.Value // string (empty string means no active turn)

	// Lifecycle
	initDone chan struct{} // Closed when thread handshake completes
	done     chan struct{} // Closed when readOutput exits
	closeErr error
	closeMu  sync.Mutex
}

// newSession creates a new session with the given configuration.
func newSession(ctx context.Context, opts ...SessionOption) (*session, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	s := &session{
		config:   cfg,
		pending:  make(map[int64]chan *JSONRPCResponse),
		outputCh: make(chan OutputMessage, 100),
		initDone: make(chan struct{}),
		done:     make(chan struct{}),
		createdAt: time.Now(),
	}
	s.status.Store(StatusCreating)
	s.lastActivity.Store(time.Now())
	s.activeTurnID.Store("")

	if err := s.start(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

// start spawns the Codex app-server process, begins output processing,
// and performs the initialize + thread/start handshake.
func (s *session) start(ctx context.Context) error {
	// Create cancellable context for process lifetime.
	procCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	args := s.buildArgs()
	s.cmd = exec.CommandContext(procCtx, s.config.codexPath, args...)

	// Create a new process group so we can kill all child processes when the
	// session closes. Without this, child processes become orphans.
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	s.setupEnv()

	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	// Let stderr pass through for error visibility.
	s.cmd.Stderr = os.Stderr

	if err := s.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start codex app-server: %w", err)
	}

	// Start output reader goroutine.
	go s.readOutput()

	// Perform the two-step handshake with a timeout.
	handshakeCtx, handshakeCancel := context.WithTimeout(ctx, s.config.startupTimeout)
	defer handshakeCancel()

	if err := s.initializeHandshake(handshakeCtx); err != nil {
		_ = s.Close()
		return fmt.Errorf("handshake: %w", err)
	}

	s.status.Store(StatusActive)
	return nil
}

// initializeHandshake performs the required two-step protocol handshake:
// 1. initialize — exchanges client/server capabilities
// 2. thread/start or thread/resume — creates or resumes a thread with config
func (s *session) initializeHandshake(ctx context.Context) error {
	// Step 1: Initialize — must complete before any other method.
	initParams := InitializeParams{
		ClientInfo: ClientInfo{
			Name:    "llmkit",
			Version: "1.9.2",
		},
	}

	initResp, err := s.sendRequest(ctx, MethodInitialize, initParams)
	if err != nil {
		return fmt.Errorf("send initialize: %w", err)
	}
	if initResp.Error != nil {
		return fmt.Errorf("initialize error: %s", initResp.Error.Message)
	}

	// Step 2: thread/start or thread/resume with configuration.
	var method string
	var params any

	if s.config.resume && s.config.threadID != "" {
		method = MethodThreadResume
		params = ThreadResumeParams{ThreadID: s.config.threadID}
	} else {
		method = MethodThreadStart
		params = s.buildThreadStartParams()
	}

	resp, err := s.sendRequest(ctx, method, params)
	if err != nil {
		return fmt.Errorf("send %s: %w", method, err)
	}
	if resp.Error != nil {
		return fmt.Errorf("%s error: %s", method, resp.Error.Message)
	}

	var result ThreadStartResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse %s result: %w", method, err)
	}

	threadID := result.Thread.ID
	if threadID == "" {
		return fmt.Errorf("%s returned empty thread ID (raw: %s)", method, string(resp.Result))
	}

	s.id = threadID
	close(s.initDone)
	return nil
}

// buildThreadStartParams constructs the thread/start params from session config.
// Model, working directory, system prompt, approval policy, and sandbox mode are
// all passed through thread/start params — NOT as CLI flags.
func (s *session) buildThreadStartParams() ThreadStartParams {
	params := ThreadStartParams{}

	if s.config.model != "" {
		params.Model = s.config.model
	}
	if s.config.workdir != "" {
		params.CWD = s.config.workdir
	}
	if s.config.systemPrompt != "" {
		params.BaseInstructions = s.config.systemPrompt
	}

	if s.config.fullAuto {
		params.ApprovalPolicy = "never"
		params.Sandbox = "workspace-write"
	} else {
		if s.config.approvalMode != "" {
			params.ApprovalPolicy = s.config.approvalMode
		}
		if s.config.sandboxMode != "" {
			params.Sandbox = s.config.sandboxMode
		}
	}

	return params
}

// buildArgs constructs CLI arguments for app-server mode.
// App-server only accepts --config/-c, --enable, --disable, and --listen.
// Model, sandbox, approval, and system prompt go through the thread/start
// JSON-RPC params, not CLI flags.
func (s *session) buildArgs() []string {
	args := []string{codexcontract.CommandAppServer}

	for _, feature := range s.config.enabledFeatures {
		args = append(args, codexcontract.FlagEnable, feature)
	}
	for _, feature := range s.config.disabledFeatures {
		args = append(args, codexcontract.FlagDisable, feature)
	}

	return args
}

// setupEnv configures environment variables for the process.
func (s *session) setupEnv() {
	if len(s.config.extraEnv) == 0 && s.config.workdir == "" {
		return
	}

	if len(s.config.extraEnv) > 0 {
		s.cmd.Env = os.Environ()
		for k, v := range s.config.extraEnv {
			s.cmd.Env = setEnvVar(s.cmd.Env, k, v)
		}
	}

	if s.config.workdir != "" {
		s.cmd.Dir = s.config.workdir
	}
}

// setEnvVar updates or adds an environment variable.
func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// sendRequest sends a JSON-RPC request and waits for the corresponding response.
func (s *session) sendRequest(ctx context.Context, method string, params any) (*JSONRPCResponse, error) {
	id := s.nextID.Add(1)

	req := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	// Register a waiter before writing, so the readOutput goroutine can
	// deliver the response even if it arrives before we start waiting.
	waiter := make(chan *JSONRPCResponse, 1)
	s.pendingMu.Lock()
	s.pending[id] = waiter
	s.pendingMu.Unlock()

	// Clean up the waiter if we return without receiving a response.
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
	}()

	// Write the request.
	writeDone := make(chan error, 1)
	go func() {
		_, writeErr := s.stdin.Write(data)
		writeDone <- writeErr
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case writeErr := <-writeDone:
		if writeErr != nil {
			return nil, fmt.Errorf("write request: %w", writeErr)
		}
	}

	// Wait for the response.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		return nil, fmt.Errorf("session closed before response received")
	case resp := <-waiter:
		if resp == nil {
			return nil, fmt.Errorf("session closed before response received")
		}
		return resp, nil
	}
}

// readOutput reads lines from stdout, classifies them as responses or
// notifications, and dispatches accordingly.
func (s *session) readOutput() {
	defer close(s.outputCh)
	defer close(s.done)

	scanner := bufio.NewScanner(s.stdout)
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Make a copy since scanner reuses the buffer.
		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)

		// Try to classify as a response first.
		if resp, isResp := parseJSONRPCLine(lineCopy); isResp {
			s.deliverResponse(resp)
			continue
		}

		// Otherwise, treat as a notification.
		msg, err := ParseOutputMessage(lineCopy)
		if err != nil {
			// Unparseable line - skip it.
			continue
		}

		s.updateFromMessage(msg)

		// Send to output channel (non-blocking with drop-oldest on full).
		select {
		case s.outputCh <- *msg:
		default:
			select {
			case <-s.outputCh:
				s.outputCh <- *msg
			default:
			}
		}
	}

	if err := scanner.Err(); err != nil {
		s.setCloseError(fmt.Errorf("read output: %w", err))
	}

	// Wait for process to exit.
	if err := s.cmd.Wait(); err != nil {
		s.setCloseError(fmt.Errorf("process exited: %w", err))
	}

	// Fail any pending waiters so they don't hang forever.
	s.failPendingRequests()

	s.status.Store(StatusClosed)
}

// deliverResponse routes a JSON-RPC response to its pending waiter.
func (s *session) deliverResponse(resp *JSONRPCResponse) {
	if resp.ID == nil {
		return
	}

	s.pendingMu.Lock()
	waiter, ok := s.pending[*resp.ID]
	if ok {
		delete(s.pending, *resp.ID)
	}
	s.pendingMu.Unlock()

	if ok {
		waiter <- resp
	}
}

// failPendingRequests unblocks all pending waiters when the process exits.
// Sends nil rather than closing channels so that receivers get a value
// instead of the zero value from a closed channel, which would cause
// nil pointer panics in callers like threadHandshake.
func (s *session) failPendingRequests() {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	for id, waiter := range s.pending {
		select {
		case waiter <- nil:
		default:
		}
		delete(s.pending, id)
	}
}

// updateFromMessage updates session state based on notification content.
func (s *session) updateFromMessage(msg *OutputMessage) {
	s.lastActivity.Store(time.Now())

	if msg.IsTurnStarted() && msg.TurnID != "" {
		s.activeTurnID.Store(msg.TurnID)
	}

	if msg.IsTurnComplete() || msg.IsTurnFailed() {
		s.activeTurnID.Store("")
		s.turnCount.Add(1)
	}
}

// WaitForInit waits for the thread handshake to complete.
func (s *session) WaitForInit(ctx context.Context) error {
	select {
	case <-s.initDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return fmt.Errorf("session closed before init")
	}
}

// ID implements Session.
func (s *session) ID() string {
	return s.id
}

// ThreadID implements Session. Returns the same value as ID() since
// the session ID is the Codex thread UUID.
func (s *session) ThreadID() string {
	return s.id
}

// Send implements Session.
func (s *session) Send(ctx context.Context, msg UserMessage) error {
	if s.Status() != StatusActive {
		return fmt.Errorf("session not active: %s", s.Status())
	}

	params := TurnStartParams{
		ThreadID: s.id,
		Input: []InputItem{
			{Type: "text", Text: msg.Content},
		},
	}

	// Fire-and-forget: the JSON-RPC response for turn/start is not awaited
	// because turn results arrive as notifications on the output channel.
	// Any JSON-RPC error response will be delivered to readOutput, which
	// has no registered waiter for this ID, so the response is dropped.
	// This is intentional — errors during a turn surface as turn.failed
	// notifications, not as JSON-RPC error responses.
	id := s.nextID.Add(1)
	req := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodTurnStart,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal turn/start: %w", err)
	}
	data = append(data, '\n')

	writeDone := make(chan error, 1)
	go func() {
		_, writeErr := s.stdin.Write(data)
		writeDone <- writeErr
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case writeErr := <-writeDone:
		if writeErr != nil {
			return fmt.Errorf("write turn/start: %w", writeErr)
		}
	}

	s.lastActivity.Store(time.Now())
	return nil
}

// Steer implements Session. It injects input into an actively running turn.
// Returns an error if there is no active turn, since expectedTurnId is required.
func (s *session) Steer(ctx context.Context, msg UserMessage) error {
	if s.Status() != StatusActive {
		return fmt.Errorf("session not active: %s", s.Status())
	}

	turnID := s.activeTurnID.Load().(string)
	if turnID == "" {
		return fmt.Errorf("no active turn to steer")
	}

	params := TurnSteerParams{
		ThreadID:       s.id,
		ExpectedTurnID: turnID,
		Input: []InputItem{
			{Type: "text", Text: msg.Content},
		},
	}

	// Fire-and-forget: same rationale as Send — turn/steer results arrive
	// as notifications, not as JSON-RPC responses.
	id := s.nextID.Add(1)
	req := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodTurnSteer,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal turn/steer: %w", err)
	}
	data = append(data, '\n')

	writeDone := make(chan error, 1)
	go func() {
		_, writeErr := s.stdin.Write(data)
		writeDone <- writeErr
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case writeErr := <-writeDone:
		if writeErr != nil {
			return fmt.Errorf("write turn/steer: %w", writeErr)
		}
	}

	s.lastActivity.Store(time.Now())
	return nil
}

// Output implements Session.
func (s *session) Output() <-chan OutputMessage {
	return s.outputCh
}

// Close implements Session.
func (s *session) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	status := s.Status()
	if status == StatusClosed || status == StatusClosing {
		return s.closeErr
	}

	s.status.Store(StatusClosing)

	// Send a shutdown JSON-RPC request (best effort).
	s.sendShutdown()

	// Close stdin to signal EOF.
	if s.stdin != nil {
		_ = s.stdin.Close()
	}

	// Wait for readOutput to finish (process exits after receiving shutdown + EOF).
	select {
	case <-s.done:
		// readOutput() finished, process has exited.
	case <-time.After(5 * time.Second):
		// Graceful shutdown timed out, force kill via context.
		if s.cancel != nil {
			s.cancel()
		}
		select {
		case <-s.done:
			// Process killed successfully.
		case <-time.After(2 * time.Second):
			// Still not dead, kill entire process group as last resort.
			if s.cmd != nil && s.cmd.Process != nil {
				_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
			}
			select {
			case <-s.done:
			case <-time.After(1 * time.Second):
				s.setCloseError(fmt.Errorf("process did not exit after kill"))
			}
		}
	}

	s.status.Store(StatusClosed)
	return s.closeErr
}

// sendShutdown sends a shutdown JSON-RPC request. Best effort - errors are ignored
// because the process may already be exiting.
func (s *session) sendShutdown() {
	id := s.nextID.Add(1)
	req := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodShutdown,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return
	}
	data = append(data, '\n')

	// Non-blocking write with a short timeout.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.stdin.Write(data)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		// Write timed out, proceed with close.
	}
}

// Status implements Session.
func (s *session) Status() SessionStatus {
	return s.status.Load().(SessionStatus)
}

// Info implements Session.
func (s *session) Info() SessionInfo {
	return SessionInfo{
		ID:           s.id,
		ThreadID:     s.id,
		Status:       s.Status(),
		Model:        s.config.model,
		CWD:          s.config.workdir,
		CreatedAt:    s.createdAt,
		LastActivity: s.lastActivity.Load().(time.Time),
		TurnCount:    int(s.turnCount.Load()),
	}
}

// Wait implements Session.
func (s *session) Wait() error {
	<-s.done
	return s.closeErr
}

// setCloseError sets the close error if not already set.
func (s *session) setCloseError(err error) {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closeErr == nil {
		s.closeErr = err
	}
}
