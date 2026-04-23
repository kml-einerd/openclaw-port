// Package channels provides inbound routing for integrations like Telegram.
//
// Adapted from openclaw/src/channels/.
package channels

import "context"

// Message represents a normalized inbound message from any external platform.
type Message struct {
	ID        string
	Channel   string
	AccountID string
	Content   string
	Metadata  map[string]interface{}
}

// Handler defines the interface for processing normalized inbound messages.
type Handler interface {
	Handle(ctx context.Context, msg Message) error
}
