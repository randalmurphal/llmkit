// Package model provides model selection, escalation chains, and cost tracking.
//
// The package implements a tiered model selection strategy:
//   - Thinking tier (opus): Complex reasoning, architecture, risk assessment
//   - Default tier (sonnet): Implementation, review, general tasks
//   - Fast tier (haiku): Search, simple transforms, high-volume
//
// This package is designed to be task-type agnostic. Higher-level packages
// (like devflow) can define their own task types and map them to model tiers.
package model

import "strings"

// ModelName represents a normalized model family name.
type ModelName string

// Claude model family constants.
const (
	ModelOpus   ModelName = "opus"
	ModelSonnet ModelName = "sonnet"
	ModelHaiku  ModelName = "haiku"
)

// Codex model family constants (agentic coding).
const (
	ModelCodex      ModelName = "codex"       // Standard codex (gpt-5.x-codex)
	ModelCodexSpark ModelName = "codex-spark"  // Fast codex (gpt-5.3-codex-spark)
	ModelCodexMini  ModelName = "codex-mini"   // Small/cheap codex (gpt-5.x-codex-mini)
)

// GPT model family constants (general-purpose OpenAI).
const (
	ModelGPT     ModelName = "gpt"      // Standard GPT (gpt-5, gpt-5.1, gpt-5.2, gpt-5.3)
	ModelGPTMini ModelName = "gpt-mini" // Small/cheap GPT (gpt-5-mini, gpt-5-nano)
	ModelGPTPro  ModelName = "gpt-pro"  // High-capability GPT (gpt-5-pro, gpt-5.2-pro)
)

// Tier represents a model capability tier.
type Tier int

// Tier constants representing model capability levels.
const (
	TierFast Tier = iota
	TierDefault
	TierThinking
)

// String returns the tier name.
func (t Tier) String() string {
	switch t {
	case TierFast:
		return "fast"
	case TierDefault:
		return "default"
	case TierThinking:
		return "thinking"
	default:
		return "unknown"
	}
}

// TierForModel returns the tier for a given model.
func TierForModel(model ModelName) Tier {
	switch NormalizeModelName(string(model)) {
	case ModelOpus, ModelGPTPro:
		return TierThinking
	case ModelHaiku, ModelCodexSpark, ModelCodexMini, ModelGPTMini:
		return TierFast
	default:
		return TierDefault
	}
}

// NormalizeModelName converts a full model identifier to its family alias.
// For example, "claude-sonnet-4-20250514" becomes "sonnet",
// "claude-opus-4-5-20251101" becomes "opus", and
// "gpt-5.3-codex-spark" becomes "codex-spark".
// If the name is already a family alias or doesn't match any known pattern,
// it is returned as-is.
func NormalizeModelName(name string) ModelName {
	switch ModelName(name) {
	case ModelOpus, ModelSonnet, ModelHaiku,
		ModelCodex, ModelCodexSpark, ModelCodexMini,
		ModelGPT, ModelGPTMini, ModelGPTPro:
		return ModelName(name)
	}
	lower := strings.ToLower(name)

	// Claude models
	if strings.Contains(lower, "opus") {
		return ModelOpus
	}
	if strings.Contains(lower, "sonnet") {
		return ModelSonnet
	}
	if strings.Contains(lower, "haiku") {
		return ModelHaiku
	}

	// Codex models (order matters: check specific patterns first)
	if strings.Contains(lower, "codex-spark") || strings.Contains(lower, "codex_spark") {
		return ModelCodexSpark
	}
	if strings.Contains(lower, "codex-mini") || strings.Contains(lower, "codex_mini") {
		return ModelCodexMini
	}
	// Match "codex" but not "opencode" (another CLI tool)
	if strings.Contains(lower, "codex") && !strings.Contains(lower, "opencode") {
		return ModelCodex
	}

	// GPT models (check after codex since codex models also contain "gpt")
	if isGPTModel(lower) {
		if strings.Contains(lower, "-pro") {
			return ModelGPTPro
		}
		if strings.Contains(lower, "-mini") || strings.Contains(lower, "-nano") {
			return ModelGPTMini
		}
		return ModelGPT
	}

	return ModelName(name)
}

// isGPTModel returns true if the lowercase name matches a GPT-5+ model pattern.
// Matches "gpt-5", "gpt-5.1", "gpt-5.2", "gpt-5.3", etc.
// Does NOT match older models like "gpt-4o" or "gpt-3.5-turbo".
func isGPTModel(lower string) bool {
	return strings.HasPrefix(lower, "gpt-5")
}
