package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunDoctor(t *testing.T) {
	t.Parallel()

	// RunDoctor will produce checks even without real services — stubs/warns are expected
	checks := RunDoctor(context.Background(), DoctorOptions{Verbose: true})

	assert.GreaterOrEqual(t, len(checks), 4, "should produce at least 4 checks")

	for _, c := range checks {
		assert.NotEmpty(t, c.Name)
		assert.Contains(t, []string{"ok", "warn", "fail"}, c.Status)
	}
}
