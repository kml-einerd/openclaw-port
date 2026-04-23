package channels

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// TelegramWebhook processes raw HTTP webhooks from Telegram and normalizes
// them into the PM-OS Message format for internal routing.
//
// Adapted from openclaw/extensions/telegram/inbound.ts.
type TelegramWebhook struct {
	router *Router
}

// NewTelegramWebhook creates a new receiver attached to the provided router.
func NewTelegramWebhook(router *Router) *TelegramWebhook {
	return &TelegramWebhook{router: router}
}

// TelegramUpdate represents the payload structure from the Telegram Bot API.
type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID int `json:"id"`
		} `json:"from"`
		Text string `json:"text"`
	} `json:"message"`
}

// ServeHTTP translates the inbound POST payload into a generalized context
// and dispatches it.
func (t *TelegramWebhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update TelegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if update.Message.Text == "" {
		// Ignore empty messages or non-text messages safely
		w.WriteHeader(http.StatusOK)
		return
	}

	msg := Message{
		ID:        strconv.Itoa(update.Message.MessageID),
		Channel:   "telegram",
		AccountID: strconv.Itoa(update.Message.From.ID),
		Content:   update.Message.Text,
	}

	// Dispatch normalized payload. In production, error logs would capture routing failures.
	_ = t.router.Dispatch(r.Context(), msg)

	// Always 200 OK so Telegram stops retrying
	w.WriteHeader(http.StatusOK)
}
