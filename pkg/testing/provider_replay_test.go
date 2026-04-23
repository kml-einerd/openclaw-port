package pmtest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayProvider(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "replay.json")
	
	err := os.WriteFile(path, []byte(`{"step1": "deterministic output", "step2": "another output"}`), 0644)
	require.NoError(t, err)

	provider, err := NewReplayProviderFromFile(path)
	require.NoError(t, err)

	res, err := provider.Execute(context.Background(), "step1", "sonnet")
	assert.NoError(t, err)
	assert.Equal(t, "deterministic output", res)

	res2, err2 := provider.Execute(context.Background(), "step2", "opus")
	assert.NoError(t, err2)
	assert.Equal(t, "another output", res2)

	_, err3 := provider.Execute(context.Background(), "unknown", "haiku")
	assert.Error(t, err3)
	assert.Contains(t, err3.Error(), "no replay configured")
}
