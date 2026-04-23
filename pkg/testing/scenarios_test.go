package pmtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadScenario(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	
	content := []byte(`---
name: test-scenario
recipe_slug: foo
expected_status: completed
expected_output_contains: "success"
gates: [gate1, gate2]
---
This is the body.
`)
	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err)

	sf, err := LoadScenario(path)
	assert.NoError(t, err)
	assert.Equal(t, "test-scenario", sf.Name)
	assert.Equal(t, "foo", sf.RecipeSlug)
	assert.Equal(t, "success", sf.ExpectedOutputContains)
	assert.Len(t, sf.Gates, 2)
}

func TestLoadScenario_Invalid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.md")
	
	// Missing boundaries
	content := []byte(`name: test-scenario`)
	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err)

	_, err = LoadScenario(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing YAML frontmatter boundaries")
}
