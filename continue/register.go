package continuedev

import (
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

	// Map top-level tool permissions (if provider.Config has these)
	if len(cfg.AllowedTools) > 0 {
		opts = append(opts, WithAllowedTools(cfg.AllowedTools))
	}
	if len(cfg.DisallowedTools) > 0 {
		opts = append(opts, WithExcludedTools(cfg.DisallowedTools))
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

		// Tool permissions from Options (overrides top-level if set)
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

	// ContinueCLI implements provider.Client directly
	return NewContinueCLI(opts...), nil
}
