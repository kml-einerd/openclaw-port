package channels

import (
	"context"
	"errors"
	"sync"
)

// Router directs inbound normalized messages to the appropriate internal handlers
// based on the channel type.
type Router struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewRouter creates a new inbound channel router.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]Handler),
	}
}

// Register binds a specific platform string (e.g., "telegram") to its execution handler.
func (r *Router) Register(channelType string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[channelType] = h
}

// Dispatch securely routes the message to the corresponding handler.
// Returns an error if the channel isn't supported.
func (r *Router) Dispatch(ctx context.Context, msg Message) error {
	r.mu.RLock()
	h, ok := r.handlers[msg.Channel]
	r.mu.RUnlock()

	if !ok {
		return errors.New("no handler registered for channel: " + msg.Channel)
	}

	return h.Handle(ctx, msg)
}
