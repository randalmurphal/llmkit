# model

**Model selection, cost tracking, and escalation chains.** Task-agnostic utilities for choosing LLM models and tracking usage costs.

---

## Package Contents

| File | Purpose |
|------|---------|
| `names.go` | Model names and tier definitions |
| `selector.go` | Task-based model selection |
| `escalation.go` | Model escalation chains |
| `cost.go` | Usage and cost tracking |

---

## Key Types

### Model Names

```go
type ModelName string

// Claude
const (
    ModelOpus   ModelName = "opus"
    ModelSonnet ModelName = "sonnet"
    ModelHaiku  ModelName = "haiku"
)

// Codex (agentic coding)
const (
    ModelCodex      ModelName = "codex"        // gpt-5.x-codex
    ModelCodexSpark ModelName = "codex-spark"   // gpt-5.3-codex-spark (fast)
    ModelCodexMini  ModelName = "codex-mini"    // gpt-5.x-codex-mini (cheap)
)

// GPT (general-purpose)
const (
    ModelGPT     ModelName = "gpt"       // gpt-5, gpt-5.1, gpt-5.2, gpt-5.3
    ModelGPTMini ModelName = "gpt-mini"  // gpt-5-mini, gpt-5-nano
    ModelGPTPro  ModelName = "gpt-pro"   // gpt-5-pro, gpt-5.2-pro
)
```

### Tiers

```go
type Tier int

const (
    TierFast     Tier = iota  // Simple, high-volume tasks
    TierDefault               // General-purpose tasks
    TierThinking              // Complex reasoning tasks
)
```

### Selector

```go
type Selector struct {
    // Configuration (private)
}

selector := NewSelector(
    WithThinkingModel(ModelOpus),
    WithDefaultModel(ModelSonnet),
    WithFastModel(ModelHaiku),
)

model := selector.SelectForTier(TierThinking)  // ModelOpus
```

### CostTracker

```go
type CostTracker struct {
    // Usage data (private)
}

tracker := NewCostTracker()
tracker.Record(ModelSonnet, 1000, 500)  // input, output tokens
cost := tracker.EstimatedCost()  // USD
```

### EscalationChain

```go
type EscalationChain struct {
    Models      []ModelName
    MaxAttempts int
}

// Predefined chains
var DefaultEscalation = EscalationChain{
    Models:      []ModelName{ModelSonnet, ModelOpus},
    MaxAttempts: 3,
}
```

---

## Usage Examples

### Model Selection by Tier

```go
selector := model.NewSelector(
    model.WithThinkingModel(model.ModelOpus),
    model.WithDefaultModel(model.ModelSonnet),
    model.WithFastModel(model.ModelHaiku),
)

// Select based on task complexity
m := selector.SelectForTier(model.TierThinking)  // complex reasoning
m = selector.SelectForTier(model.TierDefault)    // general tasks
m = selector.SelectForTier(model.TierFast)       // simple/bulk tasks
```

### Task-Based Selection

```go
// Define task types (in your application)
type TaskType string
const (
    TaskReview    TaskType = "review"
    TaskImplement TaskType = "implement"
    TaskSearch    TaskType = "search"
)

// Configure selector with task mappings
selector := model.NewSelector(
    model.WithDefaults(map[any]model.ModelName{
        TaskReview:    model.ModelOpus,
        TaskImplement: model.ModelSonnet,
        TaskSearch:    model.ModelHaiku,
    }),
)

// Select by task
m := selector.Select(TaskReview)  // ModelOpus
```

### Global Override

```go
// Force a specific model for all tasks (e.g., for debugging)
selector := model.NewSelector()
overridden := selector.WithGlobal(model.ModelHaiku)

m := overridden.SelectForTier(model.TierThinking)  // Still returns Haiku
```

### Cost Tracking

