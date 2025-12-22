// Package model provides model selection, cost tracking, and escalation chains.
//
// This package helps choose the appropriate LLM model for different task types
// and tracks token usage for cost estimation. It is designed to be task-type
// agnostic - define your own task types and map them to model tiers.
//
// # Model Selection
//
//	selector := model.NewSelector(
//	    model.WithThinkingModel(model.ModelOpus),
//	    model.WithDefaultModel(model.ModelSonnet),
//	    model.WithFastModel(model.ModelHaiku),
//	)
//	m := selector.SelectForTier(model.TierThinking)
//
// # Cost Tracking
//
//	tracker := model.NewCostTracker()
//	tracker.Record(model.ModelSonnet, 1000, 500)  // input, output tokens
//	cost := tracker.EstimatedCost()
//
// # Model Escalation
//
// Escalation chains define how to retry with more capable models:
//
//	state := model.NewEscalationState(&model.DefaultEscalation, model.ModelSonnet)
//	for !state.Exhausted() {
//	    resp, err := tryRequest(state.CurrentModel)
//	    if err == nil {
//	        return resp, nil
//	    }
//	    state.RecordFailure(err)  // May escalate to next model
//	}
package model
