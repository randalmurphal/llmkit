package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Sidecar process state.
type sidecarState int

const (
	sidecarStopped sidecarState = iota
	sidecarStarting
	sidecarRunning
	sidecarStopping
)

// Sidecar manages the Python sidecar process lifecycle.
type Sidecar struct {
	cfg Config

	mu       sync.RWMutex
	state    sidecarState
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	protocol *Protocol
	done     chan struct{} // Closed when process exits
	exitErr  error
}

// NewSidecar creates a new sidecar manager.
func NewSidecar(cfg Config) *Sidecar {
	return &Sidecar{
		cfg:   cfg.WithDefaults(),
		state: sidecarStopped,
	}
}

// Start launches the sidecar process and waits for it to become ready.
// Returns an error if the process fails to start or doesn't become ready
// within the configured timeout.
func (s *Sidecar) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.state != sidecarStopped {
		s.mu.Unlock()
		return fmt.Errorf("sidecar already %s", s.stateString())
	}
	s.state = sidecarStarting
	s.mu.Unlock()

	// Build command
	cmd := exec.CommandContext(ctx, s.cfg.PythonPath, s.cfg.SidecarPath)

	// Set working directory
	if s.cfg.WorkDir != "" {
		cmd.Dir = s.cfg.WorkDir
	}

	// Set environment
	if len(s.cfg.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range s.cfg.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.setState(sidecarStopped)
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		s.setState(sidecarStopped)
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		s.setState(sidecarStopped)
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		s.setState(sidecarStopped)
		return fmt.Errorf("start sidecar: %w", err)
	}

	// Store references
	s.mu.Lock()
	s.cmd = cmd
	s.stdin = stdin
	s.stdout = stdout
	s.stderr = stderr
	s.protocol = NewProtocol(stdout, stdin)
	s.done = make(chan struct{})
	s.mu.Unlock()

	// Start goroutine to monitor process exit
	go s.waitForExit()

	// Start goroutine to log stderr
	go s.drainStderr()

	// Initialize sidecar
	initCtx, cancel := context.WithTimeout(ctx, s.cfg.StartupTimeout)
	defer cancel()

	if err := s.initialize(initCtx); err != nil {
		_ = s.Stop()
		return fmt.Errorf("initialize sidecar: %w", err)
	}

	s.setState(sidecarRunning)
	slog.Debug("sidecar started",
		slog.String("backend", string(s.cfg.Backend)),
		slog.String("model", s.cfg.Model))

	return nil
}

// Stop gracefully shuts down the sidecar process.
func (s *Sidecar) Stop() error {
	s.mu.Lock()
	if s.state == sidecarStopped || s.state == sidecarStopping {
		s.mu.Unlock()
		return nil
	}
	s.state = sidecarStopping
	s.mu.Unlock()

	// Try graceful shutdown first
	shutdownErr := s.shutdown()

	// Close stdin to signal EOF
	s.mu.RLock()
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	s.mu.RUnlock()

	// Wait for process to exit with timeout
	select {
	case <-s.done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Force kill
		s.mu.RLock()
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		s.mu.RUnlock()
		<-s.done
	}

	s.setState(sidecarStopped)
	slog.Debug("sidecar stopped")

	return shutdownErr
}

// IsRunning returns true if the sidecar is running.
func (s *Sidecar) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == sidecarRunning
}

// Protocol returns the JSON-RPC protocol handler.
// Returns nil if the sidecar is not running.
func (s *Sidecar) Protocol() *Protocol {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state != sidecarRunning {
		return nil
	}
	return s.protocol
}

// Done returns a channel that's closed when the sidecar process exits.
func (s *Sidecar) Done() <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.done == nil {
		// Return a closed channel if never started
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return s.done
}

// ExitError returns the error from the sidecar process exit, if any.
func (s *Sidecar) ExitError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exitErr
}

// Restart stops and starts the sidecar.
func (s *Sidecar) Restart(ctx context.Context) error {
	if err := s.Stop(); err != nil {
		slog.Warn("error stopping sidecar during restart", slog.Any("error", err))
	}
	return s.Start(ctx)
}

// initialize sends the init RPC to configure the sidecar.
func (s *Sidecar) initialize(ctx context.Context) error {
	s.mu.RLock()
	proto := s.protocol
	s.mu.RUnlock()

	if proto == nil {
		return errors.New("protocol not initialized")
	}

	params := InitParams{
		Backend:    string(s.cfg.Backend),
		Model:      s.cfg.Model,
		Host:       s.cfg.Host,
		MCPServers: s.cfg.MCPServers,
	}

	var result InitResult

	// Use a separate goroutine for the call so we can respect context cancellation
	type callResult struct {
		err error
	}
	resultCh := make(chan callResult, 1)

	go func() {
		err := proto.Call("init", params, &result)
		resultCh <- callResult{err: err}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-resultCh:
		if r.err != nil {
			return r.err
		}
	}

	if !result.Ready {
		if result.Message != "" {
			return fmt.Errorf("sidecar not ready: %s", result.Message)
		}
		return errors.New("sidecar not ready")
	}

	return nil
}

// shutdown sends the shutdown RPC.
func (s *Sidecar) shutdown() error {
	s.mu.RLock()
	proto := s.protocol
	s.mu.RUnlock()

	if proto == nil {
		return nil
	}

	var result ShutdownResult
	if err := proto.Call("shutdown", nil, &result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("shutdown failed: %s", result.Message)
	}

	return nil
}

// waitForExit waits for the process to exit and captures the error.
func (s *Sidecar) waitForExit() {
	s.mu.RLock()
	cmd := s.cmd
	done := s.done
	s.mu.RUnlock()

	if cmd == nil {
		return
	}

	err := cmd.Wait()

	s.mu.Lock()
	s.exitErr = err
	s.mu.Unlock()

	close(done)
}

// drainStderr reads and logs stderr output.
func (s *Sidecar) drainStderr() {
	s.mu.RLock()
	stderr := s.stderr
	s.mu.RUnlock()

	if stderr == nil {
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			// Log stderr as debug messages
			slog.Debug("sidecar stderr", slog.String("output", string(buf[:n])))
		}
		if err != nil {
			break
		}
	}
}

// setState updates the sidecar state.
func (s *Sidecar) setState(state sidecarState) {
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
}

// stateString returns a human-readable state string.
func (s *Sidecar) stateString() string {
	switch s.state {
	case sidecarStopped:
		return "stopped"
	case sidecarStarting:
		return "starting"
	case sidecarRunning:
		return "running"
	case sidecarStopping:
		return "stopping"
	default:
		return "unknown"
	}
}
