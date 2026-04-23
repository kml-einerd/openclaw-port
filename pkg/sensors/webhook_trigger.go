// Package sensors provides trigger definitions and external input processors.
package sensors

import (
	"errors"
	"strings"
)

// WebhookTrigger enables recipes to be triggered via inbound HTTP webhooks.
type WebhookTrigger struct {
	Path   string `json:"path"`
	Method string `json:"method"`
	Secret string `json:"secret,omitempty"`
}

// Validate checks the webhook configuration for security and correctness.
func (w *WebhookTrigger) Validate() error {
	if w.Path == "" {
		return errors.New("webhook path is required")
	}
	if !strings.HasPrefix(w.Path, "/") {
		return errors.New("webhook path must start with '/'")
	}

	method := strings.ToUpper(w.Method)
	if method != "POST" && method != "GET" && method != "PUT" {
		return errors.New("webhook method must be POST, GET, or PUT")
	}

	// Secret is optional, but if provided, should have reasonable length
	if w.Secret != "" && len(w.Secret) < 8 {
		return errors.New("webhook secret must be at least 8 characters long if provided")
	}

	return nil
}
