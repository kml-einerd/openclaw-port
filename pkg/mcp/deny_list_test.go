package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDangerous(t *testing.T) {
	t.Parallel()

	assert.True(t, IsDangerous("execute_script"), "execute_script should be dangerous")
	assert.True(t, IsDangerous("rm"), "rm should be dangerous")

	assert.False(t, IsDangerous("read_file"), "read_file should be safe")
	assert.False(t, IsDangerous("list_directory"), "list_directory should be safe")
	assert.False(t, IsDangerous(""), "empty string should be safe")
}
