package recipe

import "errors"

// TriggerPhrasesExtension adds matching hints to the recipe description
// to aid in automated intent detection (e.g., via Raven v2).
type TriggerPhrasesExtension struct {
	TriggerPhrases []string `json:"trigger_phrases,omitempty"`
}

// Validate ensures trigger phrases are well-formed.
func (t *TriggerPhrasesExtension) Validate() error {
	for _, phrase := range t.TriggerPhrases {
		if len(phrase) < 3 {
			return errors.New("trigger phrase must be at least 3 characters long")
		}
	}
	return nil
}
