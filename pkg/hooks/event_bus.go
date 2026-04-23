// Package hooks provides a global event bus for asynchronous pub/sub patterns within PM-OS.
//
// Adapted from openclaw/src/hooks/.
package hooks

import (
	"context"
	"sync"
)

// HookEvent encapsulates the details of an event occurring in the system.
type HookEvent struct {
	Type    string
	Action  string
	Context map[string]interface{}
}

// HookHandler defines the signature for a callback function triggered by a hook.
// Handlers are executed asynchronously.
type HookHandler func(ctx context.Context, event HookEvent) error

var (
	hookHandlers = make(map[string][]HookHandler)
	hookMu       sync.RWMutex
)

// RegisterHook adds a new handler function for a specific event key.
// It is safe for concurrent use.
func RegisterHook(eventKey string, h HookHandler) {
	hookMu.Lock()
	defer hookMu.Unlock()
	hookHandlers[eventKey] = append(hookHandlers[eventKey], h)
}

// TriggerHook asynchronously dispatches the event to all registered handlers
// for the given eventKey.
func TriggerHook(ctx context.Context, eventKey string, event HookEvent) {
	hookMu.RLock()
	handlers := make([]HookHandler, len(hookHandlers[eventKey]))
	copy(handlers, hookHandlers[eventKey])
	hookMu.RUnlock()

	for _, h := range handlers {
		// Execute each handler in its own goroutine to prevent blocking
		go func(fn HookHandler) {
			_ = fn(ctx, event) // Errors are intentionally swallowed in async fire-and-forget hooks
		}(h)
	}
}

// ClearHooks removes all registered handlers. This is primarily useful for testing.
func ClearHooks() {
	hookMu.Lock()
	defer hookMu.Unlock()
	hookHandlers = make(map[string][]HookHandler)
}
