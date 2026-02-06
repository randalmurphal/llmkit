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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/randalmurphal/llmkit/codexcontract"
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

// WebSearchMode controls how Codex should perform web search.
type WebSearchMode string

// Web search mode constants.
const (
	WebSearchCached   WebSearchMode = "cached"
	WebSearchLive     WebSearchMode = "live"
	WebSearchDisabled WebSearchMode = "disabled"
)

// CodexCLI implements Client using the Codex CLI binary.
type CodexCLI struct {
	path    string
	model   string
	workdir string
	timeout time.Duration

	// Safety/sandbox behavior
	sandboxMode                          SandboxMode
	approvalMode                         ApprovalMode
	fullAuto                             bool
	dangerouslyBypassApprovalsAndSandbox bool

	// Session management
	sessionID string
	resumeAll bool

	// Search
	enableSearch  bool
	webSearchMode WebSearchMode

	// Additional directories
	addDirs []string

	// Images
	images []string

	// Headless CLI controls
	profile               string
	localProvider         string
	configOverrides       map[string]any
	skipGitRepoCheck      bool
	outputSchemaPath      string
	outputLastMessagePath string
	reasoningEffort       string
	hideAgentReasoning    bool
	useOSS                bool
	enabledFeatures       []string
	disabledFeatures      []string
	colorMode             string

	// Environment control
	extraEnv map[string]string
}

// CodexOption configures CodexCLI.
type CodexOption func(*CodexCLI)

