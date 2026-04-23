// Package memory provides semantic and episodic memory models for PM-OS.
//
// Adapted from openclaw/extensions/memory-core/src/short-term-promotion.ts.
package memory

import "math"

// RecencyScore computes an exponential decay score based on age and half-life.
// A negative or zero half-life defaults to 7 days.
// The result is clamped between 0 and 1.
func RecencyScore(ageDays float64, halfLifeDays int) float32 {
	if halfLifeDays <= 0 {
		halfLifeDays = 7
	}
	score := math.Exp(-ageDays * math.Log(2) / float64(halfLifeDays))
	return float32(math.Max(0, math.Min(1, score)))
}

// RecallScore holds the individual sub-scores for ranking a memory.
type RecallScore struct {
	Frequency     float32
	Relevance     float32
	Diversity     float32
	Recency       float32
	Consolidation float32
}

// RecallWeights defines the relative importance of each metric in the composite score.
type RecallWeights struct {
	Frequency     float32
	Relevance     float32
	Diversity     float32
	Recency       float32
	Consolidation float32
}

// DefaultWeights are the standard openclaw tuning parameters for recall scoring.
var DefaultWeights = RecallWeights{
	Frequency:     0.20,
	Relevance:     0.25,
	Diversity:     0.15,
	Recency:       0.15,
	Consolidation: 0.25,
}

// Composite calculates the final weighted score for a memory retrieval candidate.
func (r RecallScore) Composite(w RecallWeights) float32 {
	return r.Frequency*w.Frequency +
		r.Relevance*w.Relevance +
		r.Diversity*w.Diversity +
		r.Recency*w.Recency +
		r.Consolidation*w.Consolidation
}
