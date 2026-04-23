package hooks

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHookEventBus(t *testing.T) {
	// Must not run in parallel with other tests that modify the global hook registry
	ClearHooks()
	defer ClearHooks()

	var counter int32
	var wg sync.WaitGroup

	handler := func(ctx context.Context, event HookEvent) error {
		defer wg.Done()
		if event.Action == "test_action" {
			atomic.AddInt32(&counter, 1)
		}
		return nil
	}

	RegisterHook("user.created", handler)
	RegisterHook("user.created", handler) // Register twice to ensure both are called

	wg.Add(2)
	TriggerHook(context.Background(), "user.created", HookEvent{
		Type:   "user",
		Action: "test_action",
	})

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for hook handlers to execute")
	}

	assert.Equal(t, int32(2), atomic.LoadInt32(&counter), "Expected handler to be called twice")
}

func TestClearHooks(t *testing.T) {
	ClearHooks()

	RegisterHook("dummy", func(ctx context.Context, event HookEvent) error { return nil })
	
	hookMu.RLock()
	count := len(hookHandlers["dummy"])
	hookMu.RUnlock()
	assert.Equal(t, 1, count)

	ClearHooks()

	hookMu.RLock()
	countAfter := len(hookHandlers["dummy"])
	hookMu.RUnlock()
	assert.Equal(t, 0, countAfter)
}
