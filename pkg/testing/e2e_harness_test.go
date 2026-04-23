package pmtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHarness_StartAndStop(t *testing.T) {
	t.Parallel()

	// Use standard cross-platform commands for the test to ensure they "start"
	var cmd string
	if exec.Command("cmd", "/c", "echo", "test").Run() == nil {
		cmd = "cmd"
	} else {
		cmd = "sh"
	}

	// We'll mock the binaries using a simple shell sleep
	h, err := StartHarness(context.Background(), cmd, cmd)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	h.Stop()
}

func TestWaitForHealth_Success(t *testing.T) {
	t.Parallel()

	// Start a real local test server to poll
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	err := WaitForHealth(context.Background(), ts.URL, 1*time.Second)
	assert.NoError(t, err)
}

func TestWaitForHealth_Timeout(t *testing.T) {
	t.Parallel()

	// Use a dummy port that won't respond
	err := WaitForHealth(context.Background(), "http://127.0.0.1:49191/health", 200*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "healthcheck failed")
}
