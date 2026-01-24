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

	"github.com/randalmurphal/llmkit/claudecontract"
)

// Session manages a long-running Claude CLI process with stream-json I/O.
type Session interface {
	// ID returns the session identifier.
	// Note: May be empty until the first message exchange triggers the init.
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
	// Note: Some fields may be empty until the first message exchange.
	Info() SessionInfo

	// Wait blocks until the session completes.
	Wait() error

	// WaitForInit blocks until the init message is received.
	// Call this after Send() if you need session metadata before proceeding.
	WaitForInit(ctx context.Context) error

	// JSONLPath returns the path to Claude Code's session JSONL file.
	// This file contains the full session history written in real-time.
	// Returns empty string if session ID is not yet available or path cannot be determined.
	// Path format: ~/.claude/projects/{normalized-workdir}/{sessionId}.jsonl
	JSONLPath() string
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
		id:        cfg.sessionID, // Use provided session ID immediately (if any)
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
//
// Note: Claude CLI in stream-json mode only outputs the init message after
// receiving the first user message. The session becomes active immediately
// after the process starts, and the init message is captured when the first
// Send() triggers a response. Session metadata (ID, model) may be empty until
// the first message exchange.
func (s *session) start(ctx context.Context) error {
	// Create cancellable context for process lifetime
	procCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Build command arguments
	args := s.buildArgs()
	s.cmd = exec.CommandContext(procCtx, s.config.claudePath, args...)

	// Create a new process group so we can kill all child processes (MCP servers,
	// browsers, etc.) when the session closes. Without this, child processes become
	// orphans and accumulate, eventually exhausting system resources.
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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

	// Claude CLI in stream-json mode outputs the init message only after
	// receiving the first user message. We don't wait for init here -
	// it will be captured by updateFromMessage() when the first response
	// comes through.
	s.status.Store(StatusActive)
	return nil
}

// buildArgs constructs CLI arguments for stream-json mode.
func (s *session) buildArgs() []string {
	args := []string{
		claudecontract.FlagInputFormat, claudecontract.FormatStreamJSON,
		claudecontract.FlagOutputFormat, claudecontract.FormatStreamJSON,
		claudecontract.FlagVerbose,
	}

	// Session management
	if s.config.sessionID != "" {
		if s.config.resume {
			args = append(args, claudecontract.FlagResume, s.config.sessionID)
		} else {
			args = append(args, claudecontract.FlagSessionID, s.config.sessionID)
		}
	}
	// Note: --no-session-persistence only works with --print mode
	// Sessions don't use --print, so we skip this flag here
	// Session persistence is controlled by Claude Code's behavior

	// Model
	if s.config.model != "" {
		args = append(args, claudecontract.FlagModel, s.config.model)
	}
	if s.config.fallbackModel != "" {
		args = append(args, claudecontract.FlagFallbackModel, s.config.fallbackModel)
	}

	// System prompt
	if s.config.systemPrompt != "" {
		args = append(args, claudecontract.FlagSystemPrompt, s.config.systemPrompt)
	}
	if s.config.appendSystemPrompt != "" {
		args = append(args, claudecontract.FlagAppendSystemPrompt, s.config.appendSystemPrompt)
	}

	// Tools
	for _, tool := range s.config.allowedTools {
		args = append(args, claudecontract.FlagAllowedTools, tool)
	}
	// Note: Claude CLI uses camelCase for tool flags (--allowedTools, --disallowedTools)
	for _, tool := range s.config.disallowedTools {
		args = append(args, claudecontract.FlagDisallowedTools, tool)
	}
	if len(s.config.tools) > 0 {
		args = append(args, claudecontract.FlagTools, strings.Join(s.config.tools, ","))
	}

	// Permissions
	if s.config.dangerouslySkipPermissions {
		args = append(args, claudecontract.FlagDangerouslySkipPermissions)
	}
	if s.config.permissionMode != "" {
		args = append(args, claudecontract.FlagPermissionMode, s.config.permissionMode)
	}
	if len(s.config.settingSources) > 0 {
		args = append(args, claudecontract.FlagSettingSources, strings.Join(s.config.settingSources, ","))
	}

	// Directories
	for _, dir := range s.config.addDirs {
		args = append(args, claudecontract.FlagAddDir, dir)
	}

	// Limits
	if s.config.maxBudgetUSD > 0 {
		args = append(args, claudecontract.FlagMaxBudgetUSD, fmt.Sprintf("%.6f", s.config.maxBudgetUSD))
	}
	// NOTE: maxTurns is not supported by Claude CLI - the flag doesn't exist.
	// The config value is silently ignored.

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

// WaitForInit waits for the initialization message to be received.
// This is useful if you need session metadata (ID, model, tools) before
// proceeding. Call this after the first Send() to ensure init is available.
//
// Note: In normal usage, you don't need to call this - just Send() a message
// and the init will be captured automatically.
func (s *session) WaitForInit(ctx context.Context) error {
	// Check if already initialized
	if s.initMsg != nil {
		return nil
	}

	// Wait for init message to arrive
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.done:
			return fmt.Errorf("session closed before init")
		default:
			if s.initMsg != nil {
				return nil
			}
			time.Sleep(10 * time.Millisecond)
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

	// Wait for the readOutput goroutine to finish, which will happen when:
	// 1. Claude CLI exits after receiving EOF on stdin
	// 2. The process is killed via context cancellation
	// The s.done channel is closed by readOutput() when it exits.
	select {
	case <-s.done:
		// readOutput() finished, process has exited
	case <-time.After(5 * time.Second):
		// Graceful shutdown timed out, force kill via context
		if s.cancel != nil {
			s.cancel()
		}
		// Wait a bit more for the forced exit
		select {
		case <-s.done:
			// Process killed successfully
		case <-time.After(2 * time.Second):
			// Still not dead, kill entire process group as last resort.
			// Using negative PID kills all processes in the group (Claude CLI +
			// MCP servers + browsers), preventing orphaned child processes.
			if s.cmd != nil && s.cmd.Process != nil {
				_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
			}
			// Final wait with hard timeout
			select {
			case <-s.done:
			case <-time.After(1 * time.Second):
				// Give up waiting - process may be zombie
				s.setCloseError(fmt.Errorf("process did not exit after kill"))
			}
		}
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

// JSONLPath implements Session.
// Returns the path to Claude Code's session JSONL file.
func (s *session) JSONLPath() string {
	sessionID := s.id
	if sessionID == "" {
		return ""
	}

	// Get workdir - either from config or use cwd
	workdir := s.config.workdir
	if workdir == "" {
		// If no explicit workdir, Claude uses the cwd where it was started
		// We can't reliably determine this, so return empty
		return ""
	}

	// Determine home directory for .claude location
	homeDir := s.config.homeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return ""
		}
	}

	// Claude Code normalizes the project path:
	// /home/user/repos/project -> -home-user-repos-project
	normalizedPath := normalizeProjectPath(workdir)

	// Build the full path: ~/.claude/projects/{normalized-path}/{sessionId}.jsonl
	return fmt.Sprintf("%s/.claude/projects/%s/%s.jsonl", homeDir, normalizedPath, sessionID)
}

// normalizeProjectPath converts an absolute path to Claude Code's normalized format.
// Example: /home/user/repos/project -> -home-user-repos-project
func normalizeProjectPath(path string) string {
	// Remove leading slash and replace remaining slashes with dashes
	normalized := strings.TrimPrefix(path, "/")
	normalized = strings.ReplaceAll(normalized, "/", "-")
	// Prepend dash to match Claude Code's format
	return "-" + normalized
}
