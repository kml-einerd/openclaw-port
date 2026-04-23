package skills

import (
	"errors"
	"fmt"
)

// ValidateRequires checks if the skill's environmental and tool requirements
// are well-formed. It acts as the parser/validator for the C6 requirement.
func (pmos *PMOSMetadata) ValidateRequires() error {
	if len(pmos.RequiresTools) > 20 {
		return errors.New("a skill cannot require more than 20 tools")
	}

	for _, tool := range pmos.RequiresTools {
		if tool == "" {
			return errors.New("empty tool requirement found")
		}
	}

	for _, env := range pmos.RequiresEnv {
		if env == "" {
			return errors.New("empty env requirement found")
		}
	}

	for _, cfg := range pmos.RequiresConfig {
		if cfg == "" {
			return errors.New("empty config requirement found")
		}
	}

	if pmos.PrimaryAuth != "" {
		found := false
		for _, env := range pmos.RequiresEnv {
			if env == pmos.PrimaryAuth {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("primary_auth '%s' must be listed in requires_env", pmos.PrimaryAuth)
		}
	}

	return nil
}
