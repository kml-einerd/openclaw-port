package cli

import (
	"context"
	"testing"

	"openclaw-port/pkg/channels"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunChannelSetup_Telegram(t *testing.T) {
	t.Parallel()

	router := channels.NewRouter()
	result, err := RunChannelSetup(context.Background(), router, ChannelSetupOptions{
		ChannelType: "telegram",
		WebhookURL:  "https://example.com/webhook",
	})

	require.NoError(t, err)
	assert.True(t, result.Registered)
	assert.Equal(t, "telegram", result.ChannelType)
}

func TestRunChannelSetup_DryRun(t *testing.T) {
	t.Parallel()

	router := channels.NewRouter()
	result, err := RunChannelSetup(context.Background(), router, ChannelSetupOptions{
		ChannelType: "telegram",
		DryRun:      true,
	})

	require.NoError(t, err)
	assert.False(t, result.Registered)
	assert.Contains(t, result.Detail, "dry-run")
}

func TestRunChannelSetup_UnsupportedChannel(t *testing.T) {
	t.Parallel()

	router := channels.NewRouter()
	_, err := RunChannelSetup(context.Background(), router, ChannelSetupOptions{
		ChannelType: "whatsapp",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported channel type")
}

func TestRunChannelSetup_EmptyType(t *testing.T) {
	t.Parallel()

	router := channels.NewRouter()
	_, err := RunChannelSetup(context.Background(), router, ChannelSetupOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel type is required")
}
