// Command startup measures the cold-start latency of PM-OS binaries
// and validates it against a configurable budget.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"openclaw-port/tools/pm-bench"
)

func main() {
	budgetStr := flag.String("budget", "200ms", "Maximum allowed p95 cold-start latency")
	runs := flag.Int("runs", 10, "Number of times to run the binary")
	binary := flag.String("bin", "./pm-api", "Binary to execute")
	flag.Parse()

	budget, err := time.ParseDuration(*budgetStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid budget format '%s': %v\n", *budgetStr, err)
		os.Exit(1)
	}

	var latencies []int64
	for i := 0; i < *runs; i++ {
		start := time.Now()
		
		cmd := exec.Command(*binary, "--dry-run")
		_ = cmd.Run()
		
		latencies = append(latencies, time.Since(start).Nanoseconds())
	}

	stats := pmbench.Calculate(latencies)
	p95 := time.Duration(stats.P95)

	fmt.Printf("Startup Benchmark (%d runs of %s):\n", *runs, *binary)
	fmt.Printf("  p50:  %v\n", time.Duration(stats.P50))
	fmt.Printf("  p95:  %v\n", p95)
	fmt.Printf("  p99:  %v\n", time.Duration(stats.P99))
	fmt.Printf("  Mean: %v\n", time.Duration(stats.Mean))

	if p95 > budget {
		fmt.Fprintf(os.Stderr, "ERROR: p95 latency %v exceeds budget %v\n", p95, budget)
		os.Exit(1)
	}

	fmt.Println("PASS: Latency within budget.")
}
