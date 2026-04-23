// Package pmtest provides utilities and harnesses for end-to-end integration tests.
package pmtest

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

// Harness encapsulates background processes for end-to-end tests.
//
// Adapted from openclaw/src/e2e/gateway-harness.ts.
type Harness struct {
	apiCmd    *exec.Cmd
	engineCmd *exec.Cmd
}

// StartHarness spawns the PM-API and PM-Engine processes using the provided binary paths
// and binds them to ephemeral ports (0).
func StartHarness(ctx context.Context, apiBin, engineBin string) (*Harness, error) {
	apiCmd := exec.CommandContext(ctx, apiBin, "--port=0")
	if err := apiCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start pm-api: %w", err)
	}

	engineCmd := exec.CommandContext(ctx, engineBin, "--port=0")
	if err := engineCmd.Start(); err != nil {
		_ = apiCmd.Process.Kill()
		return nil, fmt.Errorf("failed to start pm-engine: %w", err)
	}

	h := &Harness{
		apiCmd:    apiCmd,
		engineCmd: engineCmd,
	}

	// Give processes a small buffer to initialize ports
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		h.Stop()
		return nil, ctx.Err()
	}

	return h, nil
}

// Stop cleanly terminates the child processes.
func (h *Harness) Stop() {
	if h.apiCmd != nil && h.apiCmd.Process != nil {
		_ = h.apiCmd.Process.Kill()
	}
	if h.engineCmd != nil && h.engineCmd.Process != nil {
		_ = h.engineCmd.Process.Kill()
	}
}

// WaitForHealth polls an HTTP endpoint until it returns 200 OK or times out.
func WaitForHealth(ctx context.Context, url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		
		// Wait before retrying
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("healthcheck failed for %s after %v", url, timeout)
}
