package pmbench

import (
	"testing"
)

func TestCalculate_HappyPath(t *testing.T) {
	t.Parallel()
	
	// Set up deterministic input (e.g. 100 values from 1 to 100)
	samples := make([]int64, 100)
	for i := 0; i < 100; i++ {
		// Insert in reverse to test sorting
		samples[i] = int64(100 - i)
	}

	stats := Calculate(samples)

	if stats.Samples != 100 {
		t.Errorf("expected 100 samples, got %d", stats.Samples)
	}
	if stats.Min != 1 {
		t.Errorf("expected min 1, got %d", stats.Min)
	}
	if stats.Max != 100 {
		t.Errorf("expected max 100, got %d", stats.Max)
	}
	// P50 at index 50 -> value 51
	if stats.P50 != 51 {
		t.Errorf("expected p50 51, got %d", stats.P50)
	}
	// P95 at index 95 -> value 96
	if stats.P95 != 96 {
		t.Errorf("expected p95 96, got %d", stats.P95)
	}
	// P99 at index 99 -> value 100
	if stats.P99 != 100 {
		t.Errorf("expected p99 100, got %d", stats.P99)
	}
	
	// Mean of 1..100 = 5050 / 100 = 50
	if stats.Mean != 50 {
		t.Errorf("expected mean 50, got %d", stats.Mean)
	}
}

func TestCalculate_EdgeCase_Empty(t *testing.T) {
	t.Parallel()
	
	stats := Calculate([]int64{})
	if stats.Samples != 0 {
		t.Errorf("expected 0 samples for empty slice, got %d", stats.Samples)
	}
}

func TestCalculate_EdgeCase_SingleValue(t *testing.T) {
	t.Parallel()
	
	stats := Calculate([]int64{42})
	if stats.Samples != 1 {
		t.Fatalf("expected 1 sample, got %d", stats.Samples)
	}
	if stats.Min != 42 || stats.Max != 42 || stats.Mean != 42 || stats.P50 != 42 {
		t.Errorf("expected all stats to be 42, got %+v", stats)
	}
}

func TestCalculate_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	
	input := []int64{3, 1, 2}
	Calculate(input)
	
	if input[0] != 3 || input[1] != 1 || input[2] != 2 {
		t.Errorf("Calculate mutated input slice: %v", input)
	}
}
