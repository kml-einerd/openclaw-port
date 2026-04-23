package mcp

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameRegistry_SafeName_HappyPath(t *testing.T) {
	t.Parallel()

	r := NewNameRegistry()
	name := r.SafeName("github", "search_repos")

	assert.Equal(t, "github__search_repos", name)
}

func TestNameRegistry_SafeName_Sanitization(t *testing.T) {
	t.Parallel()

	r := NewNameRegistry()
	name := r.SafeName("git!hub@server", "search(repos)")

	assert.Equal(t, "git-hub-server__search-repos", name)
}

func TestNameRegistry_SafeName_Truncation(t *testing.T) {
	t.Parallel()

	r := NewNameRegistry()
	server := "very_long_server_name_that_exceeds_thirty_chars_by_far"
	tool := "very_long_tool_name_that_exceeds_thirty_chars_too"

	name := r.SafeName(server, tool)

	assert.Equal(t, "very_long_server_name_that_exc__very_long_tool_name_that_excee", name)
	assert.True(t, len(name) <= 64)
}

func TestNameRegistry_SafeName_Collision(t *testing.T) {
	t.Parallel()

	r := NewNameRegistry()

	name1 := r.SafeName("github", "search")
	name2 := r.SafeName("github", "search")
	name3 := r.SafeName("github", "search")

	assert.Equal(t, "github__search", name1)
	assert.Equal(t, "github__search-2", name2)
	assert.Equal(t, "github__search-3", name3)
}

func TestNameRegistry_SafeName_CollisionWithTruncation(t *testing.T) {
	t.Parallel()

	r := NewNameRegistry()

	server := "very_long_server_name_that_exceeds_thirty_chars_by_far"
	tool := strings.Repeat("x", 40)

	name1 := r.SafeName(server, tool)
	name2 := r.SafeName(server, tool)

	assert.True(t, len(name1) <= 64)
	assert.True(t, strings.HasSuffix(name2, "-2"))
	assert.True(t, len(name2) <= 64)
}

func TestNameRegistry_SafeName_Concurrent(t *testing.T) {
	t.Parallel()

	r := NewNameRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.SafeName("server", "tool")
		}()
	}

	wg.Wait()

	// There should be exactly 100 items in the registry starting with "server__tool"
	r.mu.RLock()
	count := len(r.used)
	r.mu.RUnlock()

	assert.Equal(t, 100, count)
}
