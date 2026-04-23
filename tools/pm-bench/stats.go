// Package pmbench provides utilities for benchmarking PM-OS components,
// including statistical calculators for latency and throughput metrics.
//
// Adapted from openclaw/scripts/bench-cli-startup.ts.
package pmbench

import "sort"

// Stats represents the statistical percentiles and averages of a dataset.
type Stats struct {
	P50     int64
	P95     int64
	P99     int64
	Min     int64
	Max     int64
	Mean    int64
	Samples int
}

// Calculate computes performance percentiles and averages from a slice of raw integers (e.g., latencies in ns).
// Returns an empty Stats struct if the input slice is empty.
func Calculate(rawNs []int64) Stats {
	n := len(rawNs)
	if n == 0 {
		return Stats{}
	}

	// Create a copy to avoid mutating the input slice
	s := make([]int64, n)
	copy(s, rawNs)

	// Sort the slice in ascending order
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })

	var sum int64
	for _, v := range s {
		sum += v
	}

	return Stats{
		P50:     s[(n*50)/100],
		P95:     s[(n*95)/100],
		P99:     s[(n*99)/100],
		Min:     s[0],
		Max:     s[n-1],
		Mean:    sum / int64(n),
		Samples: n,
	}
}
