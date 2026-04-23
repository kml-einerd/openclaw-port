package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeScore(t *testing.T) {
	t.Parallel()

	m := MemoryScoringData{
		ID:            "m1",
		AgeDays:       7.0, // Recency will be 0.5 with half-life 7
		Frequency:     1.0,
		Relevance:     1.0,
		Diversity:     0.5,
		Consolidation: 0.9,
	}

	// w = F:0.2, Rel:0.25, D:0.15, Rec:0.15, C:0.25
	// F: 0.20, Rel: 0.25, D: 0.075, Rec: 0.075, C: 0.225
	// Total: 0.825

	score := ComputeScore(m, 7, DefaultWeights)
	assert.InDelta(t, 0.825, score, 0.001)
}

func TestRankMemories(t *testing.T) {
	t.Parallel()

	memories := []MemoryScoringData{
		{ID: "m_old", AgeDays: 100, Relevance: 0.5, Frequency: 0.1},
		{ID: "m_new", AgeDays: 1, Relevance: 1.0, Frequency: 1.0, Consolidation: 1.0},
		{ID: "m_mid", AgeDays: 14, Relevance: 0.8, Frequency: 0.5, Consolidation: 0.5},
	}

	ranked := RankMemories(memories, 7, DefaultWeights, 0)
	
	assert.Len(t, ranked, 3)
	assert.Equal(t, "m_new", ranked[0].ID)
	assert.Equal(t, "m_mid", ranked[1].ID)
	assert.Equal(t, "m_old", ranked[2].ID)

	// Test topK truncation
	top1 := RankMemories(memories, 7, DefaultWeights, 1)
	assert.Len(t, top1, 1)
	assert.Equal(t, "m_new", top1[0].ID)
}

func TestRankMemories_Empty(t *testing.T) {
	t.Parallel()

	ranked := RankMemories(nil, 7, DefaultWeights, 5)
	assert.Len(t, ranked, 0)
}
