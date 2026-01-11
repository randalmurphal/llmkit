package continuedev

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/randalmurphal/llmkit/provider"
)

// ContinueCLI implements provider.Client using the Continue CLI binary (cn).
type ContinueCLI struct {
	path       string
	configPath string
	model      string
	workdir    string
	timeout    time.Duration

	// Tool permissions
	allowedTools  []string
	askTools      []string
	excludedTools []string

	// Environment
	extraEnv map[string]string
	apiKey   string

	// Flags
	verbose bool
	resume  bool
	rule    string
}

// NewContinueCLI creates a new Continue CLI client.
// Assumes "cn" is available in PATH unless overridden with WithPath.
func NewContinueCLI(opts ...Option) *ContinueCLI {
	c := &ContinueCLI{
		path:    "cn",
		timeout: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Complete implements provider.Client.
// Executes cn in headless mode with -p flag and returns the response.
func (c *ContinueCLI) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
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
			return nil, provider.NewError("continue", "complete", cmdCtx.Err(), true)
		}
		return nil, provider.NewError("continue", "complete",
			fmt.Errorf("cn failed: %w\nstderr: %s", err, stderr.String()), false)
	}

	// Parse response
	resp, err := c.parseResponse(stdout.Bytes())
	if err != nil {
		return nil, provider.NewError("continue", "complete",
			fmt.Errorf("parse response: %w", err), false)
	}

	resp.Duration = time.Since(start)
	resp.Model = c.model

	return resp, nil
}

// Stream implements provider.Client.
// Continue CLI doesn't have native streaming in headless mode, so we simulate it.
func (c *ContinueCLI) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)

	go func() {
		defer close(ch)

		// For now, use Complete and send result as single chunk
		// Future: implement actual streaming if cn supports it
		resp, err := c.Complete(ctx, req)
		if err != nil {
			ch <- provider.StreamChunk{Error: err}
			return
		}

		ch <- provider.StreamChunk{
			Content: resp.Content,
			Done:    true,
			Usage:   &resp.Usage,
		}
	}()

	return ch, nil
}

// Provider implements provider.Client.
func (c *ContinueCLI) Provider() string {
	return "continue"
}

// Capabilities implements provider.Client.
func (c *ContinueCLI) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		Streaming:   true,
		Tools:       true,
		MCP:         true,
		Sessions:    true,
		Images:      true,
		NativeTools: []string{"Read", "Write", "Edit", "Bash", "Fetch"},
		ContextFile: "",
	}
}

// Close implements provider.Client.
func (c *ContinueCLI) Close() error {
	return nil
}

// extractPrompt gets the prompt from the request.
func (c *ContinueCLI) extractPrompt(req provider.Request) string {
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
func (c *ContinueCLI) buildArgs(prompt string) []string {
	args := []string{"-p", prompt}

	// Config file
	if c.configPath != "" {
		args = append(args, "--config", c.configPath)
	}

	// Session
	if c.resume {
		args = append(args, "--resume")
	}

	// Rule
	if c.rule != "" {
		args = append(args, "--rule", c.rule)
	}

	// Verbose
	if c.verbose {
		args = append(args, "--verbose")
	}

	// Tool permissions
	for _, allow := range c.allowedTools {
		args = append(args, "--allow", allow)
	}
	for _, ask := range c.askTools {
		args = append(args, "--ask", ask)
	}
	for _, exclude := range c.excludedTools {
		args = append(args, "--exclude", exclude)
	}

	return args
}

// buildEnv constructs environment variables.
func (c *ContinueCLI) buildEnv() []string {
	env := os.Environ()

	// Add API key if set
	if c.apiKey != "" {
		env = append(env, "CONTINUE_API_KEY="+c.apiKey)
	}

	// Add extra env vars
	for k, v := range c.extraEnv {
		env = append(env, k+"="+v)
	}

	return env
}

// parseResponse parses cn output.
// In headless mode, cn outputs just the final response text.
func (c *ContinueCLI) parseResponse(output []byte) (*provider.Response, error) {
	// Try to parse as JSON first (if --format json was used)
	var jsonResp continueJSONResponse
	if err := json.Unmarshal(output, &jsonResp); err == nil && jsonResp.Content != "" {
		return &provider.Response{
			Content:      jsonResp.Content,
			FinishReason: jsonResp.FinishReason,
			Usage: provider.TokenUsage{
				InputTokens:  jsonResp.Usage.InputTokens,
				OutputTokens: jsonResp.Usage.OutputTokens,
				TotalTokens:  jsonResp.Usage.InputTokens + jsonResp.Usage.OutputTokens,
			},
		}, nil
	}

	// Otherwise treat as plain text
	content := strings.TrimSpace(string(output))
	if content == "" {
		return nil, fmt.Errorf("empty response from cn")
	}

	return &provider.Response{
		Content:      content,
		FinishReason: "stop",
	}, nil
}

// continueJSONResponse is the JSON output format from cn --format json.
type continueJSONResponse struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// StreamWithProcess implements streaming by reading process output line by line.
// This is an alternative implementation that provides incremental output.
func (c *ContinueCLI) StreamWithProcess(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	prompt := c.extractPrompt(req)
	if prompt == "" {
		return nil, fmt.Errorf("no prompt provided")
	}

	args := c.buildArgs(prompt)

	cmdCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		cmdCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(cmdCtx, c.path, args...)
	cmd.Dir = c.workdir
	cmd.Env = c.buildEnv()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, provider.NewError("continue", "stream",
			fmt.Errorf("stdout pipe: %w", err), false)
	}

	if err := cmd.Start(); err != nil {
		return nil, provider.NewError("continue", "stream",
			fmt.Errorf("start: %w", err), false)
	}

	ch := make(chan provider.StreamChunk)

	go func() {
		defer close(ch)
		defer cmd.Wait() //nolint:errcheck

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

		if err := scanner.Err(); err != nil {
			ch <- provider.StreamChunk{Error: err}
			return
		}

		ch <- provider.StreamChunk{
			Done: true,
		}
	}()

	return ch, nil
}
