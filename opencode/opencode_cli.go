package opencode

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
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
)

// Agent specifies the OpenCode agent mode.
type Agent string

// Agent constants.
const (
	AgentBuild Agent = "build" // Full access for development (default)
	AgentPlan  Agent = "plan"  // Read-only for analysis
)

// OpenCodeCLI implements Client using the OpenCode CLI binary.
type OpenCodeCLI struct {
	path    string
	workdir string
	timeout time.Duration

	// Output control
	outputFormat OutputFormat
	quiet        bool
	debug        bool

	// Agent selection
	agent Agent

	// Prompt configuration
	systemPrompt string

	// Tool control
	allowedTools    []string
	disallowedTools []string

	// Budget and limits
	maxTurns int

	// Environment control
	extraEnv map[string]string

	// MCP configuration
	mcpConfigPath string                     // --mcp-config path
	mcpServers    map[string]MCPServerConfig // Inline server definitions
}

// MCPServerConfig defines an MCP server for the OpenCode CLI.
// Supports stdio, http, and sse transport types.
type MCPServerConfig struct {
	// Type specifies the transport type: "stdio", "http", or "sse".
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

// OpenCodeOption configures OpenCodeCLI.
type OpenCodeOption func(*OpenCodeCLI)

// NewOpenCodeCLI creates a new OpenCode CLI client.
// Assumes "opencode" is available in PATH unless overridden with WithOpenCodePath.
// By default uses JSON output format for structured responses.
func NewOpenCodeCLI(opts ...OpenCodeOption) *OpenCodeCLI {
	c := &OpenCodeCLI{
		path:         "opencode",
		timeout:      5 * time.Minute,
		outputFormat: OutputFormatJSON,
		agent:        AgentBuild, // Default to build agent
		quiet:        true,       // Default to quiet for clean output
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithOpenCodePath sets the path to the opencode binary.
func WithOpenCodePath(path string) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.path = path }
}

// WithWorkdir sets the working directory for opencode commands.
func WithWorkdir(dir string) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.workdir = dir }
}

// WithTimeout sets the default timeout for commands.
func WithTimeout(d time.Duration) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.timeout = d }
}

// WithOutputFormat sets the output format (text, json).
// Default is json for structured responses.
func WithOutputFormat(format OutputFormat) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.outputFormat = format }
}

// WithQuiet enables or disables quiet mode.
// When enabled (default), suppresses extra output for cleaner responses.
func WithQuiet(quiet bool) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.quiet = quiet }
}

// WithDebug enables debug mode for verbose output.
func WithDebug(debug bool) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.debug = debug }
}

// WithAgent sets the agent mode (build or plan).
// Build agent has full access, plan agent is read-only.
func WithAgent(agent Agent) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.agent = agent }
}

// WithSystemPrompt sets a custom system prompt.
func WithSystemPrompt(prompt string) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.systemPrompt = prompt }
}

// WithAllowedTools sets the allowed tools for opencode (whitelist).
func WithAllowedTools(tools []string) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.allowedTools = tools }
}

// WithDisallowedTools sets the tools to disallow (blacklist).
func WithDisallowedTools(tools []string) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.disallowedTools = tools }
}

// WithMaxTurns limits the number of agentic turns in a conversation.
// A value of 0 means no limit.
func WithMaxTurns(n int) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.maxTurns = n }
}

// WithEnv adds additional environment variables to the CLI process.
func WithEnv(env map[string]string) OpenCodeOption {
	return func(c *OpenCodeCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		for k, v := range env {
			c.extraEnv[k] = v
		}
	}
}

// WithEnvVar adds a single environment variable to the CLI process.
func WithEnvVar(key, value string) OpenCodeOption {
	return func(c *OpenCodeCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		c.extraEnv[key] = value
	}
}

// WithMCPConfig sets the path to an MCP configuration file.
func WithMCPConfig(path string) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.mcpConfigPath = path }
}

// WithMCPServers sets inline MCP server definitions.
// The servers are converted to JSON and passed via temp file.
func WithMCPServers(servers map[string]MCPServerConfig) OpenCodeOption {
	return func(c *OpenCodeCLI) { c.mcpServers = servers }
}

