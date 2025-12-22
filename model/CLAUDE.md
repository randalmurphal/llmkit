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

const (
    ModelOpus   ModelName = "opus"
    ModelSonnet ModelName = "sonnet"
    ModelHaiku  ModelName = "haiku"
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

## Pricing (as of 2025)

| Model | Input (per 1M) | Output (per 1M) |
|-------|----------------|-----------------|
| Opus | $15.00 | $75.00 |
| Sonnet | $3.00 | $15.00 |
| Haiku | $0.25 | $1.25 |

Prices are used by `CostTracker.EstimatedCost()`.

---

## Notes

- Model names are abstract ("opus", "sonnet", "haiku") not full IDs
- Task-type agnostic: define your own task types
- Thread-safe: Selector and CostTracker are concurrent-safe
- Prices may be updated in future versions
