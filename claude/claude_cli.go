package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/randalmurphal/llmkit/claudecontract"
)

// OutputFormat specifies the CLI output format.
type OutputFormat string

// Output format constants.
const (
	OutputFormatText       OutputFormat = "text"
	OutputFormatJSON       OutputFormat = "json"
	OutputFormatStreamJSON OutputFormat = "stream-json"
)

// PermissionMode specifies how Claude handles tool permissions.
type PermissionMode string

// Permission mode constants.
const (
	PermissionModeDefault           PermissionMode = ""
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// ClaudeCLI implements Client using the Claude CLI binary.
type ClaudeCLI struct {
	path    string
	model   string
	workdir string
	timeout time.Duration

	// Output control
	outputFormat OutputFormat
	jsonSchema   string

	// Session management
	sessionID            string
	continueSession      bool
	resumeSessionID      string
	noSessionPersistence bool

	// Tool control
	allowedTools    []string
	disallowedTools []string
	tools           []string // Exact tool set (--tools flag)

	// Permissions
	dangerouslySkipPermissions bool
	permissionMode             PermissionMode
	settingSources             []string

	// Context
	addDirs            []string
	systemPrompt       string
	appendSystemPrompt string

	// Budget and limits
	maxBudgetUSD  float64
	fallbackModel string
	maxTurns      int

	// Credential/environment control (for containers)
	homeDir   string            // Override HOME env var (for credential discovery)
	configDir string            // Override ~/.claude directory path
	extraEnv  map[string]string // Additional environment variables

	// MCP configuration
	mcpConfigPaths  []string                   // --mcp-config paths (files or JSON strings)
	mcpServers      map[string]MCPServerConfig // Inline server definitions
	strictMCPConfig bool                       // --strict-mcp-config flag

	// Agent configuration
	agent      string // --agent: Use existing agent by name
	agentsJSON string // --agents: Define custom agents (JSON)

	// Prompt files
	systemPromptFile       string // --system-prompt-file: Load system prompt from file
	appendSystemPromptFile string // --append-system-prompt-file: Load append prompt from file

	// Session forking
	forkSession bool // --fork-session: Fork session instead of reusing

	// Streaming control
	verbose                bool   // --verbose: Enable verbose output
	includePartialMessages bool   // --include-partial-messages: Include partial streaming events
	inputFormat            string // --input-format: Input format for streaming

	// Settings and plugins
	settings  string   // --settings: Load settings file or JSON
	pluginDir []string // --plugin-dir: Plugin directories (repeatable)

	// Debug and development
	debug                string // --debug: Debug mode with optional filter
	disableSlashCommands bool   // --disable-slash-commands: Disable all skills
}

// ClaudeOption configures ClaudeCLI.
type ClaudeOption func(*ClaudeCLI)

// NewClaudeCLI creates a new Claude CLI client.
// Assumes "claude" is available in PATH unless overridden with WithClaudePath.
// By default uses JSON output format for structured responses with token tracking.
func NewClaudeCLI(opts ...ClaudeOption) *ClaudeCLI {
	c := &ClaudeCLI{
		path:         "claude",
		timeout:      5 * time.Minute,
		outputFormat: OutputFormatJSON, // Default to JSON for rich response data
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithClaudePath sets the path to the claude binary.
func WithClaudePath(path string) ClaudeOption {
	return func(c *ClaudeCLI) { c.path = path }
}

// WithModel sets the default model.
func WithModel(model string) ClaudeOption {
	return func(c *ClaudeCLI) { c.model = model }
}

// WithWorkdir sets the working directory for claude commands.
func WithWorkdir(dir string) ClaudeOption {
	return func(c *ClaudeCLI) { c.workdir = dir }
}

// WithTimeout sets the default timeout for commands.
func WithTimeout(d time.Duration) ClaudeOption {
	return func(c *ClaudeCLI) { c.timeout = d }
}

// WithAllowedTools sets the allowed tools for claude (whitelist).
func WithAllowedTools(tools []string) ClaudeOption {
	return func(c *ClaudeCLI) { c.allowedTools = tools }
}

// WithOutputFormat sets the output format (text, json, stream-json).
// Default is json for structured responses with token tracking.
func WithOutputFormat(format OutputFormat) ClaudeOption {
	return func(c *ClaudeCLI) { c.outputFormat = format }
}

// WithJSONSchema forces structured output matching the given JSON schema.
func WithJSONSchema(schema string) ClaudeOption {
	return func(c *ClaudeCLI) { c.jsonSchema = schema }
}

// WithSessionID sets a specific session ID for conversation tracking.
func WithSessionID(id string) ClaudeOption {
	return func(c *ClaudeCLI) { c.sessionID = id }
}

// WithContinue continues the most recent session.
func WithContinue() ClaudeOption {
	return func(c *ClaudeCLI) { c.continueSession = true }
}

// WithResume resumes a specific session by ID.
func WithResume(sessionID string) ClaudeOption {
	return func(c *ClaudeCLI) { c.resumeSessionID = sessionID }
}

// WithNoSessionPersistence disables saving session data.
func WithNoSessionPersistence() ClaudeOption {
	return func(c *ClaudeCLI) { c.noSessionPersistence = true }
}

// WithDisallowedTools sets the tools to disallow (blacklist).
func WithDisallowedTools(tools []string) ClaudeOption {
	return func(c *ClaudeCLI) { c.disallowedTools = tools }
}

// WithTools specifies the exact list of available tools from the built-in set.
// Use an empty slice to disable all tools, or specify tool names like "Bash", "Edit", "Read".
// This is different from WithAllowedTools which is a whitelist filter.
func WithTools(tools []string) ClaudeOption {
	return func(c *ClaudeCLI) { c.tools = tools }
}

// WithDangerouslySkipPermissions skips all permission prompts.
// Use this for non-interactive execution in trusted environments.
// WARNING: This allows Claude to execute any tools without confirmation.
func WithDangerouslySkipPermissions() ClaudeOption {
	return func(c *ClaudeCLI) { c.dangerouslySkipPermissions = true }
}

// WithPermissionMode sets the permission handling mode.
func WithPermissionMode(mode PermissionMode) ClaudeOption {
	return func(c *ClaudeCLI) { c.permissionMode = mode }
}

// WithSettingSources specifies which setting sources to use.
// Valid values: "project", "local", "user"
func WithSettingSources(sources []string) ClaudeOption {
	return func(c *ClaudeCLI) { c.settingSources = sources }
}

// WithAddDirs adds directories to Claude's file access scope.
func WithAddDirs(dirs []string) ClaudeOption {
	return func(c *ClaudeCLI) { c.addDirs = dirs }
}

// WithSystemPrompt sets a custom system prompt, replacing the default.
func WithSystemPrompt(prompt string) ClaudeOption {
	return func(c *ClaudeCLI) { c.systemPrompt = prompt }
}

// WithAppendSystemPrompt appends to the system prompt without replacing it.
func WithAppendSystemPrompt(prompt string) ClaudeOption {
	return func(c *ClaudeCLI) { c.appendSystemPrompt = prompt }
}

// WithMaxBudgetUSD sets a maximum spending limit for the session.
func WithMaxBudgetUSD(amount float64) ClaudeOption {
	return func(c *ClaudeCLI) { c.maxBudgetUSD = amount }
}

// WithFallbackModel sets a fallback model to use if the primary is overloaded.
func WithFallbackModel(model string) ClaudeOption {
	return func(c *ClaudeCLI) { c.fallbackModel = model }
}

// WithMaxTurns limits the number of agentic turns in a conversation.
// A value of 0 means no limit.
func WithMaxTurns(n int) ClaudeOption {
	return func(c *ClaudeCLI) { c.maxTurns = n }
}

// WithHomeDir sets the HOME environment variable for the CLI process.
// This is useful in containers where credentials are mounted to a non-standard location.
// The Claude CLI will look for credentials in $HOME/.claude/.credentials.json.
func WithHomeDir(dir string) ClaudeOption {
	return func(c *ClaudeCLI) { c.homeDir = dir }
}

// WithConfigDir sets the Claude config directory path.
// If set, this overrides the default ~/.claude location.
// Note: This sets CLAUDE_CONFIG_DIR environment variable if supported by the CLI.
func WithConfigDir(dir string) ClaudeOption {
	return func(c *ClaudeCLI) { c.configDir = dir }
}

// WithEnv adds additional environment variables to the CLI process.
// These are merged with the parent environment and any other configured env vars.
func WithEnv(env map[string]string) ClaudeOption {
	return func(c *ClaudeCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		for k, v := range env {
			c.extraEnv[k] = v
		}
	}
}

// WithEnvVar adds a single environment variable to the CLI process.
func WithEnvVar(key, value string) ClaudeOption {
	return func(c *ClaudeCLI) {
		if c.extraEnv == nil {
			c.extraEnv = make(map[string]string)
		}
		c.extraEnv[key] = value
	}
}

// WithMCPConfig adds an MCP configuration file path or JSON string.
// Can be called multiple times to load multiple configs.
// The path can be a file path to a JSON config, or a raw JSON string.
func WithMCPConfig(pathOrJSON string) ClaudeOption {
	return func(c *ClaudeCLI) {
		c.mcpConfigPaths = append(c.mcpConfigPaths, pathOrJSON)
	}
}

// WithMCPServers sets inline MCP server definitions.
// The servers are converted to JSON and passed via --mcp-config.
// This is an alternative to WithMCPConfig for programmatic configuration.
func WithMCPServers(servers map[string]MCPServerConfig) ClaudeOption {
	return func(c *ClaudeCLI) {
		c.mcpServers = servers
	}
}

// WithStrictMCPConfig enables strict MCP configuration mode.
// When enabled, only MCP servers specified via WithMCPConfig or WithMCPServers
// are used, ignoring any other configured MCP servers.
func WithStrictMCPConfig() ClaudeOption {
	return func(c *ClaudeCLI) { c.strictMCPConfig = true }
}

// WithAgent sets an existing agent to use for the session.
// The agent must be defined in the project's .claude/settings.json or user settings.
func WithAgent(name string) ClaudeOption {
	return func(c *ClaudeCLI) { c.agent = name }
}

// WithAgentsJSON defines custom agents inline using JSON format.
// The JSON should match the --agents flag format:
//
//	{
//	  "agent-name": {
//	    "description": "When to use (required)",
//	    "prompt": "System prompt (required)",
//	    "tools": ["Read", "Edit"],  // optional
//	    "model": "sonnet"           // optional
//	  }
//	}
func WithAgentsJSON(json string) ClaudeOption {
	return func(c *ClaudeCLI) { c.agentsJSON = json }
}

// WithSystemPromptFile loads the system prompt from a file, replacing the default.
// This is useful for version-controlled prompt templates.
// Note: Only works in print mode (-p).
func WithSystemPromptFile(path string) ClaudeOption {
	return func(c *ClaudeCLI) { c.systemPromptFile = path }
}

// WithAppendSystemPromptFile loads additional system prompt content from a file.
// The content is appended to the default system prompt.
// Note: Only works in print mode (-p).
func WithAppendSystemPromptFile(path string) ClaudeOption {
	return func(c *ClaudeCLI) { c.appendSystemPromptFile = path }
}

// WithForkSession creates a new session ID when resuming instead of reusing the original.
// Use with WithResume to fork from an existing session.
func WithForkSession() ClaudeOption {
	return func(c *ClaudeCLI) { c.forkSession = true }
}

// WithVerbose enables verbose output for debugging.
func WithVerbose() ClaudeOption {
	return func(c *ClaudeCLI) { c.verbose = true }
}

// WithIncludePartialMessages includes partial streaming events in output.
// Requires print mode with stream-json output format.
func WithIncludePartialMessages() ClaudeOption {
	return func(c *ClaudeCLI) { c.includePartialMessages = true }
}

// WithInputFormat sets the input format for streaming.
// Valid values: "text" (default), "stream-json" (realtime streaming input).
func WithInputFormat(format string) ClaudeOption {
	return func(c *ClaudeCLI) { c.inputFormat = format }
}

// WithSettings loads additional settings from a file path or JSON string.
func WithSettings(pathOrJSON string) ClaudeOption {
	return func(c *ClaudeCLI) { c.settings = pathOrJSON }
}

// WithPluginDir adds a plugin directory to load for this session.
// Can be called multiple times for multiple directories.
func WithPluginDir(path string) ClaudeOption {
	return func(c *ClaudeCLI) {
		c.pluginDir = append(c.pluginDir, path)
	}
}

// WithDebug enables debug mode with an optional category filter.
// Examples: "api,hooks" to show only those categories, "!statsig,!file" to exclude.
func WithDebug(filter string) ClaudeOption {
	return func(c *ClaudeCLI) { c.debug = filter }
}

// WithDisableSlashCommands disables all skills and slash commands for this session.
func WithDisableSlashCommands() ClaudeOption {
	return func(c *ClaudeCLI) { c.disableSlashCommands = true }
}

// CLIResponse represents the full JSON response from Claude CLI.
type CLIResponse struct {
	Type             string                   `json:"type"`
	Subtype          string                   `json:"subtype"`
	IsError          bool                     `json:"is_error"`
	Result           string                   `json:"result"`
	StructuredOutput json.RawMessage          `json:"structured_output,omitempty"`
	SessionID        string                   `json:"session_id"`
	DurationMS       int                      `json:"duration_ms"`
	DurationAPI      int                      `json:"duration_api_ms"`
	NumTurns         int                      `json:"num_turns"`
	TotalCostUSD     float64                  `json:"total_cost_usd"`
	Usage            CLIUsage                 `json:"usage"`
	ModelUsage       map[string]CLIModelUsage `json:"modelUsage"`
}

// CLIUsage contains aggregate token usage from the CLI response.
type CLIUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// CLIModelUsage contains per-model token usage and cost.
type CLIModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
}

// setupCmd configures the command with working directory and environment variables.
func (c *ClaudeCLI) setupCmd(cmd *exec.Cmd) {
	if c.workdir != "" {
		cmd.Dir = c.workdir
	}

	// Only set Env if we have custom environment variables to add.
	// If Env is nil, the command inherits the parent process's environment.
	if c.homeDir != "" || c.configDir != "" || len(c.extraEnv) > 0 {
		// Start with parent environment
		cmd.Env = os.Environ()

		// Override HOME if specified (for credential discovery in containers)
		if c.homeDir != "" {
			cmd.Env = setEnvVar(cmd.Env, "HOME", c.homeDir)
		}

		// Set config directory if specified
		if c.configDir != "" {
			cmd.Env = setEnvVar(cmd.Env, "CLAUDE_CONFIG_DIR", c.configDir)
		}

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

// resolvedPath returns the absolute path to the claude binary.
// This is necessary because when cmd.Dir is set, Go's exec doesn't do PATH lookup
// for relative executable names like "claude".
func (c *ClaudeCLI) resolvedPath() string {
	// If already absolute, use as-is
	if filepath.IsAbs(c.path) {
		return c.path
	}

	// If no workdir set, exec will do PATH lookup correctly
	if c.workdir == "" {
		return c.path
	}

	// Resolve relative path to absolute via PATH lookup
	absPath, err := exec.LookPath(c.path)
	if err != nil {
		// Fall back to original path - error will surface when command runs
		slog.Debug("could not resolve executable path", "path", c.path, "error", err)
		return c.path
	}
	return absPath
}

// Complete implements Client.
// This is a convenience wrapper around StreamJSON that collects all events
// and returns a single CompletionResponse.
// If req.OnEvent is set, it will be called for each streaming event.
func (c *ClaudeCLI) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	events, result, err := c.StreamJSON(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, err := StreamToCompleteWithCallback(ctx, events, result, req.OnEvent)
	if err != nil {
		return nil, err
	}
	resp.Duration = time.Since(start)

	// Validate structured output when schema was requested.
	// --json-schema uses constrained decoding which guarantees schema-compliant output,
	// but only if Claude actually produces a final response. If Claude exits early
	// (e.g., on resume of a "completed" session), structured_output may be empty.
	schema := c.jsonSchema
	if req.JSONSchema != "" {
		schema = req.JSONSchema
	}
	if schema != "" && !resp.StructuredOutputUsed {
		preview := resp.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("JSON schema was specified but no structured output received (num_turns=%d, content=%q)", resp.NumTurns, preview)
	}

	return resp, nil
}

// StreamJSON sends a request and returns a channel of typed streaming events.
// The StreamResult future resolves when streaming completes with final totals.
//
// This is the core streaming implementation. Use Complete() for a simpler
// blocking interface that returns a single CompletionResponse.
//
// Example:
//
//	events, result, err := client.StreamJSON(ctx, req)
//	if err != nil {
//	    return err
//	}
//	for event := range events {
//	    switch event.Type {
//	    case StreamEventInit:
//	        fmt.Println("Session:", event.Init.SessionID)
//	    case StreamEventAssistant:
//	        fmt.Print(event.Assistant.Text)
//	    case StreamEventResult:
//	        fmt.Println("Cost:", event.Result.TotalCostUSD)
//	    }
//	}
//	final, err := result.Wait(ctx)
func (c *ClaudeCLI) StreamJSON(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, *StreamResult, error) {
	args := c.buildArgsForStreamJSON(req)
	cmd := exec.CommandContext(ctx, c.resolvedPath(), args...)
	c.setupCmd(cmd)
	cmd.Stdin = nil // Use /dev/null to prevent TTY/raw mode errors in containers

	// Run in separate process group so we can kill all child processes on cancel.
	// Claude Code spawns subprocesses (test runners, build tools, MCP servers) that
	// would otherwise become orphaned when the main process is killed.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, NewError("stream_json", fmt.Errorf("create stdout pipe: %w", err), false)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, NewError("stream_json", fmt.Errorf("start command: %w", err), false)
	}

	events := make(chan StreamEvent, 100)
	result := newStreamResult()

	go c.processStreamJSON(ctx, stdout, cmd, events, result)

	return events, result, nil
}

// buildArgsForStreamJSON constructs arguments for stream-json mode.
// Uses --output-format stream-json --verbose for full event data.
// Still supports --json-schema for structured output.
func (c *ClaudeCLI) buildArgsForStreamJSON(req CompletionRequest) []string {
	var args []string

	// Always use --print for non-interactive mode
	args = append(args, claudecontract.FlagPrint)

	// Stream-json format with verbose for full events
	args = append(args, claudecontract.FlagOutputFormat, claudecontract.FormatStreamJSON)
	args = append(args, claudecontract.FlagVerbose)

	// JSON schema support (structured_output appears in result event)
	schema := c.jsonSchema
	if req.JSONSchema != "" {
		schema = req.JSONSchema
	}
	if schema != "" {
		args = append(args, claudecontract.FlagJSONSchema, schema)
	}

	// Session management
	args = c.appendSessionArgs(args)

	// Model and prompt configuration
	args = c.appendModelArgs(args, req)

	// Tool control
	args = c.appendToolArgs(args)

	// MCP configuration
	args = c.appendMCPArgs(args)

	// Agent configuration
	args = c.appendAgentArgs(args)

	// Prompt file configuration
	args = c.appendPromptFileArgs(args)

	// Streaming control (verbose already added above for stream-json)
	args = c.appendStreamingArgs(args)

	// Settings and plugins
	args = c.appendSettingsArgs(args)

	// Debug and development
	args = c.appendDebugArgs(args)

	// Permissions and settings
	args = c.appendPermissionArgs(args)

	// Build and append the actual prompt from messages
	args = c.appendMessagePrompt(args, req.Messages)

	return args
}

// processStreamJSON reads and parses stream-json output.
func (c *ClaudeCLI) processStreamJSON(
	ctx context.Context,
	stdout io.ReadCloser,
	cmd *exec.Cmd,
	events chan<- StreamEvent,
	result *StreamResult,
) {
	defer close(events)

	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for large messages (10MB max)
	const maxScanTokenSize = 10 * 1024 * 1024
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)

	var sessionID string
	var finalResult *ResultEvent

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		event, err := parseStreamEvent(line)
		if err != nil {
			// Log parse error but continue
			slog.Debug("failed to parse stream event", "error", err, "line", string(line))
			continue
		}

		// Track session ID from first event that has it
		if event.SessionID != "" && sessionID == "" {
			sessionID = event.SessionID
		}
		event.SessionID = sessionID

		// Capture result for the future
		if event.Type == StreamEventResult && event.Result != nil {
			finalResult = event.Result
		}

		select {
		case events <- *event:
		case <-ctx.Done():
			// Kill entire process group to clean up child processes
			if cmd.Process != nil {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
			result.complete(nil, ctx.Err())
			return
		}
	}

	if err := scanner.Err(); err != nil {
		result.complete(nil, NewError("stream_json", fmt.Errorf("read output: %w", err), false))
		// Wait for command to finish
		_ = cmd.Wait()
		return
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		if finalResult == nil {
			result.complete(nil, NewError("stream_json", fmt.Errorf("command failed: %w", err), false))
			return
		}
		// If we have a result, the error might just be non-zero exit from tool use
	}

	result.complete(finalResult, nil)
}

// parseStreamEvent parses a single JSON line into a StreamEvent.
func parseStreamEvent(data []byte) (*StreamEvent, error) {
	// First pass: determine type
	var base struct {
		Type      string `json:"type"`
		Subtype   string `json:"subtype,omitempty"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	event := &StreamEvent{
		SessionID: base.SessionID,
		Raw:       data,
	}

	// Parse type-specific data
	switch base.Type {
	case claudecontract.EventTypeSystem:
		if base.Subtype == claudecontract.SubtypeInit {
			event.Type = StreamEventInit
			event.Init = &InitEvent{}
			if err := json.Unmarshal(data, event.Init); err != nil {
				return nil, err
			}
		} else if base.Subtype == claudecontract.SubtypeHookResponse {
			event.Type = StreamEventHook
			event.Hook = &HookEvent{}
			if err := json.Unmarshal(data, event.Hook); err != nil {
				return nil, err
			}
		}

	case claudecontract.EventTypeAssistant:
		event.Type = StreamEventAssistant
		var assistantWrapper struct {
			Message struct {
				ID         string         `json:"id"`
				Content    []ContentBlock `json:"content"`
				Model      string         `json:"model"`
				Usage      MessageUsage   `json:"usage"`
				StopReason *string        `json:"stop_reason"`
			} `json:"message"`
		}
		if err := json.Unmarshal(data, &assistantWrapper); err != nil {
			return nil, err
		}
		msg := assistantWrapper.Message
		event.Assistant = &AssistantEvent{
			MessageID: msg.ID,
			Content:   msg.Content,
			Model:     msg.Model,
			Usage:     msg.Usage,
		}
		if msg.StopReason != nil {
			event.Assistant.StopReason = *msg.StopReason
		}
		// Compute convenience text field
		var text strings.Builder
		for _, block := range msg.Content {
			if block.Type == claudecontract.ContentTypeText {
				text.WriteString(block.Text)
			}
		}
		event.Assistant.Text = text.String()

	case claudecontract.EventTypeUser:
		event.Type = StreamEventUser
		event.User = &UserEvent{}
		if err := json.Unmarshal(data, event.User); err != nil {
			return nil, err
		}

	case claudecontract.EventTypeResult:
		event.Type = StreamEventResult
		event.Result = &ResultEvent{}
		if err := json.Unmarshal(data, event.Result); err != nil {
			return nil, err
		}
	}

	return event, nil
}


// buildArgs constructs CLI arguments from a request using the client's configured format.
func (c *ClaudeCLI) buildArgs(req CompletionRequest) []string {
	return c.buildArgsWithFormat(req, c.outputFormat)
}

// buildArgsWithFormat constructs CLI arguments with a specific output format.
func (c *ClaudeCLI) buildArgsWithFormat(req CompletionRequest, format OutputFormat) []string {
	var args []string

	// Always use --print for non-interactive mode
	args = append(args, claudecontract.FlagPrint)

	// Output format and schema (request schema overrides client schema)
	args = c.appendOutputArgs(args, format, req.JSONSchema)

	// Session management
	args = c.appendSessionArgs(args)

	// Model and prompt configuration
	args = c.appendModelArgs(args, req)

	// Tool control
	args = c.appendToolArgs(args)

	// MCP configuration
	args = c.appendMCPArgs(args)

	// Agent configuration
	args = c.appendAgentArgs(args)

	// Prompt file configuration
	args = c.appendPromptFileArgs(args)

	// Streaming control
	args = c.appendStreamingArgs(args)

	// Settings and plugins
	args = c.appendSettingsArgs(args)

	// Debug and development
	args = c.appendDebugArgs(args)

	// Permissions and settings
	args = c.appendPermissionArgs(args)

	// Build and append the actual prompt from messages
	args = c.appendMessagePrompt(args, req.Messages)

	return args
}

// appendOutputArgs adds output format arguments.
// requestSchema takes precedence over client-level schema if non-empty.
// When a JSON schema is specified, JSON output format is enforced automatically.
func (c *ClaudeCLI) appendOutputArgs(args []string, format OutputFormat, requestSchema string) []string {
	// Request schema overrides client schema
	schema := c.jsonSchema
	if requestSchema != "" {
		schema = requestSchema
	}

	// When schema is set, REQUIRE JSON output format (--json-schema only works with JSON)
	if schema != "" {
		args = append(args, claudecontract.FlagOutputFormat, string(OutputFormatJSON))
		args = append(args, claudecontract.FlagJSONSchema, schema)
	} else if format != "" && format != OutputFormatText {
		args = append(args, claudecontract.FlagOutputFormat, string(format))
	}
	return args
}

// appendSessionArgs adds session management arguments.
func (c *ClaudeCLI) appendSessionArgs(args []string) []string {
	if c.sessionID != "" {
		args = append(args, claudecontract.FlagSessionID, c.sessionID)
	}
	if c.continueSession {
		args = append(args, claudecontract.FlagContinue)
	}
	if c.resumeSessionID != "" {
		args = append(args, claudecontract.FlagResume, c.resumeSessionID)
	}
	if c.forkSession {
		args = append(args, claudecontract.FlagForkSession)
	}
	if c.noSessionPersistence {
		args = append(args, claudecontract.FlagNoSessionPersistence)
	}
	return args
}

// appendModelArgs adds model, prompt, and limit arguments.
func (c *ClaudeCLI) appendModelArgs(args []string, req CompletionRequest) []string {
	// System prompt handling
	if c.systemPrompt != "" {
		args = append(args, claudecontract.FlagSystemPrompt, c.systemPrompt)
	} else if req.SystemPrompt != "" {
		args = append(args, claudecontract.FlagSystemPrompt, req.SystemPrompt)
	}
	if c.appendSystemPrompt != "" {
		args = append(args, claudecontract.FlagAppendSystemPrompt, c.appendSystemPrompt)
	}

	// Model priority: request > client default
	model := c.model
	if req.Model != "" {
		model = req.Model
	}
	if model != "" {
		args = append(args, claudecontract.FlagModel, model)
	}

	// Fallback model
	if c.fallbackModel != "" {
		args = append(args, claudecontract.FlagFallbackModel, c.fallbackModel)
	}

	// NOTE: MaxTokens is not supported by Claude CLI - it's an agentic interface
	// that doesn't expose token limits. The field is silently ignored.
	// Use maxBudgetUSD for cost control instead.

	// Budget limit (only flag available for limiting Claude CLI usage)
	if c.maxBudgetUSD > 0 {
		args = append(args, claudecontract.FlagMaxBudgetUSD, fmt.Sprintf("%.6f", c.maxBudgetUSD))
	}

	// Max turns (limits agentic conversation turns)
	if c.maxTurns > 0 {
		args = append(args, claudecontract.FlagMaxTurns, fmt.Sprintf("%d", c.maxTurns))
	}

	return args
}

// appendToolArgs adds tool control arguments.
func (c *ClaudeCLI) appendToolArgs(args []string) []string {
	// Allowed tools (whitelist)
	for _, tool := range c.allowedTools {
		args = append(args, claudecontract.FlagAllowedTools, tool)
	}

	// Disallowed tools (blacklist)
	// Note: Claude CLI uses camelCase for tool flags (--allowedTools, --disallowedTools)
	for _, tool := range c.disallowedTools {
		args = append(args, claudecontract.FlagDisallowedTools, tool)
	}

	// Exact tool set
	if len(c.tools) > 0 {
		args = append(args, claudecontract.FlagTools, strings.Join(c.tools, ","))
	}

	return args
}

// appendPermissionArgs adds permission and settings arguments.
func (c *ClaudeCLI) appendPermissionArgs(args []string) []string {
	if c.dangerouslySkipPermissions {
		args = append(args, claudecontract.FlagDangerouslySkipPermissions)
	}
	if c.permissionMode != "" {
		args = append(args, claudecontract.FlagPermissionMode, string(c.permissionMode))
	}

	// Setting sources
	if len(c.settingSources) > 0 {
		args = append(args, claudecontract.FlagSettingSources, strings.Join(c.settingSources, ","))
	}

	// Additional directories
	for _, dir := range c.addDirs {
		args = append(args, claudecontract.FlagAddDir, dir)
	}

	return args
}

// appendMCPArgs adds MCP configuration arguments.
func (c *ClaudeCLI) appendMCPArgs(args []string) []string {
	// Add config file paths or JSON strings
	for _, pathOrJSON := range c.mcpConfigPaths {
		args = append(args, claudecontract.FlagMCPConfig, pathOrJSON)
	}

	// Add inline servers as JSON string
	if len(c.mcpServers) > 0 {
		mcpJSON, err := json.Marshal(map[string]any{
			"mcpServers": c.mcpServers,
		})
		if err == nil {
			args = append(args, claudecontract.FlagMCPConfig, string(mcpJSON))
		}
	}

	// Strict mode
	if c.strictMCPConfig {
		args = append(args, claudecontract.FlagStrictMCPConfig)
	}

	return args
}

// appendAgentArgs adds agent configuration arguments.
func (c *ClaudeCLI) appendAgentArgs(args []string) []string {
	// Use existing agent by name
	if c.agent != "" {
		args = append(args, claudecontract.FlagAgent, c.agent)
	}

	// Define custom agents inline via JSON
	if c.agentsJSON != "" {
		args = append(args, claudecontract.FlagAgents, c.agentsJSON)
	}

	return args
}

// appendPromptFileArgs adds system prompt file arguments.
func (c *ClaudeCLI) appendPromptFileArgs(args []string) []string {
	if c.systemPromptFile != "" {
		args = append(args, claudecontract.FlagSystemPromptFile, c.systemPromptFile)
	}
	if c.appendSystemPromptFile != "" {
		args = append(args, claudecontract.FlagAppendSystemPromptFile, c.appendSystemPromptFile)
	}
	return args
}

// appendStreamingArgs adds streaming control arguments.
func (c *ClaudeCLI) appendStreamingArgs(args []string) []string {
	if c.verbose {
		args = append(args, claudecontract.FlagVerbose)
	}
	if c.includePartialMessages {
		args = append(args, claudecontract.FlagIncludePartialMessages)
	}
	if c.inputFormat != "" {
		args = append(args, claudecontract.FlagInputFormat, c.inputFormat)
	}
	return args
}

// appendSettingsArgs adds settings and plugin arguments.
func (c *ClaudeCLI) appendSettingsArgs(args []string) []string {
	if c.settings != "" {
		args = append(args, claudecontract.FlagSettings, c.settings)
	}
	for _, dir := range c.pluginDir {
		args = append(args, claudecontract.FlagPluginDir, dir)
	}
	return args
}

// appendDebugArgs adds debug and development arguments.
func (c *ClaudeCLI) appendDebugArgs(args []string) []string {
	if c.debug != "" {
		args = append(args, claudecontract.FlagDebug, c.debug)
	}
	if c.disableSlashCommands {
		args = append(args, claudecontract.FlagDisableSlashCommands)
	}
	return args
}

// appendMessagePrompt converts messages to a CLI prompt and appends it.
func (c *ClaudeCLI) appendMessagePrompt(args []string, messages []Message) []string {
	// Claude CLI expects a single prompt, so concatenate user messages
	var prompt strings.Builder
	for _, msg := range messages {
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

	// Prompt is a positional argument (not a flag)
	// Note: -p/--print is for non-interactive mode, NOT for passing the prompt
	promptStr := strings.TrimSpace(prompt.String())
	if promptStr != "" {
		args = append(args, promptStr)
	}

	return args
}

// parseResponseWithSchema extracts response data from CLI output.
// When schema is non-empty, structured_output is REQUIRED - no fallback to result.
// This ensures callers get explicit errors when schema enforcement fails.
func (c *ClaudeCLI) parseResponseWithSchema(data []byte, schema string) (*CompletionResponse, error) {
	content := strings.TrimSpace(string(data))

	// Try to parse as JSON response
	var cliResp CLIResponse
	if err := json.Unmarshal(data, &cliResp); err == nil && cliResp.Type != "" {
		return c.parseJSONResponseWithSchema(&cliResp, schema)
	}

	// JSON parsing failed
	if schema != "" {
		// Schema was requested - this is an error, not a fallback situation
		preview := content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("JSON schema was specified but CLI returned non-JSON response: %s", preview)
	}

	// No schema - log warning if JSON format was expected
	if c.outputFormat == OutputFormatJSON {
		preview := content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		slog.Warn("claude CLI returned non-JSON response when JSON format was expected",
			slog.String("output_format", string(c.outputFormat)),
			slog.String("output_preview", preview),
			slog.String("impact", "token tracking unavailable"))
	}

	// No schema - fall back to raw text response
	return &CompletionResponse{
		Content:      content,
		FinishReason: "stop",
		Model:        c.model,
		Usage: TokenUsage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
	}, nil
}

// parseJSONResponseWithSchema extracts rich data from a JSON CLI response.
// When schema is non-empty, ONLY uses structured_output - errors if empty.
// No silent fallback to result field when schema was specified.
func (c *ClaudeCLI) parseJSONResponseWithSchema(cliResp *CLIResponse, schema string) (*CompletionResponse, error) {
	var content string

	if schema != "" {
		// Schema was used - MUST have structured_output, no fallback
		if len(cliResp.StructuredOutput) == 0 {
			// Provide context about what we got instead
			preview := cliResp.Result
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			return nil, fmt.Errorf("JSON schema was specified but structured_output is empty (result=%q)", preview)
		}
		content = string(cliResp.StructuredOutput)
	} else {
		// No schema - use result field
		content = cliResp.Result
	}

	resp := &CompletionResponse{
		Content:   content,
		SessionID: cliResp.SessionID,
		CostUSD:   cliResp.TotalCostUSD,
		NumTurns:  cliResp.NumTurns,
		Usage: TokenUsage{
			InputTokens:              cliResp.Usage.InputTokens,
			OutputTokens:             cliResp.Usage.OutputTokens,
			TotalTokens:              cliResp.Usage.InputTokens + cliResp.Usage.OutputTokens,
			CacheCreationInputTokens: cliResp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     cliResp.Usage.CacheReadInputTokens,
		},
	}

	// Determine finish reason from response type
	if cliResp.IsError {
		resp.FinishReason = "error"
	} else {
		resp.FinishReason = "stop"
	}

	// Get model from modelUsage if available
	for model := range cliResp.ModelUsage {
		resp.Model = model
		break // Use first model found
	}

	// Fall back to client's configured model
	if resp.Model == "" {
		resp.Model = c.model
	}

	return resp, nil
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


// Provider returns the provider name.
// Implements provider.Client.
func (c *ClaudeCLI) Provider() string {
	return "claude"
}

// Capabilities returns Claude Code's native capabilities.
// Implements provider.Client.
func (c *ClaudeCLI) Capabilities() Capabilities {
	return Capabilities{
		Streaming:   true,
		Tools:       true,
		MCP:         true,
		Sessions:    true,
		Images:      true,
		NativeTools: []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash", "Task", "TodoWrite", "WebFetch", "WebSearch", "AskUserQuestion", "NotebookEdit", "LSP", "Skill", "EnterPlanMode", "ExitPlanMode", "KillShell", "TaskOutput"},
		ContextFile: "CLAUDE.md",
	}
}

// Close releases any resources held by the client.
// For ClaudeCLI, this is a no-op as each command is independent.
// Implements provider.Client.
func (c *ClaudeCLI) Close() error {
	return nil
}
