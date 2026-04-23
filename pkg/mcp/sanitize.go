// Package mcp provides utilities for integrating Model Context Protocol tools,
// including safe name formatting and cache management.
//
// Adapted from openclaw/src/agents/pi-bundle-mcp-names.ts.
package mcp

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var nonAlphanumRe = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// NameRegistry ensures that generated tool names are unique and valid.
// It is safe for concurrent use.
type NameRegistry struct {
	used map[string]bool
	mu   sync.RWMutex
}

// NewNameRegistry creates a new NameRegistry.
func NewNameRegistry() *NameRegistry {
	return &NameRegistry{
		used: make(map[string]bool),
	}
}

func sanitize(s string) string {
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	if len(s) > 30 {
		s = s[:30]
	}
	return strings.TrimRight(s, "-")
}

// SafeName generates a unique, sanitized tool name by combining the server name
// and the tool name. The resulting name is guaranteed to be 64 characters or fewer,
// and uniqueness is ensured by appending an auto-incrementing integer if there's a collision.
func (r *NameRegistry) SafeName(serverName, toolName string) string {
	base := sanitize(serverName) + "__" + sanitize(toolName)
	if len(base) > 64 {
		base = base[:64]
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.used[base] {
		r.used[base] = true
		return base
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base[:min(60, len(base))], i)
		if !r.used[candidate] {
			r.used[candidate] = true
			return candidate
		}
	}
}
