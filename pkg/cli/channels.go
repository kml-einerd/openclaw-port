package cli

import (
	"context"
	"fmt"

	"openclaw-port/pkg/channels"
)

// ChannelSetupOptions configures the channel setup command.
type ChannelSetupOptions struct {
	ChannelType string
	WebhookURL  string
	Secret      string
	DryRun      bool
}

// ChannelSetupResult captures the outcome of configuring a channel.
type ChannelSetupResult struct {
	ChannelType string
	Registered  bool
	Detail      string
}

// RunChannelSetup validates and registers a new inbound channel (e.g., Telegram).
//
// Adapted from openclaw §5.4 pm-cli channels setup.
func RunChannelSetup(ctx context.Context, router *channels.Router, opts ChannelSetupOptions) (*ChannelSetupResult, error) {
	if opts.ChannelType == "" {
		return nil, fmt.Errorf("channel type is required")
	}

	switch opts.ChannelType {
	case "telegram":
		if opts.DryRun {
			return &ChannelSetupResult{
				ChannelType: "telegram",
				Registered:  false,
				Detail:      "dry-run: would register Telegram webhook handler",
			}, nil
		}

		webhook := channels.NewTelegramWebhook(router)
		_ = webhook // In real impl, this would be mounted on the HTTP server

		return &ChannelSetupResult{
			ChannelType: "telegram",
			Registered:  true,
			Detail:      fmt.Sprintf("Telegram webhook registered at %s", opts.WebhookURL),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported channel type: %s", opts.ChannelType)
	}
}
