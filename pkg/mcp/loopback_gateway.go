package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ToolSchema represents an MCP tool definition.
type ToolSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SessionCacheKey uniquely identifies an MCP caching boundary.
type SessionCacheKey struct {
	SessionKey string
	Provider   string
	AccountID  string
}

type cacheEntry struct {
	schemas   []ToolSchema
	expiresAt time.Time
}

// LoopbackGateway handles caching and routing of MCP tools within the same process.
//
// Adapted from openclaw/src/mcp/loopback/ and src/gateway/mcp-http.ts.
type LoopbackGateway struct {
	cache sync.Map
	ttl   time.Duration
}

// NewLoopbackGateway initializes an MCP Loopback Gateway with a specific TTL.
// If ttl is <= 0, it defaults to 30 seconds.
func NewLoopbackGateway(ttl time.Duration) *LoopbackGateway {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &LoopbackGateway{ttl: ttl}
}

// CacheTools saves a list of schemas against a session boundary.
func (g *LoopbackGateway) CacheTools(key SessionCacheKey, schemas []ToolSchema) {
	g.cache.Store(key, cacheEntry{
		schemas:   schemas,
		expiresAt: time.Now().Add(g.ttl),
	})
}

// GetCachedTools retrieves schemas if they haven't expired.
func (g *LoopbackGateway) GetCachedTools(key SessionCacheKey) ([]ToolSchema, bool) {
	val, ok := g.cache.Load(key)
	if !ok {
		return nil, false
	}

	entry := val.(cacheEntry)
	if time.Now().After(entry.expiresAt) {
		g.cache.Delete(key)
		return nil, false
	}

	return entry.schemas, true
}

// FetchTools abstractly resolves tools. If cached, it returns them immediately.
// Otherwise it calls the fetchFn to retrieve them, caches the result, and returns it.
func (g *LoopbackGateway) FetchTools(ctx context.Context, key SessionCacheKey, fetchFn func(context.Context) ([]ToolSchema, error)) ([]ToolSchema, error) {
	if schemas, ok := g.GetCachedTools(key); ok {
		return schemas, nil
	}

	schemas, err := fetchFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch loopback tools: %w", err)
	}

	g.CacheTools(key, schemas)
	return schemas, nil
}
