package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultWizardSections(t *testing.T) {
	t.Parallel()

	sections := DefaultWizardSections()
	assert.Len(t, sections, 3)
	assert.Equal(t, "API Keys", sections[0].Title)
	assert.GreaterOrEqual(t, len(sections[0].Fields), 2)
}

func TestValidateWizardInput_AllPresent(t *testing.T) {
	t.Parallel()

	sections := DefaultWizardSections()
	values := map[string]string{
		"ANTHROPIC_API_KEY":    "sk-test",
		"SUPABASE_URL":         "http://localhost:54321",
		"SUPABASE_SERVICE_KEY": "eyJ...",
	}

	err := ValidateWizardInput(sections, values)
	assert.NoError(t, err)
}

func TestValidateWizardInput_MissingRequired(t *testing.T) {
	t.Parallel()

	sections := DefaultWizardSections()
	values := map[string]string{
		"SUPABASE_URL": "http://localhost:54321",
		// Missing ANTHROPIC_API_KEY
	}

	err := ValidateWizardInput(sections, values)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
}
