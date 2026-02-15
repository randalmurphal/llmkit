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

// ModelName represents a Claude model name.
type ModelName string

// Model name constants for Claude models.
const (
	ModelOpus   ModelName = "opus"
	ModelSonnet ModelName = "sonnet"
	ModelHaiku  ModelName = "haiku"
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
	case ModelOpus:
		return TierThinking
	case ModelHaiku:
		return TierFast
	default:
		return TierDefault
	}
}

// NormalizeModelName converts a full model identifier to its tier alias.
// For example, "claude-sonnet-4-20250514" becomes "sonnet" and
// "claude-opus-4-5-20251101" becomes "opus". If the name is already
// a tier alias or doesn't match any known pattern, it is returned as-is.
func NormalizeModelName(name string) ModelName {
	switch ModelName(name) {
	case ModelOpus, ModelSonnet, ModelHaiku:
		return ModelName(name)
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "opus") {
		return ModelOpus
	}
	if strings.Contains(lower, "sonnet") {
		return ModelSonnet
	}
	if strings.Contains(lower, "haiku") {
		return ModelHaiku
	}
	return ModelName(name)
}
