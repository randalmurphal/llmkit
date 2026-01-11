package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SandboxMode specifies the sandbox level for file system operations.
type SandboxMode string

// Sandbox mode constants.
const (
	SandboxReadOnly         SandboxMode = "read-only"
	SandboxWorkspaceWrite   SandboxMode = "workspace-write"
	SandboxDangerFullAccess SandboxMode = "danger-full-access"
)

// ApprovalMode specifies when to ask for user approval.
type ApprovalMode string

// Approval mode constants.
const (
	ApprovalUntrusted ApprovalMode = "untrusted"
	ApprovalOnFailure ApprovalMode = "on-failure"
	ApprovalOnRequest ApprovalMode = "on-request"
	ApprovalNever     ApprovalMode = "never"
)

// CodexCLI implements Client using the Codex CLI binary.
type CodexCLI struct {
	path    string
	model   string
	workdir string
	timeout time.Duration

	// Sandbox mode
	sandboxMode SandboxMode

	// Approval mode
	approvalMode ApprovalMode
	fullAuto     bool

	// Session management
	sessionID string

	// Search
	enableSearch bool

	// Additional directories
	addDirs []string

	// Images
	images []string

	// Environment control
	extraEnv map[string]string

	// Note: Codex does not support --mcp-config CLI flag.
	// MCP servers are configured via ~/.codex/config.toml only.
	// Use WithMCPServers to programmatically add servers to config.
}

// Note: Codex MCP configuration is handled via ~/.codex/config.toml,
// not via CLI flags. Use the `codex mcp add` command or edit config.toml directly.

// CodexOption configures CodexCLI.
type CodexOption func(*CodexCLI)

