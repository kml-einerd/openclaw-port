// Package memory provides semantic and episodic memory models for PM-OS.
package memory

import "sort"

// ScoredMemory represents a memory entry along with its computed recall score.
type ScoredMemory struct {
	ID    string
	Score float32
}

// MemoryScoringData represents the raw telemetry needed to rank a memory.
type MemoryScoringData struct {
	ID            string
	AgeDays       float64
	Frequency     float32
	Relevance     float32
	Diversity     float32
	Consolidation float32
}

// ComputeScore calculates the recall score for a given memory entry.
func ComputeScore(m MemoryScoringData, halfLifeDays int, w RecallWeights) float32 {
	r := RecallScore{
		Frequency:     m.Frequency,
		Relevance:     m.Relevance,
		Diversity:     m.Diversity,
		Recency:       RecencyScore(m.AgeDays, halfLifeDays),
		Consolidation: m.Consolidation,
	}
	return r.Composite(w)
}

// RankMemories evaluates and sorts a batch of memories descending by their
// composite recall score. It returns up to topK results.
func RankMemories(memories []MemoryScoringData, halfLifeDays int, w RecallWeights, topK int) []ScoredMemory {
	scored := make([]ScoredMemory, 0, len(memories))
	for _, m := range memories {
		scored = append(scored, ScoredMemory{
			ID:    m.ID,
			Score: ComputeScore(m, halfLifeDays, w),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		// Stable descending sort
		return scored[i].Score > scored[j].Score
	})

	if topK > 0 && len(scored) > topK {
		return scored[:topK]
	}
	return scored
}