// CLIResponse represents the JSON response from OpenCode CLI.
type CLIResponse struct {
	Result    string        `json:"result"`
	Error     string        `json:"error,omitempty"`
	IsError   bool          `json:"is_error,omitempty"`
	Usage     *CLIUsage     `json:"usage,omitempty"`
	Model     string        `json:"model,omitempty"`
	Duration  int           `json:"duration_ms,omitempty"`
	ToolCalls []CLIToolCall `json:"tool_calls,omitempty"`
}

// CLIUsage contains token usage from the CLI response.
type CLIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// CLIToolCall represents a tool call in the response.
type CLIToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// setupCmd configures the command with working directory and environment variables.
func (c *OpenCodeCLI) setupCmd(cmd *exec.Cmd) {
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
func (c *OpenCodeCLI) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	args, cleanup := c.buildArgsWithCleanup(req)
	defer cleanup()

	cmd := exec.CommandContext(ctx, c.path, args...)
	c.setupCmd(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil // Use /dev/null to prevent TTY issues

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
func (c *OpenCodeCLI) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
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

			// Try to parse as JSON streaming event
			var event streamEvent
			if err := json.Unmarshal([]byte(line), &event); err == nil && event.Type != "" {
				switch event.Type {
				case "content":
					accumulatedContent.WriteString(event.Content)
					select {
					case ch <- StreamChunk{Content: event.Content}:
					case <-ctx.Done():
						ch <- StreamChunk{Error: ctx.Err()}
						return
					}
				case "done":
					var usage *TokenUsage
					if event.Usage != nil {
						usage = &TokenUsage{
							InputTokens:  event.Usage.InputTokens,
							OutputTokens: event.Usage.OutputTokens,
							TotalTokens:  event.Usage.InputTokens + event.Usage.OutputTokens,
						}
					}
					select {
					case ch <- StreamChunk{Done: true, Usage: usage}:
					case <-ctx.Done():
						ch <- StreamChunk{Error: ctx.Err()}
						return
					}
				}
				continue
			}

			// Not JSON, treat as raw text
			accumulatedContent.WriteString(line)
			accumulatedContent.WriteString("\n")
			select {
			case ch <- StreamChunk{Content: line + "\n"}:
			case <-ctx.Done():
				ch <- StreamChunk{Error: ctx.Err()}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: NewError("stream", fmt.Errorf("read output: %w", err), false)}
			return
		}

		// Send final chunk if no done event received
		select {
		case ch <- StreamChunk{Done: true}:
		default:
		}
	}()

	return ch, nil
}

