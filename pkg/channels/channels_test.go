package channels

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockHandler struct {
	Received []Message
}

func (m *MockHandler) Handle(ctx context.Context, msg Message) error {
	m.Received = append(m.Received, msg)
	return nil
}

func TestRouter_Dispatch(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	handler := &MockHandler{}
	router.Register("telegram", handler)

	msg := Message{Channel: "telegram", Content: "hello"}
	err := router.Dispatch(context.Background(), msg)
	
	assert.NoError(t, err)
	assert.Len(t, handler.Received, 1)
	assert.Equal(t, "hello", handler.Received[0].Content)

	errUnknown := router.Dispatch(context.Background(), Message{Channel: "slack"})
	assert.Error(t, errUnknown)
	assert.Contains(t, errUnknown.Error(), "no handler registered")
}

func TestTelegramWebhook_ValidMessage(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	handler := &MockHandler{}
	router.Register("telegram", handler)

	webhook := NewTelegramWebhook(router)

	payload := []byte(`{
		"update_id": 1001,
		"message": {
			"message_id": 42,
			"from": {"id": 999},
			"text": "execute scan"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	rr := httptest.NewRecorder()

	webhook.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, handler.Received, 1)
	
	msg := handler.Received[0]
	assert.Equal(t, "telegram", msg.Channel)
	assert.Equal(t, "42", msg.ID)
	assert.Equal(t, "999", msg.AccountID)
	assert.Equal(t, "execute scan", msg.Content)
}

func TestTelegramWebhook_InvalidMethod(t *testing.T) {
	t.Parallel()
	webhook := NewTelegramWebhook(NewRouter())
	
	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rr := httptest.NewRecorder()
	
	webhook.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestTelegramWebhook_BadJSON(t *testing.T) {
	t.Parallel()
	webhook := NewTelegramWebhook(NewRouter())
	
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{bad}`)))
	rr := httptest.NewRecorder()
	
	webhook.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTelegramWebhook_EmptyText(t *testing.T) {
	t.Parallel()
	
	router := NewRouter()
	handler := &MockHandler{}
	router.Register("telegram", handler)
	webhook := NewTelegramWebhook(router)
	
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{"message":{"text":""}}`)))
	rr := httptest.NewRecorder()
	
	webhook.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, handler.Received, 0)
}
