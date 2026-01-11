package gemini

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

// OutputFormat specifies the CLI output format.
type OutputFormat string

// Output format constants.
const (
	OutputFormatText       OutputFormat = "text"
	OutputFormatJSON       OutputFormat = "json"
	OutputFormatStreamJSON OutputFormat = "stream-json"
)

// GeminiCLI implements Client using the Gemini CLI binary.
type GeminiCLI struct {
	path    string
	model   string
	workdir string
	timeout time.Duration

	// Output control
	outputFormat OutputFormat

	// Tool control
	allowedTools    []string
	disallowedTools []string

	// Permissions
	yolo bool // Auto-approve all actions (--yolo flag)

	// Context
	systemPrompt string

	// Budget and limits
	maxTurns int

	// Environment control
	extraEnv map[string]string

	// MCP configuration
	mcpConfigPath string                     // --mcp path
	mcpServers    map[string]MCPServerConfig // Inline server definitions

	// Sandbox configuration
	sandbox string // "host" (default), "docker", "remote-execution"

	// Additional directories to include
	includeDirs []string
}

// GeminiOption configures GeminiCLI.
type GeminiOption func(*GeminiCLI)

// NewGeminiCLI creates a new Gemini CLI client.
// Assumes "gemini" is available in PATH unless overridden with WithGeminiPath.
// By default uses JSON output format for structured responses.
func NewGeminiCLI(opts ...GeminiOption) *GeminiCLI {
	c := &GeminiCLI{
		path:         "gemini",
		timeout:      5 * time.Minute,
		outputFormat: OutputFormatJSON, // Default to JSON for structured data
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithGeminiPath sets the path to the gemini binary.
func WithGeminiPath(path string) GeminiOption {
	return func(c *GeminiCLI) { c.path = path }
}

// WithModel sets the default model.
func WithModel(model string) GeminiOption {
	return func(c *GeminiCLI) { c.model = model }
}

// WithWorkdir sets the working directory for gemini commands.
func WithWorkdir(dir string) GeminiOption {
	return func(c *GeminiCLI) { c.workdir = dir }
}

// WithTimeout sets the default timeout for commands.
func WithTimeout(d time.Duration) GeminiOption {
	return func(c *GeminiCLI) { c.timeout = d }
}

// WithAllowedTools sets the allowed tools for gemini (whitelist).
func WithAllowedTools(tools []string) GeminiOption {
	return func(c *GeminiCLI) { c.allowedTools = tools }
}

// WithDisallowedTools sets the tools to disallow (blacklist).
func WithDisallowedTools(tools []string) GeminiOption {
	return func(c *GeminiCLI) { c.disallowedTools = tools }
}

// WithOutputFormat sets the output format (text, json, stream-json).
// Default is json for structured responses.
func WithOutputFormat(format OutputFormat) GeminiOption {
	return func(c *GeminiCLI) { c.outputFormat = format }
}

// WithYolo enables auto-approval of all actions (no prompts).
// Use this for non-interactive execution in trusted environments.
// WARNING: This allows Gemini to execute any tools without confirmation.
func WithYolo() GeminiOption {
	return func(c *GeminiCLI) { c.yolo = true }
}

// WithSystemPrompt sets a custom system prompt.
func WithSystemPrompt(prompt string) GeminiOption {
	return func(c *GeminiCLI) { c.systemPrompt = prompt }
}

// WithMaxTurns limits the number of agentic turns in a conversation.
// A value of 0 means no limit.
func WithMaxTurns(n int) GeminiOption {
	return func(c *GeminiCLI) { c.maxTurns = n }
}

// WithEnv adds additional environment variables to the CLI process.
// These are merged with the parent environment and any other configured env vars.
func WithEnv(env map[string]string) GeminiOption {
	return func(c *GeminiCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		for k, v := range env {
			c.extraEnv[k] = v
		}
	}
}

// WithEnvVar adds a single environment variable to the CLI process.
func WithEnvVar(key, value string) GeminiOption {
	return func(c *GeminiCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		c.extraEnv[key] = value
	}
}

// WithMCPConfig sets the path to an MCP configuration file.
func WithMCPConfig(path string) GeminiOption {
	return func(c *GeminiCLI) {
		c.mcpConfigPath = path
	}
}

// WithMCPServers sets inline MCP server definitions.
// The servers are converted to JSON and passed via temp file.
func WithMCPServers(servers map[string]MCPServerConfig) GeminiOption {
	return func(c *GeminiCLI) {
		c.mcpServers = servers
	}
}

// WithSandbox sets the sandbox mode for execution.
// Valid values: "host" (default), "docker", "remote-execution"
func WithSandbox(mode string) GeminiOption {
	return func(c *GeminiCLI) { c.sandbox = mode }
}

// WithIncludeDirs adds directories to be included in the context.
func WithIncludeDirs(dirs []string) GeminiOption {
	return func(c *GeminiCLI) { c.includeDirs = dirs }
}

// MCPServerConfig defines an MCP server for the Gemini CLI.
// Supports stdio, http, and sse transport types.
type MCPServerConfig struct {
	// Type specifies the transport type: "stdio", "http", or "sse".
	// If empty, defaults to "stdio" for servers with Command set.
	Type string `json:"type,omitempty"`

	// Command is the command to run the MCP server (for stdio transport).
	Command string `json:"command,omitempty"`

	// Args are the arguments to pass to the command (for stdio transport).
	Args []string `json:"args,omitempty"`

	// Env provides environment variables for the server process.
	Env map[string]string `json:"env,omitempty"`

	// URL is the server endpoint (for http/sse transport).
	URL string `json:"url,omitempty"`

	// Headers are HTTP headers (for http/sse transport).
	Headers []string `json:"headers,omitempty"`
}

// CLIResponse represents the JSON response from Gemini CLI.
type CLIResponse struct {
	ModelResponse string         `json:"modelResponse"`
	TurnCount     int            `json:"turnCount"`
	Usage         CLIUsage       `json:"usage,omitempty"`
	ExecutedTools []ExecutedTool `json:"executedTools,omitempty"`
}

// CLIUsage contains token usage from the CLI response.
type CLIUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

// ExecutedTool represents a tool that was executed by the CLI.
type ExecutedTool struct {
	Name   string `json:"name"`
	Result string `json:"result,omitempty"`
}

// setupCmd configures the command with working directory and environment variables.
func (c *GeminiCLI) setupCmd(cmd *exec.Cmd) {
	if c.workdir != "" {
		cmd.Dir = c.workdir
	}

	// Only set Env if we have custom environment variables to add.
	// If Env is nil, the command inherits the parent process's environment.
	if len(c.extraEnv) > 0 {
		// Start with parent environment
		cmd.Env = os.Environ()

		// Add any extra environment variables
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
func (c *GeminiCLI) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	args, cleanup := c.buildArgsWithCleanup(req, c.outputFormat)
	defer cleanup()

	cmd := exec.CommandContext(ctx, c.path, args...)
	c.setupCmd(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil // Use /dev/null to prevent TTY/raw mode errors

	if err := cmd.Run(); err != nil {
		// Check for context cancellation first
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
func (c *GeminiCLI) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	// Force stream-json output for streaming
	args, cleanup := c.buildArgsWithCleanup(req, OutputFormatStreamJSON)
	cmd := exec.CommandContext(ctx, c.path, args...)
	c.setupCmd(cmd)
	cmd.Stdin = nil // Use /dev/null to prevent TTY/raw mode errors

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
			// Wait for command to finish and capture error
			cmdErr = cmd.Wait()
			if cmdErr != nil {
				// Try to send error to channel if command failed
				select {
				case ch <- StreamChunk{Error: NewError("stream", fmt.Errorf("command failed: %w", cmdErr), false)}:
				default:
					// Channel closed or full - error already sent
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

			// Try to parse as JSON streaming event
			var event streamEvent
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
			switch event.Type {
			case "text_delta", "content":
				if event.Text != "" {
					accumulatedContent.WriteString(event.Text)
					select {
					case ch <- StreamChunk{Content: event.Text}:
					case <-ctx.Done():
						ch <- StreamChunk{Error: ctx.Err()}
						return
					}
				}
			case "done", "complete":
				select {
				case ch <- StreamChunk{
					Done: true,
					Usage: &TokenUsage{
						InputTokens:  event.Usage.InputTokens,
						OutputTokens: event.Usage.OutputTokens,
						TotalTokens:  event.Usage.InputTokens + event.Usage.OutputTokens,
					},
				}:
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

		// If we didn't get a done event, send final chunk
		select {
		case ch <- StreamChunk{Done: true}:
		default:
		}
	}()

	return ch, nil
}

// buildArgs constructs CLI arguments from a request using the client's configured format.
func (c *GeminiCLI) buildArgs(req CompletionRequest) []string {
	args, _ := c.buildArgsWithCleanup(req, c.outputFormat)
	return args
}

// buildArgsWithCleanup constructs CLI arguments and returns a cleanup function for temp files.
func (c *GeminiCLI) buildArgsWithCleanup(req CompletionRequest, format OutputFormat) ([]string, func()) {
	var tempFiles []string
	var args []string

	// Output format
	args = c.appendOutputArgs(args, format)

	// Model configuration
	args = c.appendModelArgs(args, req)

	// Tool control
	args = c.appendToolArgs(args)

	// MCP configuration (may create temp files)
	args, tempFiles = c.appendMCPArgsWithCleanup(args, tempFiles)

	// Sandbox configuration
	args = c.appendSandboxArgs(args)

	// Include directories
	args = c.appendIncludeDirsArgs(args)

	// Permissions (yolo mode)
	args = c.appendPermissionArgs(args)

	// Build and append the actual prompt from messages
	args = c.appendMessagePrompt(args, req)

	// Return cleanup function
	cleanup := func() {
		for _, f := range tempFiles {
			os.Remove(f)
		}
	}

	return args, cleanup
}

// appendMCPArgsWithCleanup adds MCP configuration arguments and tracks temp files.
func (c *GeminiCLI) appendMCPArgsWithCleanup(args []string, tempFiles []string) ([]string, []string) {
	// Add config file path
	if c.mcpConfigPath != "" {
		args = append(args, "--mcp", c.mcpConfigPath)
	}

	// Add inline servers via temp config file
	if len(c.mcpServers) > 0 && c.mcpConfigPath == "" {
		// Create temp file with MCP config
		tmpFile, err := c.writeMCPConfigFile()
		if err != nil {
			slog.Warn("failed to write MCP config file, skipping MCP servers",
				slog.Any("error", err))
		} else if tmpFile != "" {
			args = append(args, "--mcp", tmpFile)
			tempFiles = append(tempFiles, tmpFile)
		}
	}

	return args, tempFiles
}

// appendOutputArgs adds output format arguments.
func (c *GeminiCLI) appendOutputArgs(args []string, format OutputFormat) []string {
	if format != "" && format != OutputFormatText {
		args = append(args, "--output-format", string(format))
	}
	return args
}

// appendModelArgs adds model and prompt arguments.
func (c *GeminiCLI) appendModelArgs(args []string, req CompletionRequest) []string {
	// Model priority: request > client default
	model := c.model
	if req.Model != "" {
		model = req.Model
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// Max turns
	if c.maxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", c.maxTurns))
	}

	return args
}

// appendToolArgs adds tool control arguments.
func (c *GeminiCLI) appendToolArgs(args []string) []string {
	// Allowed tools (whitelist)
	for _, tool := range c.allowedTools {
		args = append(args, "--allowed-tool", tool)
	}

	// Disallowed tools (blacklist)
	for _, tool := range c.disallowedTools {
		args = append(args, "--disallowed-tool", tool)
	}

	return args
}

// writeMCPConfigFile creates a temporary MCP config file from inline servers.
// Returns the path to the temp file, or empty string on error.
// The caller is responsible for cleanup.
func (c *GeminiCLI) writeMCPConfigFile() (string, error) {
	config := struct {
		MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	}{
		MCPServers: c.mcpServers,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal MCP config: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "gemini-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// appendSandboxArgs adds sandbox configuration arguments.
func (c *GeminiCLI) appendSandboxArgs(args []string) []string {
	if c.sandbox != "" && c.sandbox != "host" {
		args = append(args, "--sandbox", c.sandbox)
	}
	return args
}

// appendIncludeDirsArgs adds include directory arguments.
func (c *GeminiCLI) appendIncludeDirsArgs(args []string) []string {
	for _, dir := range c.includeDirs {
		args = append(args, "--include", dir)
	}
	return args
}

// appendPermissionArgs adds permission-related arguments.
func (c *GeminiCLI) appendPermissionArgs(args []string) []string {
	if c.yolo {
		args = append(args, "--yolo")
	}
	return args
}

// appendMessagePrompt converts messages to a CLI prompt and appends it.
func (c *GeminiCLI) appendMessagePrompt(args []string, req CompletionRequest) []string {
	// System prompt handling - Gemini uses context file or prepends to prompt
	var prompt strings.Builder

	if c.systemPrompt != "" {
		prompt.WriteString(c.systemPrompt)
		prompt.WriteString("\n\n")
	} else if req.SystemPrompt != "" {
		prompt.WriteString(req.SystemPrompt)
		prompt.WriteString("\n\n")
	}

	// Gemini CLI expects a single prompt, so concatenate user messages
	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleUser:
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		case RoleAssistant:
			// For conversation history, format as context
			if prompt.Len() > 0 {
				prompt.WriteString("\nAssistant: ")
				prompt.WriteString(msg.Content)
				prompt.WriteString("\n\nUser: ")
			}
		}
	}

	// Use -p flag for prompt (non-interactive mode)
	promptStr := strings.TrimSpace(prompt.String())
	if promptStr != "" {
		args = append(args, "-p", promptStr)
	}

	return args
}

// parseResponse extracts response data from CLI output.
// Handles both JSON format (rich data) and text format (basic).
func (c *GeminiCLI) parseResponse(data []byte) *CompletionResponse {
	content := strings.TrimSpace(string(data))

	// Try to parse as JSON response
	var cliResp CLIResponse
	if err := json.Unmarshal(data, &cliResp); err == nil && cliResp.ModelResponse != "" {
		return c.parseJSONResponse(&cliResp)
	}

	// JSON parsing failed - log warning if JSON format was expected
	if c.outputFormat == OutputFormatJSON {
		// Truncate output for logging
		preview := content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		slog.Warn("gemini CLI returned non-JSON response when JSON format was expected",
			slog.String("output_format", string(c.outputFormat)),
			slog.String("output_preview", preview),
			slog.String("impact", "token tracking unavailable"))
	}

	// Fall back to raw text response
	return &CompletionResponse{
		Content:      content,
		FinishReason: "stop",
		Model:        c.model,
		Usage: TokenUsage{
			// Token counts not available from basic CLI output
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
	}
}

// parseJSONResponse extracts rich data from a JSON CLI response.
func (c *GeminiCLI) parseJSONResponse(cliResp *CLIResponse) *CompletionResponse {
	resp := &CompletionResponse{
		Content:  cliResp.ModelResponse,
		NumTurns: cliResp.TurnCount,
		Usage: TokenUsage{
			InputTokens:  cliResp.Usage.InputTokens,
			OutputTokens: cliResp.Usage.OutputTokens,
			TotalTokens:  cliResp.Usage.InputTokens + cliResp.Usage.OutputTokens,
		},
	}

	// Set finish reason
	resp.FinishReason = "stop"

	// Get model from client configuration
	resp.Model = c.model

	// Convert executed tools to tool calls
	if len(cliResp.ExecutedTools) > 0 {
		resp.ToolCalls = make([]ToolCall, len(cliResp.ExecutedTools))
		for i, tool := range cliResp.ExecutedTools {
			resp.ToolCalls[i] = ToolCall{
				ID:   fmt.Sprintf("tool_%d", i),
				Name: tool.Name,
			}
		}
	}

	return resp
}

// isRetryableError checks if an error message indicates a transient error.
func isRetryableError(errMsg string) bool {
	errLower := strings.ToLower(errMsg)
	return strings.Contains(errLower, "rate limit") ||
		strings.Contains(errLower, "timeout") ||
		strings.Contains(errLower, "overloaded") ||
		strings.Contains(errLower, "503") ||
		strings.Contains(errLower, "429") ||
		strings.Contains(errLower, "quota")
}

// maxStderrLength limits stderr output in error messages to prevent
// leaking sensitive information and keeping errors readable.
const maxStderrLength = 500

// sanitizeStderr prepares stderr output for inclusion in error messages.
// It truncates long output and redacts common sensitive patterns.
func sanitizeStderr(stderr string) string {
	// Truncate if too long
	if len(stderr) > maxStderrLength {
		stderr = stderr[:maxStderrLength] + "... (truncated)"
	}
	return strings.TrimSpace(stderr)
}

// streamEvent represents a streaming API event from gemini.
type streamEvent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Usage struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
	} `json:"usage,omitempty"`
}

// Provider returns the provider name.
// Implements provider.Client.
func (c *GeminiCLI) Provider() string {
	return "gemini"
}

// Capabilities returns Gemini CLI's native capabilities.
// Implements provider.Client.
func (c *GeminiCLI) Capabilities() Capabilities {
	return Capabilities{
		Streaming: true,
		Tools:     true,
		MCP:       true,
		Sessions:  false, // Gemini CLI doesn't have session support like Claude
		Images:    true,
		NativeTools: []string{
			"read_file",
			"write_file",
			"run_shell_command",
			"web_fetch",
			"google_web_search",
			"save_memory",
			"write_todos",
		},
		ContextFile: "GEMINI.md",
	}
}

// Close releases any resources held by the client.
// For GeminiCLI, this is a no-op as each command is independent.
// Implements provider.Client.
func (c *GeminiCLI) Close() error {
	return nil
}
