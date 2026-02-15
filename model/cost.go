package model

import (
	"sync"
)

// Usage tracks token usage for a model.
type Usage struct {
	InputTokens              int
	OutputTokens             int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
	Requests                 int
}

// Add adds the given usage to this usage.
func (u *Usage) Add(other Usage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.CacheCreationInputTokens += other.CacheCreationInputTokens
	u.CacheReadInputTokens += other.CacheReadInputTokens
	u.Requests += other.Requests
}

// TotalTokens returns the total tokens used.
func (u *Usage) TotalTokens() int {
	return u.InputTokens + u.OutputTokens
}

// ModelPricing holds per-million-token pricing for a model.
type ModelPricing struct {
	InputPerMillion          float64
	OutputPerMillion         float64
	CacheCreationPerMillion  float64
	CacheReadPerMillion      float64
}

// ModelPrices contains current pricing for Claude models.
// Source: https://platform.claude.com/docs/en/about-claude/pricing
// Current generation: Opus 4.5/4.6, Sonnet 4/4.5, Haiku 4.5
// 5-minute cache writes = 1.25x base input; cache reads = 0.1x base input.
var ModelPrices = map[ModelName]ModelPricing{
	ModelOpus:   {InputPerMillion: 5.0, OutputPerMillion: 25.0, CacheCreationPerMillion: 6.25, CacheReadPerMillion: 0.50},
	ModelSonnet: {InputPerMillion: 3.0, OutputPerMillion: 15.0, CacheCreationPerMillion: 3.75, CacheReadPerMillion: 0.30},
	ModelHaiku:  {InputPerMillion: 1.0, OutputPerMillion: 5.0, CacheCreationPerMillion: 1.25, CacheReadPerMillion: 0.10},
}

// CostTracker tracks token usage and estimated costs across models.
type CostTracker struct {
	mu     sync.RWMutex
	totals map[ModelName]Usage
}

// NewCostTracker creates a new cost tracker.
func NewCostTracker() *CostTracker {
	return &CostTracker{
		totals: make(map[ModelName]Usage),
	}
}

// Record adds a usage record for the given model.
func (t *CostTracker) Record(model ModelName, input, output int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	u := t.totals[model]
	u.InputTokens += input
	u.OutputTokens += output
	u.Requests++
	t.totals[model] = u
}

// RecordUsage adds a usage record for the given model.
func (t *CostTracker) RecordUsage(model ModelName, usage Usage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	u := t.totals[model]
	u.Add(usage)
	t.totals[model] = u
}

// Usage returns the usage for a specific model.
func (t *CostTracker) Usage(model ModelName) Usage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totals[model]
}

// Summary returns a copy of all usage totals.
func (t *CostTracker) Summary() map[ModelName]Usage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[ModelName]Usage, len(t.totals))
	for k, v := range t.totals {
		result[k] = v
	}
	return result
}

// TotalUsage returns aggregated usage across all models.
func (t *CostTracker) TotalUsage() Usage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var total Usage
	for _, u := range t.totals {
		total.Add(u)
	}
	return total
}

// usageCost calculates the cost of a usage record against a pricing model.
func usageCost(usage Usage, prices ModelPricing) float64 {
	inputCost := float64(usage.InputTokens) / 1_000_000 * prices.InputPerMillion
	outputCost := float64(usage.OutputTokens) / 1_000_000 * prices.OutputPerMillion
	cacheCreateCost := float64(usage.CacheCreationInputTokens) / 1_000_000 * prices.CacheCreationPerMillion
	cacheReadCost := float64(usage.CacheReadInputTokens) / 1_000_000 * prices.CacheReadPerMillion
	return inputCost + outputCost + cacheCreateCost + cacheReadCost
}

// EstimatedCost calculates the estimated cost based on current pricing.
func (t *CostTracker) EstimatedCost() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var total float64
	for model, usage := range t.totals {
		prices, ok := ModelPrices[model]
		if !ok {
			prices, ok = ModelPrices[NormalizeModelName(string(model))]
			if !ok {
				continue
			}
		}
		total += usageCost(usage, prices)
	}
	return total
}

// EstimatedCostByModel returns the estimated cost for each model.
func (t *CostTracker) EstimatedCostByModel() map[ModelName]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[ModelName]float64, len(t.totals))
	for model, usage := range t.totals {
		prices, ok := ModelPrices[model]
		if !ok {
			prices, ok = ModelPrices[NormalizeModelName(string(model))]
			if !ok {
				continue
			}
		}
		result[model] = usageCost(usage, prices)
	}
	return result
}

// Reset clears all tracked usage.
func (t *CostTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.totals = make(map[ModelName]Usage)
}
