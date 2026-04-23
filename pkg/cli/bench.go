package cli

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"openclaw-port/tools/pm-bench"
)

// BenchTarget specifies what to benchmark.
type BenchTarget string

const (
	BenchStartup  BenchTarget = "startup"
	BenchRecipe   BenchTarget = "recipe"
	BenchExecutor BenchTarget = "executor"
)

// BenchOptions configures the bench command.
type BenchOptions struct {
	Target BenchTarget
	Runs   int
	Binary string
	Budget time.Duration
}

// BenchResult contains the outcome of a benchmark run.
type BenchResult struct {
	Stats  pmbench.Stats
	Passed bool
	Detail string
}

// RunBench executes a benchmark suite for the specified target.
//
// Adapted from openclaw §5.3 pm-cli bench.
func RunBench(ctx context.Context, opts BenchOptions) (*BenchResult, error) {
	if opts.Runs <= 0 {
		opts.Runs = 10
	}

	switch opts.Target {
	case BenchStartup:
		return benchStartup(ctx, opts)
	default:
		return nil, fmt.Errorf("unknown bench target: %s", opts.Target)
	}
}

func benchStartup(ctx context.Context, opts BenchOptions) (*BenchResult, error) {
	var latencies []int64

	for i := 0; i < opts.Runs; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		start := time.Now()
		cmd := exec.CommandContext(ctx, opts.Binary, "--dry-run")
		_ = cmd.Run()
		latencies = append(latencies, time.Since(start).Nanoseconds())
	}

	stats := pmbench.Calculate(latencies)
	p95 := time.Duration(stats.P95)

	passed := true
	detail := fmt.Sprintf("p95=%v within budget %v", p95, opts.Budget)
	if opts.Budget > 0 && p95 > opts.Budget {
		passed = false
		detail = fmt.Sprintf("p95=%v EXCEEDS budget %v", p95, opts.Budget)
	}

	return &BenchResult{Stats: stats, Passed: passed, Detail: detail}, nil
}