```go
tracker := model.NewCostTracker()

// Record usage
tracker.Record(model.ModelSonnet, 5000, 1000)   // 5k input, 1k output
tracker.Record(model.ModelSonnet, 3000, 500)    // Another request
tracker.Record(model.ModelHaiku, 10000, 2000)   // Haiku request

// Get total cost estimate
total := tracker.EstimatedCost()
fmt.Printf("Total cost: $%.4f\n", total)

// Get breakdown by model
costs := tracker.EstimatedCostByModel()
for model, cost := range costs {
    fmt.Printf("%s: $%.4f\n", model, cost)
}

// Get usage summary
summary := tracker.Summary()
for model, usage := range summary {
    fmt.Printf("%s: %d requests, %d tokens\n",
        model, usage.Requests, usage.TotalTokens())
}
```

### Model Escalation

```go
// Start with sonnet, escalate to opus on failure
chain := &model.EscalationChain{
    Models:      []model.ModelName{model.ModelSonnet, model.ModelOpus},
    MaxAttempts: 3,
}

state := model.NewEscalationState(chain, model.ModelSonnet)

for !state.Exhausted() {
    resp, err := client.Complete(ctx, request, state.CurrentModel)
    if err == nil {
        return resp, nil
    }

    if !state.RecordFailure(err) {
        return nil, fmt.Errorf("all models failed: %w", state.LastError)
    }
    // Loop continues with potentially escalated model
}
```

### Context Integration

```go
// Store selector in context
ctx := model.NewContext(ctx, selector)

// Retrieve later
selector := model.FromContext(ctx)
m := selector.SelectForTier(model.TierDefault)
```

---

## Pricing

### Claude (as of 2026)

| Model | Input/1M | Output/1M | Cache Create/1M | Cache Read/1M |
|-------|----------|-----------|-----------------|---------------|
| Opus | $5.00 | $25.00 | $6.25 | $0.50 |
| Sonnet | $3.00 | $15.00 | $3.75 | $0.30 |
| Haiku | $1.00 | $5.00 | $1.25 | $0.10 |

### Codex (as of 2026)

| Model | Input/1M | Output/1M | Cache Read/1M |
|-------|----------|-----------|---------------|
| codex (gpt-5.3-codex) | $1.75 | $14.00 | $0.175 |
| codex-mini (gpt-5.1-codex-mini) | $0.25 | $2.00 | $0.025 |
| codex-spark (gpt-5.3-codex-spark) | N/A (research preview, no API pricing yet) | | |

### GPT General-Purpose (as of 2026)

| Model | Input/1M | Output/1M | Cache Read/1M |
|-------|----------|-----------|---------------|
| gpt (gpt-5.2) | $1.75 | $14.00 | $0.175 |
| gpt-mini (gpt-5-mini) | $0.25 | $2.00 | $0.025 |
| gpt-pro (gpt-5-pro) | $15.00 | $120.00 | — |

Prices are used by `CostTracker.EstimatedCost()`. Full model names (e.g., `gpt-5.3-codex`) are normalized to family names for price lookup.

---

## Reasoning Effort (Codex/GPT Models)

Reasoning effort is set via the codex package, not the model package:

```go
// Client-level default
client := codex.NewCodexCLI(codex.WithReasoningEffort("high"))

// Per-request override
req := codex.CompletionRequest{
    ConfigOverrides: map[string]any{"model_reasoning_effort": "medium"},
}
```

Valid values: `minimal`, `low`, `medium`, `high`, `xhigh`

---

## Notes

- Model names are family aliases ("opus", "sonnet", "codex", "gpt", "gpt-pro") not full IDs
- `NormalizeModelName()` maps full model strings to family aliases
- Codex models (contain "codex") are checked before GPT patterns
- `gpt-4o` and older models are NOT matched — only `gpt-5+` is normalized
- Task-type agnostic: define your own task types
- Thread-safe: Selector and CostTracker are concurrent-safe
- Prices may be updated in future versions
