// Package llmkit provides utilities for working with Large Language Models.
//
// llmkit is a standalone toolkit extracted from flowgraph, designed to be
// imported à la carte. Each subpackage can be used independently:
//
//   - llmkit: Unified root API, registry, model selection, and pricing
//   - claude: Claude CLI wrapper with OAuth credential management
//   - codex: Codex CLI wrapper with headless exec/stream support
//   - claudeconfig: Claude local config parsing
//   - codexconfig: Codex local config, hooks, skills, plugins, and custom agents
//   - env: Scoped hook, MCP, env var, and tempfile lifecycle helpers
//   - worktree: Git worktree lifecycle helpers
//   - providers: Convenience blank imports for registry registration
//   - template: Prompt template rendering with {{variable}} syntax
//   - tokens: Token counting and budget management
//   - parser: Extract JSON, YAML, and code blocks from LLM responses
//   - truncate: Token-aware text truncation strategies
//
// # Quick Start
//
// Token counting:
//
//	import "github.com/randalmurphal/llmkit/v2/tokens"
//	counter := tokens.NewEstimatingCounter()
//	count := counter.Count("Hello, World!")
//
// Template rendering:
//
//	import "github.com/randalmurphal/llmkit/v2/template"
//	engine := template.NewEngine()
//	result, _ := engine.Render("Hello {{name}}", map[string]any{"name": "World"})
//
// Unified client:
//
//	import (
//	    "github.com/randalmurphal/llmkit/v2"
//	    _ "github.com/randalmurphal/llmkit/v2/providers"
//	)
//	client, _ := llmkit.New("codex", llmkit.Config{Provider: "codex", Model: "gpt-5-codex"})
//	resp, _ := client.Complete(ctx, llmkit.Request{...})
//
// # Design Philosophy
//
// llmkit follows these principles:
//
//   - Each package usable independently
//   - Clear V2 module boundaries rooted at github.com/randalmurphal/llmkit/v2
//   - Sensible defaults with full configurability
//   - Interfaces for extensibility, concrete types for simplicity
package llmkit
