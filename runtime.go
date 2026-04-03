package llmkit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/randalmurphal/llmkit/v2/claudeconfig"
	"github.com/randalmurphal/llmkit/v2/env"
)

type RuntimeConfig struct {
	Shared    SharedRuntimeConfig   `json:"shared,omitempty"`
	Providers RuntimeProviderConfig `json:"providers,omitempty"`
}

type SharedRuntimeConfig struct {
	SystemPrompt       string                     `json:"system_prompt,omitempty"`
	AppendSystemPrompt string                     `json:"append_system_prompt,omitempty"`
	AllowedTools       []string                   `json:"allowed_tools,omitempty"`
	DisallowedTools    []string                   `json:"disallowed_tools,omitempty"`
	Tools              []string                   `json:"tools,omitempty"`
	MCPServers         map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
	StrictMCPConfig    bool                       `json:"strict_mcp_config,omitempty"`
	MaxBudgetUSD       float64                    `json:"max_budget_usd,omitempty"`
	MaxTurns           int                        `json:"max_turns,omitempty"`
	Env                map[string]string          `json:"env,omitempty"`
	AddDirs            []string                   `json:"add_dirs,omitempty"`
}

type RuntimeProviderConfig struct {
	Claude *ClaudeRuntimeConfig `json:"claude,omitempty"`
	Codex  *CodexRuntimeConfig  `json:"codex,omitempty"`
}

type ClaudeRuntimeConfig struct {
	SystemPromptFile           string                    `json:"system_prompt_file,omitempty"`
	AppendSystemPromptFile     string                    `json:"append_system_prompt_file,omitempty"`
	SkillRefs                  []string                  `json:"skill_refs,omitempty"`
	AgentRef                   string                    `json:"agent_ref,omitempty"`
	InlineAgents               map[string]InlineAgentDef `json:"inline_agents,omitempty"`
	Hooks                      map[string][]HookMatcher  `json:"hooks,omitempty"`
	DangerouslySkipPermissions bool                      `json:"dangerously_skip_permissions,omitempty"`
	PermissionMode             string                    `json:"permission_mode,omitempty"`
	SettingSources             []string                  `json:"setting_sources,omitempty"`
}

type CodexRuntimeConfig struct {
	ReasoningEffort           string `json:"reasoning_effort,omitempty"`
	WebSearchMode             string `json:"web_search_mode,omitempty"`
	SandboxMode               string `json:"sandbox_mode,omitempty"`
	ApprovalMode              string `json:"approval_mode,omitempty"`
	BypassApprovalsAndSandbox bool   `json:"bypass_approvals_and_sandbox,omitempty"`
}

type InlineAgentDef struct {
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	Tools       []string `json:"tools,omitempty"`
	Model       string   `json:"model,omitempty"`
}

type HookMatcher struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []HookEntry `json:"hooks"`
}

type HookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
	Once    bool   `json:"once,omitempty"`
}

type SkillAsset struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Content         string            `json:"content"`
	SupportingFiles map[string]string `json:"supporting_files,omitempty"`
}

type RuntimeAssets struct {
	Skills      map[string]SkillAsset `json:"skills,omitempty"`
	HookScripts map[string]string     `json:"hook_scripts,omitempty"`
}

type PrepareRequest struct {
	Provider       string         `json:"provider"`
	WorkDir        string         `json:"work_dir"`
	RuntimeConfig  RuntimeConfig  `json:"runtime_config"`
	Assets         *RuntimeAssets `json:"assets,omitempty"`
	Tag            string         `json:"tag,omitempty"`
	RecoverOrphans bool           `json:"recover_orphans,omitempty"`
}

type PreparedRuntime struct {
	Provider string         `json:"provider"`
	Scope    io.Closer      `json:"-"`
	Metadata map[string]any `json:"metadata,omitempty"`

	cleanup []func() error
}

var hookRefPattern = regexp.MustCompile(`\{\{hook:([^}]+)\}\}`)

