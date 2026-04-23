package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPMOSMetadata_ValidateRequires(t *testing.T) {
	t.Parallel()

	valid := &PMOSMetadata{
		RequiresTools: []string{"git"},
		RequiresEnv:   []string{"TOKEN"},
		PrimaryAuth:   "TOKEN",
	}
	assert.NoError(t, valid.ValidateRequires())

	invalidAuth := &PMOSMetadata{
		RequiresEnv: []string{"OTHER"},
		PrimaryAuth: "TOKEN",
	}
	assert.Error(t, invalidAuth.ValidateRequires())

	tooManyTools := &PMOSMetadata{
		RequiresTools: make([]string, 21),
	}
	assert.Error(t, tooManyTools.ValidateRequires())

	emptyTool := &PMOSMetadata{RequiresTools: []string{""}}
	assert.Error(t, emptyTool.ValidateRequires())

	emptyEnv := &PMOSMetadata{RequiresEnv: []string{""}}
	assert.Error(t, emptyEnv.ValidateRequires())

	emptyCfg := &PMOSMetadata{RequiresConfig: []string{""}}
	assert.Error(t, emptyCfg.ValidateRequires())
}