// NewCodexCLI creates a new Codex CLI client.
// Assumes "codex" is available in PATH unless overridden with WithCodexPath.
func NewCodexCLI(opts ...CodexOption) *CodexCLI {
	c := &CodexCLI{
		path:    "codex",
		timeout: 5 * time.Minute,
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

// WithFullAuto enables automatic approvals.
func WithFullAuto() CodexOption {
	return func(c *CodexCLI) { c.fullAuto = true }
}

// WithDangerouslyBypassApprovalsAndSandbox enables Codex YOLO mode.
func WithDangerouslyBypassApprovalsAndSandbox() CodexOption {
	return func(c *CodexCLI) { c.dangerouslyBypassApprovalsAndSandbox = true }
}

// WithSessionID sets the session ID for resuming sessions.
// Use "last" to resume the most recent session.
func WithSessionID(id string) CodexOption {
	return func(c *CodexCLI) { c.sessionID = id }
}

// WithResumeAll includes all previous conversation turns when resuming.
// Only applies when session is "last" or resume mode is active.
func WithResumeAll() CodexOption {
	return func(c *CodexCLI) { c.resumeAll = true }
}

// WithSearch enables live web search capabilities.
// Deprecated compatibility alias for WithWebSearchMode(WebSearchLive).
func WithSearch() CodexOption {
	return func(c *CodexCLI) {
		c.enableSearch = true
		if c.webSearchMode == "" {
			c.webSearchMode = WebSearchLive
		}
	}
}

// WithWebSearchMode sets the web search mode.
func WithWebSearchMode(mode WebSearchMode) CodexOption {
	return func(c *CodexCLI) { c.webSearchMode = mode }
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

// WithProfile sets the codex profile name.
func WithProfile(profile string) CodexOption {
	return func(c *CodexCLI) { c.profile = profile }
}

// WithLocalProvider selects the local OSS provider backend (e.g., "lmstudio", "ollama").
func WithLocalProvider(provider string) CodexOption {
	return func(c *CodexCLI) { c.localProvider = provider }
}

// WithConfigOverride adds a single -c key=value override.
func WithConfigOverride(key string, value any) CodexOption {
	return func(c *CodexCLI) {
		if c.configOverrides == nil {
			c.configOverrides = make(map[string]any)
		}
		c.configOverrides[key] = value
	}
}

// WithConfigOverrides adds multiple -c key=value overrides.
func WithConfigOverrides(overrides map[string]any) CodexOption {
	return func(c *CodexCLI) {
		if c.configOverrides == nil {
			c.configOverrides = make(map[string]any, len(overrides))
		}
		for k, v := range overrides {
			c.configOverrides[k] = v
		}
	}
}

// WithSkipGitRepoCheck allows execution outside a git repository.
func WithSkipGitRepoCheck() CodexOption {
	return func(c *CodexCLI) { c.skipGitRepoCheck = true }
}

// WithOutputSchema sets --output-schema.
func WithOutputSchema(path string) CodexOption {
	return func(c *CodexCLI) { c.outputSchemaPath = path }
}

// WithOutputLastMessage sets --output-last-message.
func WithOutputLastMessage(path string) CodexOption {
	return func(c *CodexCLI) { c.outputLastMessagePath = path }
}

// WithReasoningEffort sets model_reasoning_effort via -c override.
func WithReasoningEffort(effort string) CodexOption {
	return func(c *CodexCLI) { c.reasoningEffort = effort }
}

// WithHideAgentReasoning hides internal reasoning via -c override.
func WithHideAgentReasoning() CodexOption {
	return func(c *CodexCLI) { c.hideAgentReasoning = true }
}

// WithOSS enables the local OSS backend mode.
func WithOSS() CodexOption {
	return func(c *CodexCLI) { c.useOSS = true }
}

// WithEnabledFeatures enables one or more codex feature flags.
func WithEnabledFeatures(features []string) CodexOption {
	return func(c *CodexCLI) {
		c.enabledFeatures = append(c.enabledFeatures, features...)
	}
}

// WithDisabledFeatures disables one or more codex feature flags.
func WithDisabledFeatures(features []string) CodexOption {
	return func(c *CodexCLI) {
		c.disabledFeatures = append(c.disabledFeatures, features...)
	}
}

// WithColorMode sets output color mode: auto, always, or never.
func WithColorMode(mode string) CodexOption {
	return func(c *CodexCLI) { c.colorMode = mode }
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

	if len(c.extraEnv) > 0 {
		cmd.Env = os.Environ()
		for k, v := range c.extraEnv {
			cmd.Env = setEnvVar(cmd.Env, k, v)
		}
	}
}

// resolvedPath returns an absolute command path when needed.
func (c *CodexCLI) resolvedPath() string {
	if filepath.IsAbs(c.path) {
		return c.path
	}
	if c.workdir == "" {
		return c.path
	}
	abs, err := exec.LookPath(c.path)
	if err != nil {
		slog.Debug("could not resolve executable path", "path", c.path, "error", err)
		return c.path
	}
	return abs
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

// withTimeoutContext applies client timeout if caller did not set a deadline.
func (c *CodexCLI) withTimeoutContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

// Complete implements Client.
func (c *CodexCLI) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	ctx, cancel := c.withTimeoutContext(ctx)
	defer cancel()

	start := time.Now()
	args, cleanup := c.buildArgsWithCleanup(req)
	defer cleanup()

	cmd := exec.CommandContext(ctx, c.resolvedPath(), args...)
	c.setupCmd(cmd)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Ensure child processes are terminated on context cancellation.
	stopKill := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		case <-stopKill:
		}
	}()

	err := cmd.Run()
	close(stopKill)
	if err != nil {
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
	ctx, cancel := c.withTimeoutContext(ctx)

	args, cleanup := c.buildArgsWithCleanup(req)
	cmd := exec.CommandContext(ctx, c.resolvedPath(), args...)
	c.setupCmd(cmd)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		cleanup()
		return nil, NewError("stream", fmt.Errorf("create stdout pipe: %w", err), false)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		cleanup()
		return nil, NewError("stream", fmt.Errorf("start command: %w", err), false)
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer cleanup()
		defer cancel()

		killDone := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if cmd.Process != nil {
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				}
			case <-killDone:
			}
		}()
		defer close(killDone)

		scanner := bufio.NewScanner(stdout)
		const maxScanTokenSize = 10 * 1024 * 1024
		scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)

		var sawTerminal bool
		var sawText bool
	scanLoop:
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 {
				continue
			}

			event, parseErr := parseEventLine(line)
			if parseErr != nil {
				text := string(line)
				sawText = true
				if !sendChunk(ctx, ch, StreamChunk{Content: text + "\n"}) {
					return
				}
				continue
			}

			if event.Text != "" {
				if event.TextFromTurnOutput && sawText {
					// Avoid duplicate final text when turn output mirrors streamed deltas.
				} else {
					sawText = true
					if !sendChunk(ctx, ch, StreamChunk{Content: event.Text}) {
						return
					}
				}
			}
			if len(event.ToolCalls) > 0 {
				if !sendChunk(ctx, ch, StreamChunk{ToolCalls: event.ToolCalls}) {
					return
				}
			}
			if event.ErrMsg != "" {
				sawTerminal = true
				if !sendChunk(ctx, ch, StreamChunk{Error: NewError("stream", fmt.Errorf("%s", event.ErrMsg), false)}) {
					return
				}
				break scanLoop
			}
			if event.Done {
				sawTerminal = true
				chunk := StreamChunk{Done: true}
				if event.Usage != nil {
					chunk.Usage = event.Usage
				}
				if !sendChunk(ctx, ch, chunk) {
					return
				}
				break scanLoop
			}
		}

		if err := scanner.Err(); err != nil {
			_ = sendChunk(ctx, ch, StreamChunk{Error: NewError("stream", fmt.Errorf("read output: %w", err), false)})
			return
		}

		if waitErr := cmd.Wait(); waitErr != nil {
			if ctx.Err() != nil {
				_ = sendChunk(ctx, ch, StreamChunk{Error: ctx.Err()})
				return
			}
			if !sawTerminal {
				errMsg := sanitizeStderr(stderr.String())
				if errMsg == "" {
					errMsg = waitErr.Error()
				}
				_ = sendChunk(ctx, ch, StreamChunk{Error: NewError("stream", fmt.Errorf("%s", errMsg), isRetryableError(errMsg))})
			}
			return
		}

		if !sawTerminal {
			_ = sendChunk(ctx, ch, StreamChunk{Done: true})
		}
	}()

	return ch, nil
}

