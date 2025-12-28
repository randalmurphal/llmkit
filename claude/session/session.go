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
	"time"
)

// Session manages a long-running Claude CLI process with stream-json I/O.
type Session interface {
	// ID returns the session identifier.
	ID() string

	// Send sends a user message to Claude and returns immediately.
	// Messages are delivered via the Output channel.
	Send(ctx context.Context, msg UserMessage) error

	// Output returns a channel of parsed output messages from Claude.
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
}

// session implements Session.
type session struct {
	id     string
	config sessionConfig

	// Process management
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cancel context.CancelFunc

	// Output handling
	outputCh chan OutputMessage
	initMsg  *InitMessage

	// State
	status       atomic.Value // SessionStatus
	createdAt    time.Time
	lastActivity atomic.Value // time.Time
	turnCount    atomic.Int32
	totalCost    atomic.Value // float64

	// Lifecycle
	done     chan struct{}
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
		config:    cfg,
		outputCh:  make(chan OutputMessage, 100),
		done:      make(chan struct{}),
		createdAt: time.Now(),
	}
	s.status.Store(StatusCreating)
	s.lastActivity.Store(time.Now())
	s.totalCost.Store(0.0)

	if err := s.start(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

// start spawns the Claude CLI process and begins output processing.
func (s *session) start(ctx context.Context) error {
	// Create cancellable context for process lifetime
	procCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Build command arguments
	args := s.buildArgs()
	s.cmd = exec.CommandContext(procCtx, s.config.claudePath, args...)
	s.setupEnv()

	// Create pipes
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

	// Capture stderr for error messages
	s.cmd.Stderr = os.Stderr // TODO: capture to buffer for error reporting

	// Start the process
	if err := s.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start claude: %w", err)
	}

	// Start output reader goroutine
	go s.readOutput()

	// Wait for initialization with timeout
	initCtx, initCancel := context.WithTimeout(ctx, s.config.startupTimeout)
	defer initCancel()

	if err := s.waitForInit(initCtx); err != nil {
		_ = s.Close() // Best effort cleanup on init failure
		return fmt.Errorf("wait for init: %w", err)
	}

	s.status.Store(StatusActive)
	return nil
}

// buildArgs constructs CLI arguments for stream-json mode.
func (s *session) buildArgs() []string {
	args := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
	}

	// Session management
	if s.config.sessionID != "" {
		if s.config.resume {
			args = append(args, "--resume", s.config.sessionID)
		} else {
			args = append(args, "--session-id", s.config.sessionID)
		}
	}
	if s.config.noSessionPersistence {
		args = append(args, "--no-session-persistence")
	}

	// Model
	if s.config.model != "" {
		args = append(args, "--model", s.config.model)
	}
	if s.config.fallbackModel != "" {
		args = append(args, "--fallback-model", s.config.fallbackModel)
	}

	// System prompt
	if s.config.systemPrompt != "" {
		args = append(args, "--system-prompt", s.config.systemPrompt)
	}
	if s.config.appendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", s.config.appendSystemPrompt)
	}

	// Tools
	for _, tool := range s.config.allowedTools {
		args = append(args, "--allowedTools", tool)
	}
	for _, tool := range s.config.disallowedTools {
		args = append(args, "--disallowed-tools", tool)
	}
	if len(s.config.tools) > 0 {
		args = append(args, "--tools", strings.Join(s.config.tools, ","))
	}

	// Permissions
	if s.config.dangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if s.config.permissionMode != "" {
		args = append(args, "--permission-mode", s.config.permissionMode)
	}
	if len(s.config.settingSources) > 0 {
		args = append(args, "--setting-sources", strings.Join(s.config.settingSources, ","))
	}

	// Directories
	for _, dir := range s.config.addDirs {
		args = append(args, "--add-dir", dir)
	}

	// Limits
	if s.config.maxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.6f", s.config.maxBudgetUSD))
	}
	if s.config.maxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", s.config.maxTurns))
	}

	return args
}

