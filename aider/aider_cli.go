package aider

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/randalmurphal/llmkit/provider"
)

// AiderCLI implements provider.Client using the Aider CLI binary.
type AiderCLI struct {
	path    string
	model   string
	workdir string
	timeout time.Duration

	// Files
	editableFiles []string
	readOnlyFiles []string

	// Git control
	noGit         bool
	noAutoCommits bool

	// Output control
	noStream bool
	dryRun   bool

	// Confirmation
	yesAlways bool

	// Edit format
	editFormat string

	// Environment
	extraEnv      map[string]string
	ollamaAPIBase string
}

// NewAiderCLI creates a new Aider CLI client.
// Assumes "aider" is available in PATH unless overridden with WithPath.
func NewAiderCLI(opts ...Option) *AiderCLI {
	c := &AiderCLI{
		path:    "aider",
		timeout: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Complete implements provider.Client.
// Executes aider with --message flag and returns the response.
func (c *AiderCLI) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	start := time.Now()

	// Get the last user message as the prompt
	prompt := c.extractPrompt(req)
	if prompt == "" {
		return nil, fmt.Errorf("no prompt provided")
	}

	// Build command arguments
	args := c.buildArgs(prompt)

	// Create command with timeout
	cmdCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		cmdCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(cmdCtx, c.path, args...)
	cmd.Dir = c.workdir
	cmd.Env = c.buildEnv()

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()
	if err != nil {
		// Check for context cancellation
		if cmdCtx.Err() != nil {
			return nil, provider.NewError("aider", "complete", cmdCtx.Err(), true)
		}
		return nil, provider.NewError("aider", "complete",
			fmt.Errorf("aider failed: %w\nstderr: %s", err, stderr.String()), false)
	}

	// Parse response
	resp := ParseAiderOutput(stdout.String())
	resp.Duration = time.Since(start)
	resp.Model = c.model

	return resp, nil
}

// Stream implements provider.Client.
// Aider streams output by default, so we read it line by line.
func (c *AiderCLI) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	prompt := c.extractPrompt(req)
	if prompt == "" {
		return nil, provider.NewError("aider", "stream", fmt.Errorf("no prompt provided"), false)
	}

	args := c.buildArgs(prompt)

	cmdCtx := ctx
	var cancel context.CancelFunc
	if c.timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, c.timeout)
	}

	cmd := exec.CommandContext(cmdCtx, c.path, args...)
	cmd.Dir = c.workdir
	cmd.Env = c.buildEnv()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, provider.NewError("aider", "stream",
			fmt.Errorf("stdout pipe: %w", err), false)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, provider.NewError("aider", "stream",
			fmt.Errorf("stderr pipe: %w", err), false)
	}

	if err := cmd.Start(); err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, provider.NewError("aider", "stream",
			fmt.Errorf("start: %w", err), false)
	}

	ch := make(chan provider.StreamChunk)

	go func() {
		defer close(ch)
		if cancel != nil {
			defer cancel() // Cancel context when goroutine completes
		}

		// Read stderr in background with proper synchronization
		var stderrBuf bytes.Buffer
		var stderrWg sync.WaitGroup
		stderrWg.Add(1)
		go func() {
			defer stderrWg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				stderrBuf.WriteString(scanner.Text())
				stderrBuf.WriteString("\n")
			}
		}()

		// Read stdout
		scanner := bufio.NewScanner(stdout)
		var content strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			content.WriteString(line)
			content.WriteString("\n")

			ch <- provider.StreamChunk{
				Content: line + "\n",
				Done:    false,
			}
		}

		// Wait for stderr goroutine to complete before accessing stderrBuf
		stderrWg.Wait()

		// Wait for command to finish
		err := cmd.Wait()
		if err != nil {
			ch <- provider.StreamChunk{
				Error: provider.NewError("aider", "stream",
					fmt.Errorf("aider failed: %w\nstderr: %s", err, stderrBuf.String()), false),
			}
			return
		}

		if scanErr := scanner.Err(); scanErr != nil {
			ch <- provider.StreamChunk{Error: scanErr}
			return
		}

		ch <- provider.StreamChunk{
			Done: true,
		}
	}()

	return ch, nil
}

// Provider implements provider.Client.
func (c *AiderCLI) Provider() string {
	return "aider"
}

// Capabilities implements provider.Client.
func (c *AiderCLI) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		Streaming:   true,
		Tools:       true,
		MCP:         false, // MCP support pending
		Sessions:    false,
		Images:      false,
		NativeTools: []string{"file_edit", "shell"},
		ContextFile: "",
	}
}

// Close implements provider.Client.
func (c *AiderCLI) Close() error {
	return nil
}

// extractPrompt gets the prompt from the request.
func (c *AiderCLI) extractPrompt(req provider.Request) string {
	// Use system prompt + last user message
	var parts []string

	if req.SystemPrompt != "" {
		parts = append(parts, req.SystemPrompt)
	}

	// Find last user message
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == provider.RoleUser {
			parts = append(parts, req.Messages[i].GetText())
			break
		}
	}

	return strings.Join(parts, "\n\n")
}

// buildArgs constructs CLI arguments.
func (c *AiderCLI) buildArgs(prompt string) []string {
	args := []string{
		"--message", prompt,
		"--no-pretty",       // Always disable for scripting
		"--no-fancy-input",  // Always disable for scripting
	}

	// Model
	if c.model != "" {
		args = append(args, "--model", c.model)
	}

	// Timeout (in seconds)
	if c.timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", int(c.timeout.Seconds())))
	}

	// Confirmation
	if c.yesAlways {
		args = append(args, "--yes-always")
	}

	// Streaming
	if c.noStream {
		args = append(args, "--no-stream")
	}

	// Dry run
	if c.dryRun {
		args = append(args, "--dry-run")
	}

	// Git control
	if c.noGit {
		args = append(args, "--no-git")
	}
	if c.noAutoCommits {
		args = append(args, "--no-auto-commits")
	}

	// Edit format
	if c.editFormat != "" {
		args = append(args, "--edit-format", c.editFormat)
	}

	// Files
	for _, f := range c.editableFiles {
		args = append(args, "--file", f)
	}
	for _, f := range c.readOnlyFiles {
		args = append(args, "--read", f)
	}

	return args
}

// buildEnv constructs environment variables.
func (c *AiderCLI) buildEnv() []string {
	env := os.Environ()

	// Add Ollama API base if set
	if c.ollamaAPIBase != "" {
		env = setEnvVar(env, "OLLAMA_API_BASE", c.ollamaAPIBase)
	}

	// Add extra env vars
	for k, v := range c.extraEnv {
		env = setEnvVar(env, k, v)
	}

	return env
}

// setEnvVar sets or replaces an environment variable in the env slice.
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
