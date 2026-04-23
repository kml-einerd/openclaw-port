// Package recipe provides the execution abstractions and schema for PM-OS tasks.
package recipe

import (
	"context"
	"fmt"
	"time"
)

// Executor represents the core execution engine that processes a step using a specific model.
type Executor interface {
	Execute(ctx context.Context, stepID string, model string) (string, error)
}

// ExecuteWithFallback runs a step, retrying with fallback models from the wave configuration if execution fails.
//
// Adapted from openclaw/src/agents/run-executor.ts (Pattern B).
func ExecuteWithFallback(
	ctx context.Context,
	stepID string,
	preferredModel string,
	fallbackModels []string,
	executor Executor,
) (string, error) {
	models := fallbackModels
	if len(models) == 0 {
		models = []string{preferredModel}
	}

	var lastErr error
	for attempt, model := range models {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		result, err := executor.Execute(ctx, stepID, model)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Wait before the next fallback attempt, using exponential backoff
		if attempt < len(models)-1 {
			select {
			case <-time.After(time.Duration(attempt+1) * 10 * time.Millisecond): // Scaled down for testing realism
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	return "", fmt.Errorf("step %s failed after %d fallbacks: %w", stepID, len(models), lastErr)
}
