# Changelog

All notable changes to this project will be documented in this file.

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