func PrepareRuntime(_ context.Context, req PrepareRequest) (*PreparedRuntime, error) {
	if req.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if req.WorkDir == "" {
		return nil, fmt.Errorf("work_dir is required")
	}
	if err := ValidateRuntimeConfig(req.Provider, req.RuntimeConfig); err != nil {
		return nil, err
	}

	prepared := &PreparedRuntime{
		Provider: req.Provider,
		Metadata: map[string]any{},
	}

	scopeCfg := env.ScopeConfig{
		MCPServers:     req.RuntimeConfig.Shared.MCPServers,
		Env:            req.RuntimeConfig.Shared.Env,
		Tag:            req.Tag,
		RecoverOrphans: req.RecoverOrphans,
	}

	if req.Provider == "claude" && req.RuntimeConfig.Providers.Claude != nil {
		scopeCfg.Hooks = convertClaudeHooks(req.WorkDir, req.RuntimeConfig.Providers.Claude.Hooks)
	}

	scope, err := env.NewScope(req.Provider, req.WorkDir, scopeCfg)
	if err != nil {
		return nil, fmt.Errorf("create environment scope: %w", err)
	}
	prepared.Scope = scope

	if err := writeRuntimeAssets(req, prepared); err != nil {
		_ = prepared.Close()
		return nil, err
	}

	return prepared, nil
}

func (p *PreparedRuntime) Close() error {
	if p == nil {
		return nil
	}

	var errs []error
	for i := len(p.cleanup) - 1; i >= 0; i-- {
		if err := p.cleanup[i](); err != nil {
			errs = append(errs, err)
		}
	}
	if p.Scope != nil {
		if err := p.Scope.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errorsJoin(errs...)
}

func writeRuntimeAssets(req PrepareRequest, prepared *PreparedRuntime) error {
	if req.Provider != "claude" || req.RuntimeConfig.Providers.Claude == nil {
		return nil
	}
	cfg := req.RuntimeConfig.Providers.Claude
	if req.Assets == nil {
		req.Assets = &RuntimeAssets{}
	}

	if err := writeHookScripts(req.WorkDir, req.Assets.HookScripts); err != nil {
		return err
	}

	if len(cfg.SkillRefs) > 0 {
		created, err := writeClaudeSkills(req.WorkDir, cfg.SkillRefs, req.Assets.Skills)
		if err != nil {
			return err
		}
		prepared.cleanup = append(prepared.cleanup, func() error {
			return removeAll(created)
		})
		prepared.Metadata["skills"] = created
	}

	if len(cfg.InlineAgents) > 0 {
		restore, names, err := writeClaudeInlineAgents(req.WorkDir, cfg.InlineAgents)
		if err != nil {
			return err
		}
		prepared.cleanup = append(prepared.cleanup, restore)
		prepared.Metadata["inline_agents"] = names
	}

	return nil
}

func convertClaudeHooks(workDir string, hooks map[string][]HookMatcher) map[string][]env.Hook {
	if len(hooks) == 0 {
		return nil
	}
	out := make(map[string][]env.Hook, len(hooks))
	for event, matchers := range hooks {
		for _, matcher := range matchers {
			for _, hook := range matcher.Hooks {
				command := hook.Command
				command = hookRefPattern.ReplaceAllStringFunc(command, func(match string) string {
					parts := hookRefPattern.FindStringSubmatch(match)
					if len(parts) != 2 {
						return match
					}
					return filepath.Join(workDir, ".claude", "hooks", parts[1])
				})
				out[event] = append(out[event], env.Hook{
					Matcher: matcher.Matcher,
					Type:    hook.Type,
					Command: command,
					Prompt:  hook.Prompt,
					Timeout: hook.Timeout,
					Once:    hook.Once,
				})
			}
		}
	}
	return out
}

func writeHookScripts(workDir string, scripts map[string]string) error {
	if len(scripts) == 0 {
		return nil
	}
	hooksDir := filepath.Join(workDir, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}
	for name, content := range scripts {
		if err := validateAssetPathComponent(name); err != nil {
			return fmt.Errorf("invalid hook script %q: %w", name, err)
		}
		path := filepath.Join(hooksDir, name)
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			return fmt.Errorf("write hook script %s: %w", name, err)
		}
	}
	return nil
}

