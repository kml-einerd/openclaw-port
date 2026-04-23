package pmtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ReplayProvider is a deterministic LLM provider that reads pre-configured
// responses from memory or a file.
type ReplayProvider struct {
	Responses map[string]string
}

// NewReplayProviderFromFile loads mock LLM responses from a JSON file.
func NewReplayProviderFromFile(path string) (*ReplayProvider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var responses map[string]string
	if err := json.Unmarshal(data, &responses); err != nil {
		return nil, fmt.Errorf("invalid replay file format: %w", err)
	}

	return &ReplayProvider{Responses: responses}, nil
}

// Execute simulates an LLM response by returning the configured string for the stepID.
func (p *ReplayProvider) Execute(ctx context.Context, stepID string, model string) (string, error) {
	// In replay mode, we return deterministic strings based on stepID.
	if resp, ok := p.Responses[stepID]; ok {
		return resp, nil
	}
	return "", fmt.Errorf("no replay configured for step: %s", stepID)
}