// setupEnv configures environment variables for the process.
func (s *session) setupEnv() {
	if s.config.homeDir == "" && s.config.configDir == "" && len(s.config.extraEnv) == 0 {
		return
	}

	s.cmd.Env = os.Environ()

	if s.config.homeDir != "" {
		s.cmd.Env = setEnvVar(s.cmd.Env, "HOME", s.config.homeDir)
	}
	if s.config.configDir != "" {
		s.cmd.Env = setEnvVar(s.cmd.Env, "CLAUDE_CONFIG_DIR", s.config.configDir)
	}
	for k, v := range s.config.extraEnv {
		s.cmd.Env = setEnvVar(s.cmd.Env, k, v)
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

// readOutput reads and parses JSON lines from stdout.
func (s *session) readOutput() {
	defer close(s.outputCh)
	defer close(s.done)

	scanner := bufio.NewScanner(s.stdout)
	// Increase buffer size for large messages
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		msg, err := ParseOutputMessage(line)
		if err != nil {
			// Log parse error but continue
			continue
		}

		// Update session state from messages
		s.updateFromMessage(msg)

		// Filter hook output if not requested
		if msg.IsHook() && !s.config.includeHookOutput {
			continue
		}

		// Send to output channel (non-blocking with select)
		select {
		case s.outputCh <- *msg:
		default:
			// Channel full, drop oldest
			select {
			case <-s.outputCh:
				s.outputCh <- *msg
			default:
			}
		}
	}

	// Process ended
	if err := scanner.Err(); err != nil {
		s.setCloseError(fmt.Errorf("read output: %w", err))
	}

	// Wait for process to exit
	if err := s.cmd.Wait(); err != nil {
		s.setCloseError(fmt.Errorf("process exited: %w", err))
	}

	s.status.Store(StatusClosed)
}

// updateFromMessage updates session state based on message content.
func (s *session) updateFromMessage(msg *OutputMessage) {
	s.lastActivity.Store(time.Now())

	if msg.IsInit() && msg.Init != nil {
		s.initMsg = msg.Init
		s.id = msg.SessionID
	}

	if msg.IsResult() && msg.Result != nil {
		s.turnCount.Add(1)
		cost := s.totalCost.Load().(float64)
		s.totalCost.Store(cost + msg.Result.TotalCostUSD)
	}
}

// waitForInit waits for the initialization message.
func (s *session) waitForInit(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-s.outputCh:
			if !ok {
				return fmt.Errorf("session closed before init")
			}
			if msg.IsInit() {
				// Re-send to channel for consumer
				s.outputCh <- msg
				return nil
			}
			// Re-queue non-init messages
			s.outputCh <- msg
		}
	}
}

// ID implements Session.
func (s *session) ID() string {
	return s.id
}

// Send implements Session.
func (s *session) Send(ctx context.Context, msg UserMessage) error {
	if s.Status() != StatusActive {
		return fmt.Errorf("session not active: %s", s.Status())
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Add newline delimiter
	data = append(data, '\n')

	// Write with context timeout
	done := make(chan error, 1)
	go func() {
		_, err := s.stdin.Write(data)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("write message: %w", err)
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

	// Close stdin to signal EOF to Claude
	if s.stdin != nil {
		_ = s.stdin.Close() // Best effort
	}

	// Give process time to exit gracefully
	done := make(chan struct{})
	go func() {
		_ = s.cmd.Wait() // Captured in closeErr via readOutput
		close(done)
	}()

	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(5 * time.Second):
		// Force kill
		if s.cancel != nil {
			s.cancel()
		}
		<-done
	}

	s.status.Store(StatusClosed)
	return s.closeErr
}

// Status implements Session.
func (s *session) Status() SessionStatus {
	return s.status.Load().(SessionStatus)
}

// Info implements Session.
func (s *session) Info() SessionInfo {
	return SessionInfo{
		ID:           s.id,
		Status:       s.Status(),
		Model:        s.getModel(),
		CWD:          s.config.workdir,
		CreatedAt:    s.createdAt,
		LastActivity: s.lastActivity.Load().(time.Time),
		TurnCount:    int(s.turnCount.Load()),
		TotalCostUSD: s.totalCost.Load().(float64),
	}
}

// getModel returns the model from init message or config.
func (s *session) getModel() string {
	if s.initMsg != nil && s.initMsg.Model != "" {
		return s.initMsg.Model
	}
	return s.config.model
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
