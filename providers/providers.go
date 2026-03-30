// Package providers registers the supported V2 LLM CLI providers.
// Import this package to make Claude and Codex available via llmkit.New():
//
//	import _ "github.com/randalmurphal/llmkit/v2/providers"
package providers

import (
	_ "github.com/randalmurphal/llmkit/v2/claude"
	_ "github.com/randalmurphal/llmkit/v2/codex"
)