func writeClaudeSkills(workDir string, refs []string, skills map[string]SkillAsset) ([]string, error) {
	created := make([]string, 0, len(refs))
	for _, ref := range refs {
		asset, ok := skills[ref]
		if !ok {
			return nil, fmt.Errorf("skill asset %q not provided", ref)
		}
		if err := validateAssetPathComponent(ref); err != nil {
			return nil, fmt.Errorf("invalid skill ref %q: %w", ref, err)
		}
		dir := filepath.Join(workDir, ".claude", "skills", ref)
		skill := &claudeconfig.Skill{
			Name:        asset.Name,
			Description: asset.Description,
			Content:     asset.Content,
		}
		if err := claudeconfig.WriteSkillMD(skill, dir); err != nil {
			return nil, fmt.Errorf("write skill %s: %w", ref, err)
		}
		for name, content := range asset.SupportingFiles {
			if err := validateRelativeAssetPath(name); err != nil {
				return nil, fmt.Errorf("invalid skill supporting file %q: %w", name, err)
			}
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, fmt.Errorf("create supporting file dir: %w", err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return nil, fmt.Errorf("write skill supporting file %s/%s: %w", ref, name, err)
			}
		}
		created = append(created, dir)
	}
	return created, nil
}

func writeClaudeInlineAgents(workDir string, defs map[string]InlineAgentDef) (func() error, []string, error) {
	svc := claudeconfig.NewAgentService(workDir)
	existing, err := svc.List()
	if err != nil {
		return nil, nil, fmt.Errorf("load existing claude agents: %w", err)
	}
	original := append([]claudeconfig.SubAgent(nil), existing...)
	names := make([]string, 0, len(defs))

	for name, def := range defs {
		agent := claudeconfig.SubAgent{
			Name:        name,
			Description: def.Description,
			Model:       def.Model,
			Prompt:      def.Prompt,
			SkillRefs:   nil,
		}
		if len(def.Tools) > 0 {
			agent.Tools = &claudeconfig.ToolPermissions{Allow: append([]string(nil), def.Tools...)}
		}
		if _, err := svc.Get(name); err == nil {
			if err := svc.Update(name, agent); err != nil {
				return nil, nil, fmt.Errorf("update inline agent %s: %w", name, err)
			}
		} else {
			if err := svc.Create(agent); err != nil {
				return nil, nil, fmt.Errorf("create inline agent %s: %w", name, err)
			}
		}
		names = append(names, name)
	}
	slices.Sort(names)

	return func() error {
		settings, err := claudeconfig.LoadProjectSettings(workDir)
		if err != nil {
			return fmt.Errorf("load project settings for agent restore: %w", err)
		}
		settings.SetExtension("agents", original)
		if err := claudeconfig.SaveProjectSettings(workDir, settings); err != nil {
			return fmt.Errorf("restore inline agents: %w", err)
		}
		return nil
	}, names, nil
}

func validateAssetPathComponent(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || filepath.IsAbs(name) {
		return fmt.Errorf("path traversal is not allowed")
	}
	return nil
}

func validateRelativeAssetPath(name string) error {
	if name == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(name) || strings.Contains(name, "..") {
		return fmt.Errorf("path traversal is not allowed")
	}
	return nil
}

func removeAll(paths []string) error {
	var errs []error
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			errs = append(errs, err)
		}
	}
	return errorsJoin(errs...)
}

func errorsJoin(errs ...error) error {
	filtered := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	parts := make([]string, 0, len(filtered))
	for _, err := range filtered {
		parts = append(parts, err.Error())
	}
	return fmt.Errorf(strings.Join(parts, "; "))
}

func validateCodexReasoningEffort(value string) error {
	if value == "" {
		return nil
	}
	switch strings.ToLower(value) {
	case "minimal", "low", "medium", "high", "xhigh":
		return nil
	default:
		return fmt.Errorf("invalid reasoning_effort %q", value)
	}
}

func validateCodexWebSearchMode(value string) error {
	if value == "" {
		return nil
	}
	switch strings.ToLower(value) {
	case "cached", "live", "disabled":
		return nil
	default:
		return fmt.Errorf("invalid web_search_mode %q", value)
	}
}

