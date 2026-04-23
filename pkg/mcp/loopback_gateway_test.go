package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoopbackGateway_Caching(t *testing.T) {
	t.Parallel()

	gw := NewLoopbackGateway(50 * time.Millisecond)
	key := SessionCacheKey{SessionKey: "s1", Provider: "test", AccountID: "a1"}

	// Initially missed
	_, ok := gw.GetCachedTools(key)
	assert.False(t, ok)

	// Fetch caches it
	calls := 0
	fetcher := func(ctx context.Context) ([]ToolSchema, error) {
		calls++
		return []ToolSchema{{Name: "tool1"}}, nil
	}

	schemas, err := gw.FetchTools(context.Background(), key, fetcher)
	assert.NoError(t, err)
	assert.Len(t, schemas, 1)
	assert.Equal(t, 1, calls)

	// Second fetch hits cache
	schemas2, err2 := gw.FetchTools(context.Background(), key, fetcher)
	assert.NoError(t, err2)
	assert.Len(t, schemas2, 1)
	assert.Equal(t, 1, calls) // Call count did not increase

	// Wait for TTL expiration
	time.Sleep(60 * time.Millisecond)

	// Third fetch misses cache and calls fetcher again
	schemas3, err3 := gw.FetchTools(context.Background(), key, fetcher)
	assert.NoError(t, err3)
	assert.Len(t, schemas3, 1)
	assert.Equal(t, 2, calls) // Call count increased
}

func TestLoopbackGateway_FetchError(t *testing.T) {
	t.Parallel()

	gw := NewLoopbackGateway(0) // Default 30s
	key := SessionCacheKey{SessionKey: "s2"}

	fetcher := func(ctx context.Context) ([]ToolSchema, error) {
		return nil, errors.New("upstream failed")
	}

	schemas, err := gw.FetchTools(context.Background(), key, fetcher)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upstream failed")
	assert.Nil(t, schemas)

	// Verify it wasn't cached
	_, ok := gw.GetCachedTools(key)
	assert.False(t, ok)
}
