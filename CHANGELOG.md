# Changelog

All notable changes to this project will be documented in this file.

## [2.0.0] - 2026-03-29

### Added

- Root-owned V2 API at `github.com/randalmurphal/llmkit/v2` with shared `Client`, `Config`, `Request`, `Response`, model selection, pricing, and strict typed structured output helpers.
- `codexconfig/` for Codex config, hooks, instruction hierarchy, skills, plugins, custom agents, and rules.
- `env/` for scoped hook, MCP, environment, and tempfile lifecycle management.
- `worktree/` for git worktree creation, pruning, and optional safety hooks.
- `providers/` convenience package for Claude and Codex registry registration.

### Changed

- Promoted the shared client/config/type/model surface to the repo root.
- Reduced supported runtime providers to Claude and Codex.
- Added strict structured-output plumbing with provider-aware schema handling.
- Hardened Claude and Codex session management for long-running consumer usage.

### Removed

- `aider/`, `continue/`, `gemini/`, `local/`, and `opencode/`
- The old public `provider/` and `model/` packages

## [1.0.0] - 2025-12-22

### Added

Initial release with packages extracted from flowgraph:

- **claude/** - Claude CLI wrapper with streaming support
  - `Client` interface for completions and streaming
  - `ClaudeCLI` implementation wrapping the Claude CLI
  - `MockClient` for testing
  - Credential management with OAuth support
  - Comprehensive option functions for configuration

- **tokens/** - Token counting and budget management
  - `Counter` interface with `EstimatingCounter` implementation
  - `Budget` for managing token allocation across prompt components
  - Model-specific token limits

- **template/** - Prompt template rendering
  - `Engine` with Handlebars-like syntax support
  - Built-in functions (truncate, json, upper, lower, etc.)
  - Variable extraction from templates
  - Custom function support

- **parser/** - LLM response parsing
  - JSON extraction from responses
  - Code block extraction with language detection
  - Structured response parsing

- **truncate/** - Token-aware text truncation
  - Multiple strategies: FromEnd, FromStart, FromMiddle
  - Configurable suffixes
  - Convenience functions for common operations

- **model/** - Model selection and cost tracking
  - `Selector` for tier-based model selection
  - `CostTracker` for usage tracking
  - `EscalationChain` for model escalation
  - Pricing data for Claude models
