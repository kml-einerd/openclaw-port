package recipe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWaveFallbackExtension(t *testing.T) {
	t.Parallel()

	valid := &WaveFallbackExtension{FallbackModels: []string{"m1", "m2", "m3"}}
	assert.NoError(t, valid.Validate())

	invalid := &WaveFallbackExtension{FallbackModels: []string{"1", "2", "3", "4", "5", "6"}}
	assert.Error(t, invalid.Validate())
}

func TestRequiresExtension(t *testing.T) {
	t.Parallel()

	valid := &RequiresExtension{Tools: []string{"git"}, EnvVars: []string{"TOKEN"}}
	assert.NoError(t, valid.Validate())

	invalidTool := &RequiresExtension{Tools: []string{""}}
	assert.Error(t, invalidTool.Validate())

	invalidEnv := &RequiresExtension{EnvVars: []string{""}}
	assert.Error(t, invalidEnv.Validate())
}

func TestTriggerPhrasesExtension(t *testing.T) {
	t.Parallel()

	valid := &TriggerPhrasesExtension{TriggerPhrases: []string{"review code"}}
	assert.NoError(t, valid.Validate())

	invalid := &TriggerPhrasesExtension{TriggerPhrases: []string{"ab"}}
	assert.Error(t, invalid.Validate())
}

func TestRetentionPolicy(t *testing.T) {
	t.Parallel()

	valid := &RetentionPolicy{MaxAgeDays: 30, MaxPerScope: 100, HalfLifeDays: 7}
	assert.NoError(t, valid.Validate())

	invalidAge := &RetentionPolicy{MaxAgeDays: -1}
	assert.Error(t, invalidAge.Validate())
	
	invalidScope := &RetentionPolicy{MaxPerScope: -1}
	assert.Error(t, invalidScope.Validate())
	
	invalidHalfLife := &RetentionPolicy{HalfLifeDays: -1}
	assert.Error(t, invalidHalfLife.Validate())
}

func TestCollectionHintExtension(t *testing.T) {
	t.Parallel()

	valid := &CollectionHintExtension{CollectionHint: "sessions"}
	assert.NoError(t, valid.Validate())

	validEmpty := &CollectionHintExtension{}
	assert.NoError(t, validEmpty.Validate())

	invalid := &CollectionHintExtension{CollectionHint: "unknown"}
	assert.Error(t, invalid.Validate())
}
