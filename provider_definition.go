package llmkit

import (
	"fmt"
	"sort"
	"sync"
)

type ProviderDefinition struct {
	Name        string             `json:"name"`
	Supported   bool               `json:"supported"`
	Shared      SharedSupport      `json:"shared"`
	Environment EnvironmentSupport `json:"environment"`
	validate    func(RuntimeConfig) error
}

type SharedSupport struct {
	SystemPrompt       bool `json:"system_prompt"`
	AppendSystemPrompt bool `json:"append_system_prompt"`
	AllowedTools       bool `json:"allowed_tools"`
	DisallowedTools    bool `json:"disallowed_tools"`
	Tools              bool `json:"tools"`
	MCPServers         bool `json:"mcp_servers"`
	StrictMCPConfig    bool `json:"strict_mcp_config"`
	MaxBudgetUSD       bool `json:"max_budget_usd"`
	MaxTurns           bool `json:"max_turns"`
	Env                bool `json:"env"`
	AddDirs            bool `json:"add_dirs"`
}

type EnvironmentSupport struct {
	Hooks        bool `json:"hooks"`
	MCP          bool `json:"mcp"`
	Skills       bool `json:"skills"`
	Instructions bool `json:"instructions"`
	CustomAgents bool `json:"custom_agents"`
}

var (
	providerDefinitionsMu sync.RWMutex
	providerDefinitions   = map[string]ProviderDefinition{}
)

func RegisterProviderDefinition(def ProviderDefinition) {
	if def.Name == "" {
		panic("provider definition name is required")
	}
	providerDefinitionsMu.Lock()
	defer providerDefinitionsMu.Unlock()
	if _, exists := providerDefinitions[def.Name]; exists {
		panic(fmt.Sprintf("provider definition %q already registered", def.Name))
	}
	providerDefinitions[def.Name] = def
}

func ListProviders() []ProviderDefinition {
	providerDefinitionsMu.RLock()
	defer providerDefinitionsMu.RUnlock()

	names := make([]string, 0, len(providerDefinitions))
	for name := range providerDefinitions {
		names = append(names, name)
	}
	sort.Strings(names)

	defs := make([]ProviderDefinition, 0, len(names))
	for _, name := range names {
		defs = append(defs, providerDefinitions[name].public())
	}
	return defs
}

func GetProviderDefinition(name string) (ProviderDefinition, bool) {
	providerDefinitionsMu.RLock()
	defer providerDefinitionsMu.RUnlock()

	def, ok := providerDefinitions[name]
	if !ok {
		return ProviderDefinition{}, false
	}
	return def.public(), true
}

func ValidateRuntimeConfig(provider string, cfg RuntimeConfig) error {
	providerDefinitionsMu.RLock()
	def, ok := providerDefinitions[provider]
	providerDefinitionsMu.RUnlock()
	if !ok {
		return fmt.Errorf("unsupported provider %q", provider)
	}
	if !def.Supported {
		return fmt.Errorf("provider %q is not supported", provider)
	}
	if err := validateUnsupportedSharedConfig(provider, def.Shared, cfg.Shared); err != nil {
		return err
	}
	switch provider {
	case "claude":
		return validateClaudeRuntimeConfig(cfg)
	case "codex":
		return validateCodexRuntimeConfig(cfg)
	}
	return nil
}

func (d ProviderDefinition) public() ProviderDefinition {
	d.validate = nil
	return d
}

func validateUnsupportedSharedConfig(provider string, support SharedSupport, cfg SharedRuntimeConfig) error {
	switch {
	case cfg.SystemPrompt != "" && !support.SystemPrompt:
		return fmt.Errorf("shared.system_prompt is not supported when provider=%s", provider)
	case cfg.AppendSystemPrompt != "" && !support.AppendSystemPrompt:
		return fmt.Errorf("shared.append_system_prompt is not supported when provider=%s", provider)
	case len(cfg.AllowedTools) > 0 && !support.AllowedTools:
		return fmt.Errorf("shared.allowed_tools is not supported when provider=%s", provider)
	case len(cfg.DisallowedTools) > 0 && !support.DisallowedTools:
		return fmt.Errorf("shared.disallowed_tools is not supported when provider=%s", provider)
	case len(cfg.Tools) > 0 && !support.Tools:
		return fmt.Errorf("shared.tools is not supported when provider=%s", provider)
	case len(cfg.MCPServers) > 0 && !support.MCPServers:
		return fmt.Errorf("shared.mcp_servers is not supported when provider=%s", provider)
	case cfg.StrictMCPConfig && !support.StrictMCPConfig:
		return fmt.Errorf("shared.strict_mcp_config is not supported when provider=%s", provider)
	case cfg.MaxBudgetUSD > 0 && !support.MaxBudgetUSD:
		return fmt.Errorf("shared.max_budget_usd is not supported when provider=%s", provider)
	case cfg.MaxTurns > 0 && !support.MaxTurns:
		return fmt.Errorf("shared.max_turns is not supported when provider=%s", provider)
	case len(cfg.Env) > 0 && !support.Env:
		return fmt.Errorf("shared.env is not supported when provider=%s", provider)
	case len(cfg.AddDirs) > 0 && !support.AddDirs:
		return fmt.Errorf("shared.add_dirs is not supported when provider=%s", provider)
	default:
		return nil
	}
}
