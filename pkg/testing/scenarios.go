package pmtest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ScenarioFrontmatter represents the YAML metadata in a QA scenario markdown file.
type ScenarioFrontmatter struct {
	Name                 string   `yaml:"name"`
	RecipeSlug           string   `yaml:"recipe_slug"`
	ExpectedStatus       string   `yaml:"expected_status"`
	ExpectedOutputContains string `yaml:"expected_output_contains"`
	Gates                []string `yaml:"gates"`
}

// LoadScenario parses a single QA scenario markdown file.
func LoadScenario(path string) (*ScenarioFrontmatter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario file: %w", err)
	}

	parts := bytes.SplitN(content, []byte("---"), 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid scenario file: missing YAML frontmatter boundaries in %s", path)
	}

	var sf ScenarioFrontmatter
	if err := yaml.Unmarshal(parts[1], &sf); err != nil {
		return nil, fmt.Errorf("failed to parse scenario frontmatter in %s: %w", path, err)
	}

	return &sf, nil
}

// LoadAllScenarios loads all QA scenarios from a given directory recursively.
func LoadAllScenarios(dir string) ([]*ScenarioFrontmatter, error) {
	var scenarios []*ScenarioFrontmatter

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			sf, err := LoadScenario(path)
			if err != nil {
				return err // Could log and continue, but strict fail is better for tests
			}
			scenarios = append(scenarios, sf)
		}
		return nil
	})

	return scenarios, err
}