func sendChunk(ctx context.Context, ch chan<- StreamChunk, chunk StreamChunk) bool {
	select {
	case ch <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}

// buildArgsWithCleanup constructs CLI arguments and returns a cleanup function.
func (c *CodexCLI) buildArgsWithCleanup(req CompletionRequest) ([]string, func()) {
	args := c.buildExecArgs(req)
	cleanup := func() {}
	return args, cleanup
}

func (c *CodexCLI) buildExecArgs(req CompletionRequest) []string {
	resumeID := strings.TrimSpace(c.sessionID)

	args := []string{codexcontract.CommandExec}
	if resumeID != "" {
		args = append(args, codexcontract.CommandResume)
		if strings.EqualFold(resumeID, "last") {
			args = append(args, "--last")
			if c.resumeAll {
				args = append(args, codexcontract.FlagAll)
			}
		} else {
			args = append(args, resumeID)
		}
	}

	args = append(args, codexcontract.FlagJSON)

	model := c.model
	if req.Model != "" {
		model = req.Model
	}
	if model != "" {
		args = append(args, codexcontract.FlagModel, model)
	}

	if c.profile != "" {
		args = append(args, codexcontract.FlagProfile, c.profile)
	}
	if c.localProvider != "" {
		args = append(args, codexcontract.FlagLocalProvider, c.localProvider)
	}
	if c.useOSS {
		args = append(args, codexcontract.FlagOSS)
	}
	if c.colorMode != "" {
		args = append(args, codexcontract.FlagColor, c.colorMode)
	}
	for _, feature := range c.enabledFeatures {
		if feature == "" {
			continue
		}
		args = append(args, codexcontract.FlagEnable, feature)
	}
	for _, feature := range c.disabledFeatures {
		if feature == "" {
			continue
		}
		args = append(args, codexcontract.FlagDisable, feature)
	}

	if c.skipGitRepoCheck {
		args = append(args, codexcontract.FlagSkipGitRepoCheck)
	}

	if c.fullAuto {
		args = append(args, codexcontract.FlagFullAuto)
	}
	if c.dangerouslyBypassApprovalsAndSandbox {
		args = append(args, codexcontract.FlagDangerouslyBypassApprovalsSandbox)
	}

	if c.sandboxMode != "" && !c.dangerouslyBypassApprovalsAndSandbox {
		args = append(args, codexcontract.FlagSandbox, string(c.sandboxMode))
	}
	if c.approvalMode != "" && !c.dangerouslyBypassApprovalsAndSandbox {
		args = append(args, codexcontract.FlagAskForApproval, string(c.approvalMode))
	}

	if c.workdir != "" {
		args = append(args, codexcontract.FlagCD, c.workdir)
	}

	for _, dir := range c.addDirs {
		args = append(args, codexcontract.FlagAddDir, dir)
	}
	for _, img := range c.images {
		args = append(args, codexcontract.FlagImage, img)
	}

	webSearchMode := c.webSearchMode
	if req.WebSearchMode != "" {
		webSearchMode = req.WebSearchMode
	}

	reqForOverrides := req
	if webSearchMode != "" {
		reqForOverrides.ConfigOverrides = cloneOverrides(req.ConfigOverrides)
		reqForOverrides.ConfigOverrides["web_search"] = string(webSearchMode)
	} else if c.enableSearch {
		reqForOverrides.ConfigOverrides = cloneOverrides(req.ConfigOverrides)
		reqForOverrides.ConfigOverrides["web_search"] = string(WebSearchLive)
	}

	outputSchemaPath := c.outputSchemaPath
	if req.OutputSchemaPath != "" {
		outputSchemaPath = req.OutputSchemaPath
	}
	if outputSchemaPath != "" {
		args = append(args, codexcontract.FlagOutputSchema, outputSchemaPath)
	}

	outputLastMessagePath := c.outputLastMessagePath
	if req.OutputLastMessagePath != "" {
		outputLastMessagePath = req.OutputLastMessagePath
	}
	if outputLastMessagePath != "" {
		args = append(args, codexcontract.FlagOutputLastMessage, outputLastMessagePath)
	}

	for _, override := range c.mergedConfigOverrides(reqForOverrides) {
		args = append(args, codexcontract.FlagConfig, override)
	}

	prompt := c.buildPrompt(req)
	if prompt != "" {
		args = append(args, prompt)
	}

	return args
}

func (c *CodexCLI) mergedConfigOverrides(req CompletionRequest) []string {
	if len(c.configOverrides) == 0 && len(req.ConfigOverrides) == 0 && c.reasoningEffort == "" && !c.hideAgentReasoning {
		return nil
	}

	merged := make(map[string]any, len(c.configOverrides)+len(req.ConfigOverrides)+2)
	for k, v := range c.configOverrides {
		merged[k] = v
	}
	for k, v := range req.ConfigOverrides {
		merged[k] = v
	}
	if c.reasoningEffort != "" {
		merged["model_reasoning_effort"] = c.reasoningEffort
	}
	if c.hideAgentReasoning {
		merged["hide_agent_reasoning"] = true
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	overrides := make([]string, 0, len(keys))
	for _, k := range keys {
		overrides = append(overrides, k+"="+formatConfigValue(merged[k]))
	}
	return overrides
}

func formatConfigValue(v any) string {
	switch t := v.(type) {
	case string:
		return strconv.Quote(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", t)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", t)
	case float32, float64:
		return fmt.Sprintf("%v", t)
	case nil:
		return "null"
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return strconv.Quote(fmt.Sprintf("%v", t))
		}
		return string(b)
	}
}

func cloneOverrides(in map[string]any) map[string]any {
	if len(in) == 0 {
		return make(map[string]any)
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// buildPrompt constructs the prompt from messages.
func (c *CodexCLI) buildPrompt(req CompletionRequest) string {
	var prompt strings.Builder

	if req.SystemPrompt != "" {
		prompt.WriteString("System: ")
		prompt.WriteString(req.SystemPrompt)
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
		case RoleTool:
			if msg.Name != "" {
				prompt.WriteString("Tool(")
				prompt.WriteString(msg.Name)
				prompt.WriteString("): ")
			} else {
				prompt.WriteString("Tool: ")
			}
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
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

	lines := bytes.Split(data, []byte("\n"))
	var contentBuilder strings.Builder
	var sawText bool
	var toolCalls []ToolCall

	for _, rawLine := range lines {
		line := bytes.TrimSpace(rawLine)
		if len(line) == 0 {
			continue
		}

		event, err := parseEventLine(line)
		if err != nil {
			contentBuilder.Write(line)
			contentBuilder.WriteString("\n")
			continue
		}

		if event.SessionID != "" {
			resp.SessionID = event.SessionID
		}
		if event.Usage != nil {
			resp.Usage = *event.Usage
		}
		if len(event.ToolCalls) > 0 {
			toolCalls = append(toolCalls, event.ToolCalls...)
		}
		if event.Text != "" {
			if !(event.TextFromTurnOutput && sawText) {
				sawText = true
				contentBuilder.WriteString(event.Text)
			}
		}
		if event.ErrMsg != "" {
			resp.FinishReason = "error"
			if contentBuilder.Len() == 0 {
				contentBuilder.WriteString(event.ErrMsg)
			}
		}
	}

	resp.Content = strings.TrimSpace(contentBuilder.String())
	resp.ToolCalls = toolCalls

	if resp.Usage.TotalTokens == 0 && (resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0) {
		resp.Usage.TotalTokens = resp.Usage.InputTokens + resp.Usage.OutputTokens
	}

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

type parsedLineEvent struct {
	Text               string
	TextFromTurnOutput bool
	ToolCalls          []ToolCall
	SessionID          string
	Usage              *TokenUsage
	Done               bool
	ErrMsg             string
}

func parseEventLine(line []byte) (*parsedLineEvent, error) {
	var event map[string]any
	if err := json.Unmarshal(line, &event); err != nil {
		return nil, err
	}

	parsed := &parsedLineEvent{}
	eventType := getString(event, "type")
	parsed.SessionID = firstNonEmpty(getString(event, "thread_id"), getString(event, "session_id"), getString(event, "id"))

	if usage := parseUsage(event["usage"]); usage != nil {
		parsed.Usage = usage
	}

	switch eventType {
	case codexcontract.EventContent, codexcontract.EventText, codexcontract.EventAssistant, codexcontract.EventMessage:
		parsed.Text = firstNonEmpty(getString(event, "content"), getString(event, "message"))

	case codexcontract.EventToolCall:
		if tc := parseToolCallMap(getMap(event, "tool_call")); tc != nil {
			parsed.ToolCalls = append(parsed.ToolCalls, *tc)
		}

	case codexcontract.EventSession:
		// Session ID already captured.

	case codexcontract.EventUsage:
		// Usage already captured.

	case codexcontract.EventDone, codexcontract.EventComplete, codexcontract.EventEnd:
		parsed.Done = true

	case codexcontract.EventResult:
		if resultText := parseResultText(event["result"]); resultText != "" {
			parsed.Text = resultText
		}

	case codexcontract.EventThreadStarted:
		// Session/thread already captured.

	case codexcontract.EventItemStarted, codexcontract.EventItemUpdated, codexcontract.EventItemCompleted:
		itemMap := toMap(event["item"])
		if len(itemMap) > 0 {
			itemType := getString(itemMap, "type")
			switch itemType {
			case codexcontract.ItemReasoning:
				// Intentionally ignored for user-facing content.
			default:
				if isAgentMessageType(itemType) {
					parsed.Text = firstNonEmpty(getString(itemMap, "delta"), getString(itemMap, "text"), extractTextFromContent(itemMap["content"]))
				}
				if tc := parseToolCallFromItem(itemMap); tc != nil {
					parsed.ToolCalls = append(parsed.ToolCalls, *tc)
				}
			}
		}

	case codexcontract.EventTurnCompleted:
		parsed.Done = true
		if outText := extractTextFromContent(event["output"]); outText != "" {
			parsed.Text = outText
			parsed.TextFromTurnOutput = true
		}
		if parsed.Usage == nil {
			if usage := parseUsage(event["turn_usage"]); usage != nil {
				parsed.Usage = usage
			}
		}

	case codexcontract.EventTurnFailed, codexcontract.EventError:
		parsed.Done = true
		parsed.ErrMsg = firstNonEmpty(getString(event, "error"), getString(event, "message"), parseResultText(event["result"]))
		if parsed.ErrMsg == "" {
			parsed.ErrMsg = "codex turn failed"
		}

	default:
		// Unknown event type: best effort extraction for forward compatibility.
		if parsed.Text == "" {
			parsed.Text = firstNonEmpty(getString(event, "content"), getString(event, "message"))
		}
		if parsed.Text == "" {
			if item := toMap(event["item"]); len(item) > 0 {
				parsed.Text = extractTextFromContent(item["content"])
			}
		}
	}

	return parsed, nil
}

func isAgentMessageType(itemType string) bool {
	if itemType == "" {
		return true
	}
	itemType = strings.ToLower(itemType)
	return itemType == codexcontract.ItemAgentMessage ||
		itemType == "assistant_message" ||
		itemType == "message" ||
		itemType == "output_text"
}

func parseToolCallFromItem(item map[string]any) *ToolCall {
	if tc := parseToolCallMap(getMap(item, "tool_call")); tc != nil {
		if tc.ID == "" {
			tc.ID = getString(item, "id")
		}
		return tc
	}

	itemType := strings.ToLower(getString(item, "type"))
	if !strings.Contains(itemType, "tool") && !strings.Contains(itemType, "command") {
		return nil
	}

	name := firstNonEmpty(getString(item, "name"), getString(item, "tool_name"), getString(item, "command"))
	if name == "" {
		return nil
	}

	var args json.RawMessage
	if raw := item["arguments"]; raw != nil {
		args = toRawJSON(raw)
	} else if raw := item["input"]; raw != nil {
		args = toRawJSON(raw)
	}

	return &ToolCall{
		ID:        getString(item, "id"),
		Name:      name,
		Arguments: args,
	}
}

func parseToolCallMap(m map[string]any) *ToolCall {
	if len(m) == 0 {
		return nil
	}
	name := getString(m, "name")
	if name == "" {
		return nil
	}
	var args json.RawMessage
	if raw := m["arguments"]; raw != nil {
		args = toRawJSON(raw)
	}
	return &ToolCall{
		ID:        getString(m, "id"),
		Name:      name,
		Arguments: args,
	}
}

func parseUsage(v any) *TokenUsage {
	m := toMap(v)
	if len(m) == 0 {
		return nil
	}
	usage := &TokenUsage{
		InputTokens:  firstNonZeroInt(toInt(m["input_tokens"]), toInt(m["inputTokens"])),
		OutputTokens: firstNonZeroInt(toInt(m["output_tokens"]), toInt(m["outputTokens"])),
		TotalTokens:  firstNonZeroInt(toInt(m["total_tokens"]), toInt(m["totalTokens"])),
	}
	if usage.TotalTokens == 0 && (usage.InputTokens > 0 || usage.OutputTokens > 0) {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	if usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	return usage
}

func parseResultText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	m := toMap(v)
	if len(m) == 0 {
		return ""
	}
	return firstNonEmpty(getString(m, "content"), getString(m, "text"), extractTextFromContent(m["output"]))
}

func extractTextFromContent(v any) string {
	var out strings.Builder
	appendExtractedText(&out, v)
	return out.String()
}

func appendExtractedText(out *strings.Builder, v any) {
	switch x := v.(type) {
	case string:
		out.WriteString(x)
	case []any:
		for _, item := range x {
			appendExtractedText(out, item)
		}
	case map[string]any:
		if t, ok := x["type"].(string); ok && strings.EqualFold(t, codexcontract.ItemReasoning) {
			return
		}
		if delta := getString(x, "delta"); delta != "" {
			out.WriteString(delta)
		}
		if text := getString(x, "text"); text != "" {
			out.WriteString(text)
		}
		if content, ok := x["content"]; ok {
			appendExtractedText(out, content)
		}
		if message, ok := x["message"]; ok {
			appendExtractedText(out, message)
		}
		if output, ok := x["output"]; ok {
			appendExtractedText(out, output)
		}
	}
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	return toMap(m[key])
}

func toMap(v any) map[string]any {
	switch x := v.(type) {
	case map[string]any:
		return x
	case json.RawMessage:
		if len(x) == 0 {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(x, &m); err == nil {
			return m
		}
	case []byte:
		if len(x) == 0 {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(x, &m); err == nil {
			return m
		}
	}
	return nil
}

func toRawJSON(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	if raw, ok := v.(json.RawMessage); ok {
		return raw
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonZeroInt(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

// Resume resumes a previous session by ID.
// Use sessionID="last" to resume the most recent session.
func (c *CodexCLI) Resume(ctx context.Context, sessionID, prompt string) (*CompletionResponse, error) {
	ctx, cancel := c.withTimeoutContext(ctx)
	defer cancel()

	start := time.Now()

	args := []string{codexcontract.CommandExec, codexcontract.CommandResume}
	if strings.EqualFold(strings.TrimSpace(sessionID), "last") || strings.TrimSpace(sessionID) == "" {
		args = append(args, "--last")
		if c.resumeAll {
			args = append(args, codexcontract.FlagAll)
		}
	} else {
		args = append(args, sessionID)
	}
	args = append(args, codexcontract.FlagJSON)
	if prompt != "" {
		args = append(args, prompt)
	}

	cmd := exec.CommandContext(ctx, c.resolvedPath(), args...)
	c.setupCmd(cmd)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

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
	if resp.SessionID == "" {
		resp.SessionID = sessionID
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
		MCP:       false, // MCP uses ~/.codex/config.toml
		Sessions:  true,
		Images:    true,
		NativeTools: []string{
			"shell",
			"apply_patch",
			"read_file",
			"list_dir",
			"web_search",
		},
		ContextFile: "AGENTS.md",
	}
}

// Close releases any resources held by the client.
func (c *CodexCLI) Close() error {
	return nil
}
