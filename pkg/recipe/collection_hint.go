package recipe

import "errors"

// CollectionHintExtension provides semantic partitioning clues for short-term
// vs long-term memory structures associated with a step.
type CollectionHintExtension struct {
	CollectionHint string `json:"collection_hint,omitempty"`
}

// Validate ensures the collection hint is a valid known scope identifier.
func (c *CollectionHintExtension) Validate() error {
	if c.CollectionHint == "" {
		return nil // Optional
	}
	
	switch c.CollectionHint {
	case "sessions", "episodes", "knowledge", "scratchpad":
		return nil
	default:
		return errors.New("invalid collection_hint, must be 'sessions', 'episodes', 'knowledge', or 'scratchpad'")
	}
}
