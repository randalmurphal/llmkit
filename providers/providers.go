// Package providers registers all known LLM CLI providers.
// Import this package to make all providers available via provider.New():
//
//	import _ "github.com/randalmurphal/llmkit/providers"
package providers

import (
	_ "github.com/randalmurphal/llmkit/aider"
	_ "github.com/randalmurphal/llmkit/claude"
	_ "github.com/randalmurphal/llmkit/codex"
	_ "github.com/randalmurphal/llmkit/continue"
	_ "github.com/randalmurphal/llmkit/gemini"
	_ "github.com/randalmurphal/llmkit/local"
	_ "github.com/randalmurphal/llmkit/opencode"
)
