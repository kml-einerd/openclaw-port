// Package skills provides types and parsers for PM-OS Skill files.
//
// Adapted from openclaw/skills/*/SKILL.md examples.
package skills

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// InstallInstruction describes a command to install a skill dependency.
type InstallInstruction struct {
	Label   string `yaml:"label"`
	Command string `yaml:"command"`
}

// PMOSMetadata holds PM-OS specific requirements and configurations for a skill.
type PMOSMetadata struct {
	RequiresTools  []string             `yaml:"requires_tools"`
	RequiresEnv    []string             `yaml:"requires_env"`
	RequiresConfig []string             `yaml:"requires_config"`
	Install        []InstallInstruction `yaml:"install"`
	PrimaryAuth    string               `yaml:"primary_auth"`
	Version        string               `yaml:"version"`
	Emoji          string               `yaml:"emoji"`
}

// Metadata is the top-level metadata object in the frontmatter.
type Metadata struct {
	PMOS PMOSMetadata `yaml:"pmos"`
}

// SkillFrontmatter represents the YAML frontmatter structure of a SKILL.md file.
type SkillFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Metadata    Metadata `yaml:"metadata"`
}

// ParseFrontmatter parses the raw YAML bytes into a SkillFrontmatter struct.
func ParseFrontmatter(data []byte) (*SkillFrontmatter, error) {
	var sf SkillFrontmatter
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("failed to parse skill frontmatter: %w", err)
	}
	return &sf, nil
}
