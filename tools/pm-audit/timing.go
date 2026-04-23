// Package pmaudit provides tools for parsing and analyzing system logs
// to generate execution time summaries.
//
// Adapted from openclaw/scripts/ci-run-timings.mjs.
package pmaudit

import (
	"encoding/json"
	"sort"
	"time"
)

// JobTiming represents a single parsed job execution with its latency.
type JobTiming struct {
	Name     string
	Duration time.Duration
}

// ParseTimings consumes JSON lines (e.g. from systemd journal) and computes
// the parsed timings. It ignores non-JSON lines and lines lacking expected fields.
// Expected JSON format per line: {"JOB": "name", "DURATION_MS": 1500}
func ParseTimings(jsonLines [][]byte) []JobTiming {
	var timings []JobTiming

	for _, line := range jsonLines {
		if len(line) == 0 {
			continue
		}

		var entry struct {
			Job        string `json:"JOB"`
			DurationMs int64  `json:"DURATION_MS"`
		}

		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip lines that don't match the format or aren't JSON
			continue
		}

		if entry.Job != "" && entry.DurationMs > 0 {
			timings = append(timings, JobTiming{
				Name:     entry.Job,
				Duration: time.Duration(entry.DurationMs) * time.Millisecond,
			})
		}
	}

	// Sort descending by duration
	sort.Slice(timings, func(i, j int) bool {
		return timings[i].Duration > timings[j].Duration
	})

	return timings
}

// TopK returns up to the k slowest jobs from the sorted slice.
func TopK(sorted []JobTiming, k int) []JobTiming {
	if k <= 0 {
		return nil
	}
	if len(sorted) <= k {
		return sorted
	}
	return sorted[:k]
}