// buildArgsWithCleanup constructs CLI arguments and returns a cleanup function for temp files.
func (c *OpenCodeCLI) buildArgsWithCleanup(req CompletionRequest) ([]string, func()) {
	var args []string
	var tempFiles []string

	// Output format - OpenCode uses --format flag
	if c.outputFormat == OutputFormatJSON {
		args = append(args, "--format", "json")
	}

	// Quiet mode
	if c.quiet {
		args = append(args, "-q")
	}

	// Debug mode
	if c.debug {
		args = append(args, "-d")
	}

	// Model selection - passed via --model flag
	model := req.Model
	if model == "" {
		// Check options for model override
		if m, ok := req.Options["model"].(string); ok && m != "" {
			model = m
		}
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// Agent selection - OpenCode uses --agent flag
	if c.agent != "" {
		args = append(args, "--agent", string(c.agent))
	}

	// MCP configuration
	if c.mcpConfigPath != "" {
		args = append(args, "--mcp-config", c.mcpConfigPath)
	} else if len(c.mcpServers) > 0 {
		// Create temp file with MCP config
		tmpFile, err := c.writeMCPConfigFile()
		if err != nil {
			slog.Warn("failed to write MCP config file, skipping MCP servers",
				slog.Any("error", err))
		} else if tmpFile != "" {
			args = append(args, "--mcp-config", tmpFile)
			tempFiles = append(tempFiles, tmpFile)
		}
	}

	// Build the prompt from messages
	var prompt strings.Builder
	if req.SystemPrompt != "" || c.systemPrompt != "" {
		sp := req.SystemPrompt
		if sp == "" {
			sp = c.systemPrompt
		}
		prompt.WriteString(sp)
		prompt.WriteString("\n\n")
	}

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

	// Use -p flag for prompt (non-interactive mode)
	promptStr := strings.TrimSpace(prompt.String())
	if promptStr != "" {
		args = append(args, "-p", promptStr)
	}

	// Return cleanup function
	cleanup := func() {
		for _, f := range tempFiles {
			_ = os.Remove(f)
		}
	}

	return args, cleanup
}

// writeMCPConfigFile creates a temporary MCP config file from inline servers.
// Returns the path to the temp file, or empty string on error.
func (c *OpenCodeCLI) writeMCPConfigFile() (string, error) {
	config := struct {
		MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	}{
		MCPServers: c.mcpServers,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal MCP config: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "opencode-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// parseResponse extracts response data from CLI output.
func (c *OpenCodeCLI) parseResponse(data []byte) *CompletionResponse {
	content := strings.TrimSpace(string(data))

	// Try to parse as JSON response
	var cliResp CLIResponse
	if err := json.Unmarshal(data, &cliResp); err == nil && (cliResp.Result != "" || cliResp.Error != "") {
		return c.parseJSONResponse(&cliResp)
	}

	// JSON parsing failed - log warning if JSON format was expected
	if c.outputFormat == OutputFormatJSON && content != "" {
		preview := content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		slog.Warn("opencode CLI returned non-JSON response when JSON format was expected",
			slog.String("output_format", string(c.outputFormat)),
			slog.String("output_preview", preview),
			slog.String("impact", "token tracking unavailable"))
	}

	// Fall back to raw text response
	return &CompletionResponse{
		Content:      content,
		FinishReason: "stop",
		Usage: TokenUsage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
	}
}

// parseJSONResponse extracts rich data from a JSON CLI response.
func (c *OpenCodeCLI) parseJSONResponse(cliResp *CLIResponse) *CompletionResponse {
	resp := &CompletionResponse{
		Content: cliResp.Result,
		Model:   cliResp.Model,
	}

	if cliResp.Usage != nil {
		resp.Usage = TokenUsage{
			InputTokens:  cliResp.Usage.InputTokens,
			OutputTokens: cliResp.Usage.OutputTokens,
			TotalTokens:  cliResp.Usage.InputTokens + cliResp.Usage.OutputTokens,
		}
	}

	// Convert tool calls
	if len(cliResp.ToolCalls) > 0 {
		resp.ToolCalls = make([]ToolCall, len(cliResp.ToolCalls))
		for i, tc := range cliResp.ToolCalls {
			resp.ToolCalls[i] = ToolCall(tc)
		}
	}

	// Determine finish reason
	if cliResp.IsError || cliResp.Error != "" {
		resp.FinishReason = "error"
	} else {
		resp.FinishReason = "stop"
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
		strings.Contains(errLower, "529")
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

// streamEvent represents a streaming event from OpenCode.
type streamEvent struct {
	Type    string    `json:"type"`
	Content string    `json:"content,omitempty"`
	Usage   *CLIUsage `json:"usage,omitempty"`
}

// Provider returns the provider name.
func (c *OpenCodeCLI) Provider() string {
	return "opencode"
}

// Capabilities returns OpenCode's native capabilities.
func (c *OpenCodeCLI) Capabilities() Capabilities {
	return Capabilities{
		Streaming: true,
		Tools:     true,
		MCP:       true,
		Sessions:  false, // OpenCode doesn't have built-in session management
		Images:    false, // OpenCode doesn't support image inputs
		// Native tools based on official documentation
		NativeTools: []string{
			"glob",        // Find files by pattern
			"grep",        // Search file contents
			"ls",          // List directory contents
			"view",        // View file contents
			"write",       // Write to files
			"edit",        // Edit files
			"patch",       // Apply patches to files
			"diagnostics", // Get diagnostics information
			"bash",        // Execute shell commands
			"fetch",       // Retrieve data from URLs
			"sourcegraph", // Search code across public repositories
			"agent",       // Run sub-tasks with the AI agent
		},
		ContextFile: "", // No specific context file convention
	}
}

// Close releases any resources held by the client.
// For OpenCodeCLI, this is a no-op as each command is independent.
func (c *OpenCodeCLI) Close() error {
	return nil
}
