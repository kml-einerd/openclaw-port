package pmaudit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTimings(t *testing.T) {
	t.Parallel()

	input := [][]byte{
		[]byte(`{"JOB": "fast-job", "DURATION_MS": 50}`),
		[]byte(`{"JOB": "slow-job", "DURATION_MS": 5000}`),
		[]byte(`{"JOB": "medium-job", "DURATION_MS": 1000}`),
		[]byte(`not json at all`),
		[]byte(`{"OTHER_FIELD": "ignored"}`),
	}

	timings := ParseTimings(input)

	assert.Len(t, timings, 3)
	// Must be sorted descending
	assert.Equal(t, "slow-job", timings[0].Name)
	assert.Equal(t, 5000*time.Millisecond, timings[0].Duration)

	assert.Equal(t, "medium-job", timings[1].Name)
	assert.Equal(t, 1000*time.Millisecond, timings[1].Duration)

	assert.Equal(t, "fast-job", timings[2].Name)
	assert.Equal(t, 50*time.Millisecond, timings[2].Duration)
}

func TestTopK(t *testing.T) {
	t.Parallel()

	sorted := []JobTiming{
		{Name: "1", Duration: 5 * time.Second},
		{Name: "2", Duration: 4 * time.Second},
		{Name: "3", Duration: 3 * time.Second},
		{Name: "4", Duration: 2 * time.Second},
	}

	top2 := TopK(sorted, 2)
	assert.Len(t, top2, 2)
	assert.Equal(t, "1", top2[0].Name)
	assert.Equal(t, "2", top2[1].Name)

	top10 := TopK(sorted, 10)
	assert.Len(t, top10, 4)

	top0 := TopK(sorted, 0)
	assert.Len(t, top0, 0)
}