// NewCodexCLI creates a new Codex CLI client.
// Assumes "codex" is available in PATH unless overridden with WithCodexPath.
func NewCodexCLI(opts ...CodexOption) *CodexCLI {
	c := &CodexCLI{
		path:        "codex",
		timeout:     5 * time.Minute,
		sandboxMode: SandboxWorkspaceWrite, // Safe default
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithCodexPath sets the path to the codex binary.
func WithCodexPath(path string) CodexOption {
	return func(c *CodexCLI) { c.path = path }
}

// WithModel sets the default model.
func WithModel(model string) CodexOption {
	return func(c *CodexCLI) { c.model = model }
}

// WithWorkdir sets the working directory for codex commands.
func WithWorkdir(dir string) CodexOption {
	return func(c *CodexCLI) { c.workdir = dir }
}

// WithTimeout sets the default timeout for commands.
func WithTimeout(d time.Duration) CodexOption {
	return func(c *CodexCLI) { c.timeout = d }
}

// WithSandboxMode sets the sandbox mode for file system operations.
// Options: "read-only", "workspace-write", "danger-full-access"
func WithSandboxMode(mode SandboxMode) CodexOption {
	return func(c *CodexCLI) { c.sandboxMode = mode }
}

// WithApprovalMode sets when to ask for user approval.
// Options: "untrusted", "on-failure", "on-request", "never"
func WithApprovalMode(mode ApprovalMode) CodexOption {
	return func(c *CodexCLI) { c.approvalMode = mode }
}

// WithFullAuto enables automatic approvals (equivalent to --full-auto).
func WithFullAuto() CodexOption {
	return func(c *CodexCLI) { c.fullAuto = true }
}

// WithSessionID sets the session ID for resuming sessions.
func WithSessionID(id string) CodexOption {
	return func(c *CodexCLI) { c.sessionID = id }
}

// WithSearch enables web search capabilities.
func WithSearch() CodexOption {
	return func(c *CodexCLI) { c.enableSearch = true }
}

// WithAddDir adds an additional directory to the accessible paths.
func WithAddDir(dir string) CodexOption {
	return func(c *CodexCLI) {
		c.addDirs = append(c.addDirs, dir)
	}
}

// WithAddDirs adds multiple additional directories to the accessible paths.
func WithAddDirs(dirs []string) CodexOption {
	return func(c *CodexCLI) {
		c.addDirs = append(c.addDirs, dirs...)
	}
}

// WithImage adds an image to attach to the request.
func WithImage(imagePath string) CodexOption {
	return func(c *CodexCLI) {
		c.images = append(c.images, imagePath)
	}
}

// WithImages adds multiple images to attach to the request.
func WithImages(imagePaths []string) CodexOption {
	return func(c *CodexCLI) {
		c.images = append(c.images, imagePaths...)
	}
}

// WithEnv adds additional environment variables to the CLI process.
func WithEnv(env map[string]string) CodexOption {
	return func(c *CodexCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		for k, v := range env {
			c.extraEnv[k] = v
		}
	}
}

// WithEnvVar adds a single environment variable to the CLI process.
func WithEnvVar(key, value string) CodexOption {
	return func(c *CodexCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		c.extraEnv[key] = value
	}
}

// Note: Codex does not support --mcp-config CLI flag.
// MCP servers must be configured via ~/.codex/config.toml.
// Use `codex mcp add <server-name> -- <command>` to add servers.

// CLIEvent represents a JSON event from Codex CLI output.
type CLIEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id,omitempty"`
	Message   string          `json:"message,omitempty"`
	Content   string          `json:"content,omitempty"`
	Error     string          `json:"error,omitempty"`
	Usage     *CLIUsage       `json:"usage,omitempty"`
	ToolCall  *CLIToolCall    `json:"tool_call,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
}

// CLIUsage contains token usage from the CLI response.
type CLIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// CLIToolCall represents a tool call event.
type CLIToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// setupCmd configures the command with working directory and environment variables.
func (c *CodexCLI) setupCmd(cmd *exec.Cmd) {
	if c.workdir != "" {
		cmd.Dir = c.workdir
	}

	// Only set Env if we have custom environment variables to add.
	if len(c.extraEnv) > 0 {
		cmd.Env = os.Environ()
		for k, v := range c.extraEnv {
			cmd.Env = setEnvVar(cmd.Env, k, v)
		}
	}
}

// setEnvVar updates or adds an environment variable in an env slice.
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

// Complete implements Client.
func (c *CodexCLI) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	args, cleanup := c.buildArgsWithCleanup(req)
	defer cleanup()

	cmd := exec.CommandContext(ctx, c.path, args...)
	c.setupCmd(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, NewError("complete", ctx.Err(), false)
		}
		errMsg := sanitizeStderr(stderr.String())
		retryable := isRetryableError(errMsg)
		return nil, NewError("complete", fmt.Errorf("%w: %s", err, errMsg), retryable)
	}

	resp := c.parseResponse(stdout.Bytes())
	resp.Duration = time.Since(start)

	return resp, nil
}

// Stream implements Client.
func (c *CodexCLI) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	args, cleanup := c.buildArgsWithCleanup(req)
	cmd := exec.CommandContext(ctx, c.path, args...)
	c.setupCmd(cmd)
	cmd.Stdin = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cleanup()
		return nil, NewError("stream", fmt.Errorf("create stdout pipe: %w", err), false)
	}

	if err := cmd.Start(); err != nil {
		cleanup()
		return nil, NewError("stream", fmt.Errorf("start command: %w", err), false)
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer cleanup() // Clean up temp files when stream completes
		var cmdErr error
		defer func() {
			cmdErr = cmd.Wait()
			if cmdErr != nil {
				select {
				case ch <- StreamChunk{Error: NewError("stream", fmt.Errorf("command failed: %w", cmdErr), false)}:
				default:
				}
			}
		}()

		scanner := bufio.NewScanner(stdout)
		var accumulatedContent strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Try to parse as JSON event
			var event CLIEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				// Not JSON, treat as raw text
				accumulatedContent.WriteString(line)
				accumulatedContent.WriteString("\n")
				select {
				case ch <- StreamChunk{Content: line + "\n"}:
				case <-ctx.Done():
					ch <- StreamChunk{Error: ctx.Err()}
					return
				}
				continue
			}

			// Handle different event types
			chunk := c.processEvent(&event, &accumulatedContent)
			if chunk != nil {
				select {
				case ch <- *chunk:
				case <-ctx.Done():
					ch <- StreamChunk{Error: ctx.Err()}
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: NewError("stream", fmt.Errorf("read output: %w", err), false)}
			return
		}

		// Send final chunk if we didn't get a completion event
		select {
		case ch <- StreamChunk{Done: true}:
		default:
		}
	}()

	return ch, nil
}

// processEvent converts a CLI event to a stream chunk.
func (c *CodexCLI) processEvent(event *CLIEvent, content *strings.Builder) *StreamChunk {
	switch event.Type {
	case "content", "text", "assistant":
		text := event.Content
		if text == "" {
			text = event.Message
		}
		if text != "" {
			content.WriteString(text)
			return &StreamChunk{Content: text}
		}
	case "tool_call":
		if event.ToolCall != nil {
			return &StreamChunk{
				ToolCalls: []ToolCall{{
					ID:        event.ToolCall.ID,
					Name:      event.ToolCall.Name,
					Arguments: event.ToolCall.Arguments,
				}},
			}
		}
	case "done", "complete", "end":
		chunk := &StreamChunk{Done: true}
		if event.Usage != nil {
			chunk.Usage = &TokenUsage{
				InputTokens:  event.Usage.InputTokens,
				OutputTokens: event.Usage.OutputTokens,
				TotalTokens:  event.Usage.TotalTokens,
			}
		}
		return chunk
	case "error":
		errMsg := event.Error
		if errMsg == "" {
			errMsg = event.Message
		}
		return &StreamChunk{Error: NewError("stream", fmt.Errorf("%s", errMsg), false)}
	}
	return nil
}

// buildArgsWithCleanup constructs CLI arguments and returns a cleanup function for temp files.
func (c *CodexCLI) buildArgsWithCleanup(req CompletionRequest) ([]string, func()) {
	// Use exec subcommand for non-interactive mode with JSON output
	args := []string{"exec", "--json"}

	// Model
	model := c.model
	if req.Model != "" {
		model = req.Model
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// Sandbox mode
	if c.sandboxMode != "" {
		args = append(args, "--sandbox", string(c.sandboxMode))
	}

	// Approval mode
	if c.approvalMode != "" {
		args = append(args, "--ask-for-approval", string(c.approvalMode))
	}

	// Full auto mode
	if c.fullAuto {
		args = append(args, "--full-auto")
	}

	// Working directory
	if c.workdir != "" {
		args = append(args, "--cd", c.workdir)
	}

	// Additional directories
	for _, dir := range c.addDirs {
		args = append(args, "--add-dir", dir)
	}

	// Images
	for _, img := range c.images {
		args = append(args, "--image", img)
	}

	// Search
	if c.enableSearch {
		args = append(args, "--search")
	}

	// Note: MCP is configured via ~/.codex/config.toml, not CLI flags

	// Build prompt from messages
	prompt := c.buildPrompt(req)
	args = append(args, prompt)

	// Return no-op cleanup function (no temp files needed)
	cleanup := func() {}

	return args, cleanup
}

// buildPrompt constructs the prompt from messages.
func (c *CodexCLI) buildPrompt(req CompletionRequest) string {
	var prompt strings.Builder

	// Include system prompt if provided
	if req.SystemPrompt != "" {
		prompt.WriteString("System: ")
		prompt.WriteString(req.SystemPrompt)
		prompt.WriteString("\n\n")
	}

	// Build conversation from messages
	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleUser:
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		case RoleAssistant:
			if prompt.Len() > 0 {
				prompt.WriteString("\nAssistant: ")
				prompt.WriteString(msg.Content)
				prompt.WriteString("\n\nUser: ")
			}
		}
	}

	return strings.TrimSpace(prompt.String())
}

// parseResponse extracts response data from CLI output.
func (c *CodexCLI) parseResponse(data []byte) *CompletionResponse {
	resp := &CompletionResponse{
		Model:        c.model,
		FinishReason: "stop",
	}

	// Codex uses newline-delimited JSON events
	lines := bytes.Split(data, []byte("\n"))
	var contentBuilder strings.Builder
	var toolCalls []ToolCall

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var event CLIEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// If JSON parsing fails, treat as raw text
			contentBuilder.Write(line)
			contentBuilder.WriteString("\n")
			continue
		}

		switch event.Type {
		case "content", "text", "assistant", "message":
			text := event.Content
			if text == "" {
				text = event.Message
			}
			contentBuilder.WriteString(text)
		case "tool_call":
			if event.ToolCall != nil {
				toolCalls = append(toolCalls, ToolCall{
					ID:        event.ToolCall.ID,
					Name:      event.ToolCall.Name,
					Arguments: event.ToolCall.Arguments,
				})
			}
		case "session":
			if event.SessionID != "" {
				resp.SessionID = event.SessionID
			}
		case "usage":
			if event.Usage != nil {
				resp.Usage = TokenUsage{
					InputTokens:  event.Usage.InputTokens,
					OutputTokens: event.Usage.OutputTokens,
					TotalTokens:  event.Usage.TotalTokens,
				}
			}
		case "done", "complete", "end":
			if event.Usage != nil {
				resp.Usage = TokenUsage{
					InputTokens:  event.Usage.InputTokens,
					OutputTokens: event.Usage.OutputTokens,
					TotalTokens:  event.Usage.TotalTokens,
				}
			}
			if event.SessionID != "" {
				resp.SessionID = event.SessionID
			}
		case "error":
			errMsg := event.Error
			if errMsg == "" {
				errMsg = event.Message
			}
			resp.FinishReason = "error"
			if resp.Content == "" {
				resp.Content = errMsg
			}
		case "result":
			// Try to extract content from result
			if event.Result != nil {
				var resultData struct {
					Content string `json:"content"`
					Text    string `json:"text"`
				}
				if err := json.Unmarshal(event.Result, &resultData); err == nil {
					if resultData.Content != "" {
						contentBuilder.WriteString(resultData.Content)
					} else if resultData.Text != "" {
						contentBuilder.WriteString(resultData.Text)
					}
				}
			}
		}
	}

	resp.Content = strings.TrimSpace(contentBuilder.String())
	resp.ToolCalls = toolCalls

	// Calculate total tokens if not provided
	if resp.Usage.TotalTokens == 0 && (resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0) {
		resp.Usage.TotalTokens = resp.Usage.InputTokens + resp.Usage.OutputTokens
	}

	// Log warning if no content was extracted
	if resp.Content == "" && len(resp.ToolCalls) == 0 {
		preview := string(data)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		slog.Warn("codex CLI returned empty response",
			slog.String("output_preview", preview))
	}

	return resp
}

// Resume resumes a previous session by ID.
// Uses `codex exec resume <SESSION_ID>` for non-interactive mode.
func (c *CodexCLI) Resume(ctx context.Context, sessionID, prompt string) (*CompletionResponse, error) {
	start := time.Now()

	// Use exec resume for non-interactive mode with JSON output
	args := []string{"exec", "resume", sessionID, "--json"}
	if prompt != "" {
		args = append(args, prompt)
	}

	cmd := exec.CommandContext(ctx, c.path, args...)
	c.setupCmd(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, NewError("resume", ctx.Err(), false)
		}
		errMsg := sanitizeStderr(stderr.String())
		retryable := isRetryableError(errMsg)
		return nil, NewError("resume", fmt.Errorf("%w: %s", err, errMsg), retryable)
	}

	resp := c.parseResponse(stdout.Bytes())
	resp.Duration = time.Since(start)
	resp.SessionID = sessionID

	return resp, nil
}

// isRetryableError checks if an error message indicates a transient error.
func isRetryableError(errMsg string) bool {
	errLower := strings.ToLower(errMsg)
	return strings.Contains(errLower, "rate limit") ||
		strings.Contains(errLower, "timeout") ||
		strings.Contains(errLower, "overloaded") ||
		strings.Contains(errLower, "503") ||
		strings.Contains(errLower, "529") ||
		strings.Contains(errLower, "429")
}

// maxStderrLength limits stderr output in error messages.
const maxStderrLength = 500

// sanitizeStderr prepares stderr output for inclusion in error messages.
func sanitizeStderr(stderr string) string {
	if len(stderr) > maxStderrLength {
		stderr = stderr[:maxStderrLength] + "... (truncated)"
	}
	return strings.TrimSpace(stderr)
}

// Provider returns the provider name.
func (c *CodexCLI) Provider() string {
	return "codex"
}

// Capabilities returns Codex CLI's native capabilities.
func (c *CodexCLI) Capabilities() Capabilities {
	return Capabilities{
		Streaming: true,
		Tools:     true,
		MCP:       false, // MCP requires config.toml, not supported via CLI flags
		Sessions:  true,
		Images:    true,
		// Codex native tools based on documentation
		NativeTools: []string{"shell", "apply_diff", "read_file", "list_dir", "web_search"},
		ContextFile: "AGENTS.md",
	}
}

// Close releases any resources held by the client.
func (c *CodexCLI) Close() error {
	return nil
}
