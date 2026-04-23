package recipe

import "fmt"

// RequiresExtension specifies environmental and tool dependencies
// that must be met before a recipe can be executed.
type RequiresExtension struct {
	Tools    []string `json:"tools,omitempty"`
	EnvVars  []string `json:"env_vars,omitempty"`
	Services []string `json:"services,omitempty"`
}

// Validate ensures there are no obvious misconfigurations in the requirements.
func (r *RequiresExtension) Validate() error {
	for _, tool := range r.Tools {
		if tool == "" {
			return fmt.Errorf("tool requirement cannot be empty string")
		}
	}
	for _, env := range r.EnvVars {
		if env == "" {
			return fmt.Errorf("env_var requirement cannot be empty string")
		}
	}
	return nil
}
