// Package llmkit provides utilities for working with Large Language Models.
//
// llmkit is a standalone toolkit extracted from flowgraph, designed to be
// imported à la carte. Each subpackage can be used independently:
//
//   - claude: Claude CLI wrapper with OAuth credential management
//   - template: Prompt template rendering with {{variable}} syntax
//   - tokens: Token counting and budget management
//   - parser: Extract JSON, YAML, and code blocks from LLM responses
//   - truncate: Token-aware text truncation strategies
//   - model: Model selection, cost tracking, and escalation chains
//
// # Quick Start
//
// Token counting:
//
//	import "github.com/randalmurphal/llmkit/tokens"
//	counter := tokens.NewEstimatingCounter()
//	count := counter.Count("Hello, World!")
//
// Template rendering:
//
//	import "github.com/randalmurphal/llmkit/template"
//	engine := template.NewEngine()
//	result, _ := engine.Render("Hello {{name}}", map[string]any{"name": "World"})
//
// Claude CLI:
//
//	import "github.com/randalmurphal/llmkit/claude"
//	client := claude.NewCLI()
//	resp, _ := client.Complete(ctx, claude.CompletionRequest{...})
//
// # Design Philosophy
//
// llmkit follows these principles:
//
//   - Zero external dependencies (stdlib only)
//   - Each package usable independently
//   - Stable, semver-friendly API
//   - Sensible defaults with full configurability
//   - Interfaces for extensibility, concrete types for simplicity
package llmkit
