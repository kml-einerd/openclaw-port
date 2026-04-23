package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter_HappyPath(t *testing.T) {
	t.Parallel()

	yamlData := []byte(`
name: code-reviewer
description: |
  Use when: (1) PR feedback.
metadata:
  pmos:
    requires_tools: ["git", "gh"]
    requires_env: ["GITHUB_TOKEN"]
    install:
      - label: "Install gh CLI"
        command: "brew install gh"
    primary_auth: "GITHUB_TOKEN"
    version: "1.0.0"
`)

	sf, err := ParseFrontmatter(yamlData)
	require.NoError(t, err)
	assert.Equal(t, "code-reviewer", sf.Name)
	assert.Contains(t, sf.Description, "Use when")
	
	pmos := sf.Metadata.PMOS
	assert.Equal(t, []string{"git", "gh"}, pmos.RequiresTools)
	assert.Equal(t, []string{"GITHUB_TOKEN"}, pmos.RequiresEnv)
	assert.Len(t, pmos.Install, 1)
	assert.Equal(t, "brew install gh", pmos.Install[0].Command)
	assert.Equal(t, "1.0.0", pmos.Version)
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	t.Parallel()

	yamlData := []byte(`
name: code-reviewer
  description: invalid indentation
`)

	sf, err := ParseFrontmatter(yamlData)
	assert.Error(t, err)
	assert.Nil(t, sf)
}
