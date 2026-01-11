package aider

import (
	"github.com/randalmurphal/llmkit/provider"
)

func init() {
	provider.Register("aider", newFromProviderConfig)
}

// newFromProviderConfig creates an AiderCLI from a provider.Config.
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

	// Map Aider-specific options from Options map
	if cfg.Options != nil {
		// Path to aider binary
		if path := cfg.GetStringOption("path", ""); path != "" {
			opts = append(opts, WithPath(path))
		}

		// Ollama API base
		if base := cfg.GetStringOption("ollama_api_base", ""); base != "" {
			opts = append(opts, WithOllamaAPIBase(base))
		}

		// Edit format
		if format := cfg.GetStringOption("edit_format", ""); format != "" {
			opts = append(opts, WithEditFormat(format))
		}

		// Boolean flags
		if cfg.GetBoolOption("no_git", false) {
			opts = append(opts, WithNoGit())
		}
		if cfg.GetBoolOption("no_auto_commits", false) {
			opts = append(opts, WithNoAutoCommits())
		}
		if cfg.GetBoolOption("no_stream", false) {
			opts = append(opts, WithNoStream())
		}
		if cfg.GetBoolOption("dry_run", false) {
			opts = append(opts, WithDryRun())
		}
		if cfg.GetBoolOption("yes_always", false) {
			opts = append(opts, WithYesAlways())
		}

		// File lists
		if files := cfg.GetStringSliceOption("editable_files"); len(files) > 0 {
			opts = append(opts, WithEditableFiles(files))
		}
		if files := cfg.GetStringSliceOption("read_only_files"); len(files) > 0 {
			opts = append(opts, WithReadOnlyFiles(files))
		}
	}

	// AiderCLI implements provider.Client directly
	return NewAiderCLI(opts...), nil
}