func validateClaudeRuntimeConfig(cfg RuntimeConfig) error {
	if cfg.Providers.Codex != nil {
		return fmt.Errorf("providers.codex is not valid when provider=claude")
	}
	return nil
}

func validateCodexRuntimeConfig(cfg RuntimeConfig) error {
	if cfg.Providers.Claude != nil {
		return fmt.Errorf("providers.claude is not valid when provider=codex")
	}
	if cfg.Providers.Codex == nil {
		return nil
	}
	if err := validateCodexReasoningEffort(cfg.Providers.Codex.ReasoningEffort); err != nil {
		return err
	}
	if err := validateCodexWebSearchMode(cfg.Providers.Codex.WebSearchMode); err != nil {
		return err
	}
	return nil
}

func buildRuntimeSystemPrompt(workDir string, cfg RuntimeConfig, provider string) (string, error) {
	prompt := cfg.Shared.SystemPrompt
	appendPrompt := cfg.Shared.AppendSystemPrompt

	if provider == "claude" && cfg.Providers.Claude != nil {
		if cfg.Providers.Claude.SystemPromptFile != "" {
			content, err := loadPromptFile(workDir, cfg.Providers.Claude.SystemPromptFile)
			if err != nil {
				return "", err
			}
			prompt = content
		}
		if cfg.Providers.Claude.AppendSystemPromptFile != "" {
			content, err := loadPromptFile(workDir, cfg.Providers.Claude.AppendSystemPromptFile)
			if err != nil {
				return "", err
			}
			appendPrompt = strings.TrimSpace(strings.Join([]string{appendPrompt, content}, "\n\n"))
		}
	}

	if prompt == "" {
		return appendPrompt, nil
	}
	if appendPrompt == "" {
		return prompt, nil
	}
	return strings.TrimSpace(prompt + "\n\n" + appendPrompt), nil
}

func loadPromptFile(workDir, path string) (string, error) {
	fullPath := path
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(workDir, path)
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read prompt file %s: %w", fullPath, err)
	}
	return string(data), nil
}

func BuildConfig(provider, model, workDir string, runtime RuntimeConfig, session *SessionMetadata) (Config, error) {
	if err := ValidateRuntimeConfig(provider, runtime); err != nil {
		return Config{}, err
	}

	systemPrompt, err := buildRuntimeSystemPrompt(workDir, runtime, provider)
	if err != nil {
		return Config{}, err
	}

	cfg := DefaultConfig()
	cfg.Provider = provider
	cfg.Model = model
	cfg.WorkDir = workDir
	cfg.SystemPrompt = systemPrompt
	cfg.MaxTurns = runtime.Shared.MaxTurns
	cfg.MaxBudgetUSD = runtime.Shared.MaxBudgetUSD
	cfg.AllowedTools = append([]string(nil), runtime.Shared.AllowedTools...)
	cfg.DisallowedTools = append([]string(nil), runtime.Shared.DisallowedTools...)
	cfg.Tools = append([]string(nil), runtime.Shared.Tools...)
	cfg.MCPServers = cloneMCPServers(runtime.Shared.MCPServers)
	cfg.StrictMCPConfig = runtime.Shared.StrictMCPConfig
	cfg.Env = cloneStringMap(runtime.Shared.Env)
	cfg.AddDirs = append([]string(nil), runtime.Shared.AddDirs...)
	cfg.Session = session
	cfg.Runtime = runtime

	if provider == "codex" && runtime.Providers.Codex != nil {
		cfg.ReasoningEffort = runtime.Providers.Codex.ReasoningEffort
		cfg.WebSearchMode = runtime.Providers.Codex.WebSearchMode
	}

	return cfg, nil
}

func cloneMCPServers(in map[string]MCPServerConfig) map[string]MCPServerConfig {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]MCPServerConfig, len(in))
	for name, server := range in {
		out[name] = cloneMCPServer(server)
	}
	return out
}

func cloneMCPServer(in MCPServerConfig) MCPServerConfig {
	return MCPServerConfig{
		Type:     in.Type,
		Command:  in.Command,
		Args:     append([]string(nil), in.Args...),
		Env:      cloneStringMap(in.Env),
		URL:      in.URL,
		Headers:  cloneStringMap(in.Headers),
		Disabled: in.Disabled,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
