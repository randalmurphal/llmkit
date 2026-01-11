package continuedev

import (
	"context"

	"github.com/randalmurphal/llmkit/provider"
)

func init() {
	provider.Register("continue", newFromProviderConfig)
}

// newFromProviderConfig creates a ContinueCLI from a provider.Config.
func newFromProviderConfig(cfg provider.Config) (provider.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := make([]Option, 0, 16)

	// Map common config fields
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, WithTimeout(cfg.Timeout))
	}
	if cfg.WorkDir != "" {
		opts = append(opts, WithWorkdir(cfg.WorkDir))
	}
	if len(cfg.Env) > 0 {
		opts = append(opts, WithEnv(cfg.Env))
	}

	// Map Continue-specific options from Options map
	if cfg.Options != nil {
		// Path to cn binary
		if path := cfg.GetStringOption("path", ""); path != "" {
			opts = append(opts, WithPath(path))
		}

		// Config file path
		if configPath := cfg.GetStringOption("config_path", ""); configPath != "" {
			opts = append(opts, WithConfigPath(configPath))
		}

		// API key
		if apiKey := cfg.GetStringOption("api_key", ""); apiKey != "" {
			opts = append(opts, WithAPIKey(apiKey))
		}

		// Rule
		if rule := cfg.GetStringOption("rule", ""); rule != "" {
			opts = append(opts, WithRule(rule))
		}

		// Verbose
		if cfg.GetBoolOption("verbose", false) {
			opts = append(opts, WithVerbose())
		}

		// Resume
		if cfg.GetBoolOption("resume", false) {
			opts = append(opts, WithResume())
		}

		// Tool permissions
		if allowed := cfg.GetStringSliceOption("allowed_tools"); len(allowed) > 0 {
			opts = append(opts, WithAllowedTools(allowed))
		}
		if ask := cfg.GetStringSliceOption("ask_tools"); len(ask) > 0 {
			opts = append(opts, WithAskTools(ask))
		}
		if excluded := cfg.GetStringSliceOption("excluded_tools"); len(excluded) > 0 {
			opts = append(opts, WithExcludedTools(excluded))
		}
	}

	return &continueProviderAdapter{
		cli: NewContinueCLI(opts...),
	}, nil
}

// continueProviderAdapter wraps ContinueCLI to implement provider.Client.
type continueProviderAdapter struct {
	cli *ContinueCLI
}

// Complete implements provider.Client.
func (a *continueProviderAdapter) Complete(ctx context.Context, req provider.Request) (*provider.Response, error) {
	return a.cli.Complete(ctx, req)
}

// Stream implements provider.Client.
func (a *continueProviderAdapter) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamChunk, error) {
	return a.cli.Stream(ctx, req)
}

// Provider implements provider.Client.
func (a *continueProviderAdapter) Provider() string {
	return a.cli.Provider()
}

// Capabilities implements provider.Client.
func (a *continueProviderAdapter) Capabilities() provider.Capabilities {
	return a.cli.Capabilities()
}

// Close implements provider.Client.
func (a *continueProviderAdapter) Close() error {
	return a.cli.Close()
}
