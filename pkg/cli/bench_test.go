package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBench_Startup(t *testing.T) {
	t.Parallel()

	// Use "cmd" on Windows or "echo" on Linux — a fast binary that exits immediately
	result, err := RunBench(context.Background(), BenchOptions{
		Target: BenchStartup,
		Runs:   3,
		Binary: "cmd",
		Budget: 5 * time.Second,
	})

	require.NoError(t, err)
	assert.True(t, result.Passed, "budget should pass for a trivial command")
	assert.Equal(t, 3, result.Stats.Samples)
}

func TestRunBench_UnknownTarget(t *testing.T) {
	t.Parallel()

	_, err := RunBench(context.Background(), BenchOptions{Target: "unknown"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown bench target")
}
