package local

import (
	"context"
	"time"

	"github.com/randalmurphal/llmkit/provider"
)

func init() {
	provider.Register("local", newFromProviderConfig)
}

// newFromProviderConfig creates a local Client from a provider.Config.
// This is the factory function registered with the provider registry.
func newFromProviderConfig(cfg provider.Config) (provider.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	localCfg := Config{
		Model: cfg.Model,
	}

	// Map timeout
	if cfg.Timeout > 0 {
		localCfg.RequestTimeout = cfg.Timeout
	}

	// Map work dir
	if cfg.WorkDir != "" {
		localCfg.WorkDir = cfg.WorkDir
	}

	// Map env
	if len(cfg.Env) > 0 {
		localCfg.Env = cfg.Env
	}

	// Map local-specific options
	if cfg.Options != nil {
		// Backend
		if backend := cfg.GetStringOption("backend", ""); backend != "" {
			localCfg.Backend = Backend(backend)
		}

		// Sidecar path
		if path := cfg.GetStringOption("sidecar_path", ""); path != "" {
			localCfg.SidecarPath = path
		}

		// Host
		if host := cfg.GetStringOption("host", ""); host != "" {
			localCfg.Host = host
		}

		// Python path
		if pyPath := cfg.GetStringOption("python_path", ""); pyPath != "" {
			localCfg.PythonPath = pyPath
		}

		// Startup timeout
		if timeout := cfg.GetStringOption("startup_timeout", ""); timeout != "" {
			if d, err := time.ParseDuration(timeout); err == nil {
				localCfg.StartupTimeout = d
			}
		}
	}

	// Map MCP servers
	if cfg.MCP != nil && len(cfg.MCP.MCPServers) > 0 {
		localCfg.MCPServers = convertMCPConfig(cfg)
	}

	return &localProviderAdapter{
		client: NewClientWithConfig(localCfg),
	}, nil
}

// localProviderAdapter wraps Client to implement provider.Client.
type localProviderAdapter struct {
	client *Client
}

// Complete implements provider.Client.
func (a *localProviderAdapter) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	return a.client.Complete(ctx, req)
}

// Stream implements provider.Client.
func (a *localProviderAdapter) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	return a.client.Stream(ctx, req)
}

// Provider implements provider.Client.
func (a *localProviderAdapter) Provider() string {
	return "local"
}

// Capabilities implements provider.Client.
func (a *localProviderAdapter) Capabilities() provider.Capabilities {
	return a.client.Capabilities()
}

// Close implements provider.Client.
func (a *localProviderAdapter) Close() error {
	return a.client.Close()
}

// convertMCPConfig converts provider MCP config to local MCPServerConfig.
func convertMCPConfig(cfg provider.Config) map[string]MCPServerConfig {
	if cfg.MCP == nil || len(cfg.MCP.MCPServers) == 0 {
		return nil
	}

	servers := make(map[string]MCPServerConfig, len(cfg.MCP.MCPServers))
	for name, server := range cfg.MCP.MCPServers {
		if server == nil || server.Disabled {
			continue
		}
		servers[name] = MCPServerConfig{
			Type:    server.Type,
			Command: server.Command,
			Args:    server.Args,
			Env:     server.Env,
			URL:     server.URL,
			Headers: server.Headers,
		}
	}
	return servers
}
