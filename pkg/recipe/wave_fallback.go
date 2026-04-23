// Package recipe provides extensions to the core PM-OS Recipe schema.
package recipe

import "errors"

// WaveFallbackExtension adds model fallback policies to a Wave.
// Adapted from openclaw/src/agents/run-executor.ts (Pattern B).
type WaveFallbackExtension struct {
	FallbackModels []string `json:"fallback_models,omitempty"`
}

// Validate checks that the fallback models configuration is safe.
func (e *WaveFallbackExtension) Validate() error {
	if len(e.FallbackModels) > 5 {
		return errors.New("cannot specify more than 5 fallback models for a wave")
	}
	return nil
}
