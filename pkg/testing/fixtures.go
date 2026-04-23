package pmtest

import (
	"context"

	"github.com/stretchr/testify/assert"
)

// MockExecutor is a reusable mock for testing step execution logic without real LLM calls.
type MockExecutor struct {
	ExecuteFn func(ctx context.Context, stepID string, model string) (string, error)
	Calls     []struct {
		StepID string
		Model  string
	}
}

// Execute records the call details and executes the custom ExecuteFn if provided.
// Otherwise it returns a dummy "mock output".
func (m *MockExecutor) Execute(ctx context.Context, stepID string, model string) (string, error) {
	m.Calls = append(m.Calls, struct {
		StepID string
		Model  string
	}{StepID: stepID, Model: model})

	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, stepID, model)
	}
	return "mock output", nil
}

// AssertCalledTimes asserts that the Execute method was called exactly n times.
func (m *MockExecutor) AssertCalledTimes(t assert.TestingT, n int) {
	assert.Len(t, m.Calls, n)
}
