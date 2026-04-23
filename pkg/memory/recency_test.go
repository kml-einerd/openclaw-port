package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecencyScore(t *testing.T) {
	t.Parallel()

	// 0 days age should be 1.0
	score0 := RecencyScore(0, 7)
	assert.InDelta(t, 1.0, score0, 0.001)

	// Half-life (7 days) should yield 0.5
	score7 := RecencyScore(7, 7)
	assert.InDelta(t, 0.5, score7, 0.001)

	// Two half-lives (14 days) should yield 0.25
	score14 := RecencyScore(14, 7)
	assert.InDelta(t, 0.25, score14, 0.001)

	// Extremely old should approach 0
	scoreOld := RecencyScore(1000, 7)
	assert.InDelta(t, 0.0, scoreOld, 0.001)
	
	// Negative age shouldn't exceed 1.0 because of math.Min
	scoreNeg := RecencyScore(-5, 7)
	assert.InDelta(t, 1.0, scoreNeg, 0.001)
	
	// Default half-life of 7 when <= 0
	scoreZeroHL := RecencyScore(7, 0)
	assert.InDelta(t, 0.5, scoreZeroHL, 0.001)
}

func TestCompositeScore(t *testing.T) {
	t.Parallel()

	score := RecallScore{
		Frequency:     1.0,
		Relevance:     1.0,
		Diversity:     0.5,
		Recency:       0.8,
		Consolidation: 0.9,
	}

	w := DefaultWeights // F:0.2, Rel:0.25, D:0.15, Rec:0.15, C:0.25

	// (1.0*0.2) + (1.0*0.25) + (0.5*0.15) + (0.8*0.15) + (0.9*0.25)
	// 0.20 + 0.25 + 0.075 + 0.120 + 0.225 = 0.87
	
	composite := score.Composite(w)
	assert.InDelta(t, 0.87, composite, 0.001)
}
