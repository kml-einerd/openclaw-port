package recipe

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockExecutor struct {
	FailuresBeforeSuccess int
	Calls                 []string
}

func (m *MockExecutor) Execute(ctx context.Context, stepID string, model string) (string, error) {
	m.Calls = append(m.Calls, model)
	if len(m.Calls) <= m.FailuresBeforeSuccess {
		return "", errors.New("simulated model failure")
	}
	return "success", nil
}

func TestExecuteWithFallback_SuccessFirstTry(t *testing.T) {
	t.Parallel()

	exec := &MockExecutor{FailuresBeforeSuccess: 0}
	res, err := ExecuteWithFallback(context.Background(), "step1", "haiku", []string{"haiku", "sonnet"}, exec)

	assert.NoError(t, err)
	assert.Equal(t, "success", res)
	assert.Len(t, exec.Calls, 1)
	assert.Equal(t, "haiku", exec.Calls[0])
}

func TestExecuteWithFallback_SuccessAfterFallback(t *testing.T) {
	t.Parallel()

	exec := &MockExecutor{FailuresBeforeSuccess: 1}
	res, err := ExecuteWithFallback(context.Background(), "step1", "haiku", []string{"haiku", "sonnet", "opus"}, exec)

	assert.NoError(t, err)
	assert.Equal(t, "success", res)
	assert.Len(t, exec.Calls, 2)
	assert.Equal(t, "haiku", exec.Calls[0])
	assert.Equal(t, "sonnet", exec.Calls[1])
}

func TestExecuteWithFallback_AllFail(t *testing.T) {
	t.Parallel()

	exec := &MockExecutor{FailuresBeforeSuccess: 5}
	res, err := ExecuteWithFallback(context.Background(), "step1", "haiku", []string{"haiku", "sonnet"}, exec)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 fallbacks")
	assert.Equal(t, "", res)
	assert.Len(t, exec.Calls, 2)
}

func TestExecuteWithFallback_ContextCancel(t *testing.T) {
	t.Parallel()

	exec := &MockExecutor{FailuresBeforeSuccess: 5}
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := ExecuteWithFallback(ctx, "step1", "haiku", []string{"haiku", "sonnet"}, exec)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestExecuteWithFallback_ContextCancelDuringWait(t *testing.T) {
	t.Parallel()
	
	// Create an executor that takes time and fails, so it triggers the sleep
	exec := &MockExecutor{FailuresBeforeSuccess: 5}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	
	_, err := ExecuteWithFallback(ctx, "step1", "haiku", []string{"haiku", "sonnet"}, exec)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
