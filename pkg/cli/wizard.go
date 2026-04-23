package cli

import "fmt"

// WizardOptions controls the init wizard flow.
type WizardOptions struct {
	SkipValidation bool
	Reset          bool
	Quick          bool
}

// WizardSection represents one step in the interactive wizard.
type WizardSection struct {
	Title  string
	Fields []WizardField
}

// WizardField represents a single prompted input field.
type WizardField struct {
	Key         string
	Label       string
	Default     string
	Required    bool
	Sensitive   bool
}

// DefaultWizardSections returns the canonical PM-OS init wizard sections.
//
// Adapted from openclaw §5.2 pm-cli init --wizard.
func DefaultWizardSections() []WizardSection {
	return []WizardSection{
		{
			Title: "API Keys",
			Fields: []WizardField{
				{Key: "ANTHROPIC_API_KEY", Label: "Anthropic API Key", Required: true, Sensitive: true},
				{Key: "SUPABASE_URL", Label: "Supabase Project URL", Required: true},
				{Key: "SUPABASE_SERVICE_KEY", Label: "Supabase Service Role Key", Required: true, Sensitive: true},
			},
		},
		{
			Title: "Workspace",
			Fields: []WizardField{
				{Key: "PM_WORKSPACE", Label: "Workspace directory", Default: "/home/pmos/workspace"},
				{Key: "PM_DATA_DIR", Label: "Data directory", Default: "/var/lib/pmos"},
			},
		},
		{
			Title: "Executor",
			Fields: []WizardField{
				{Key: "PM_EXECUTOR", Label: "Default executor", Default: "anthropic-direct"},
			},
		},
	}
}

// ValidateWizardInput checks that all required fields have non-empty values.
func ValidateWizardInput(sections []WizardSection, values map[string]string) error {
	for _, s := range sections {
		for _, f := range s.Fields {
			if f.Required {
				val := values[f.Key]
				if val == "" {
					return fmt.Errorf("required field '%s' (%s) is empty", f.Key, f.Label)
				}
			}
		}
	}
	return nil
}
