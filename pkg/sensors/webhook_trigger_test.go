package sensors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookTrigger_Validate(t *testing.T) {
	t.Parallel()

	valid := &WebhookTrigger{Path: "/hooks/github", Method: "POST", Secret: "supersecret"}
	assert.NoError(t, valid.Validate())

	noPath := &WebhookTrigger{Method: "POST"}
	assert.Error(t, noPath.Validate())

	badPath := &WebhookTrigger{Path: "hooks/github", Method: "POST"}
	assert.Error(t, badPath.Validate())

	badMethod := &WebhookTrigger{Path: "/h", Method: "PATCH"}
	assert.Error(t, badMethod.Validate())

	badSecret := &WebhookTrigger{Path: "/h", Method: "GET", Secret: "short"}
	assert.Error(t, badSecret.Validate())
}
