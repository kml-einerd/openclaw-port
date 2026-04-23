// Package planner provides planning and intent-matching utilities for PM-OS.
package planner

import (
	"strings"

	"openclaw-port/pkg/skills"
)

// SkillMatch represents a matched skill along with its relevance score.
type SkillMatch struct {
	Name        string
	Description string
	Score       float32
}

// MatchSkills evaluates a set of parsed skill frontmatters against an intent query
// and returns the best matching skills sorted by relevance.
//
// Adapted from openclaw skill discovery mechanics (§7.3, §7.4).
func MatchSkills(query string, frontmatters []skills.SkillFrontmatter) []SkillMatch {
	query = strings.ToLower(query)
	var matches []SkillMatch

	for _, sf := range frontmatters {
		score := computeRelevance(query, sf)
		if score > 0.1 {
			matches = append(matches, SkillMatch{
				Name:        sf.Name,
				Description: sf.Description,
				Score:       score,
			})
		}
	}

	// Sort descending by score
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}

// computeRelevance does a simple keyword overlap scoring between the query
// and the skill's name + description. In production Raven v2 uses embeddings,
// but this provides a usable baseline.
func computeRelevance(query string, sf skills.SkillFrontmatter) float32 {
	target := strings.ToLower(sf.Name + " " + sf.Description)
	words := strings.Fields(query)
	if len(words) == 0 {
		return 0
	}

	hits := 0
	for _, w := range words {
		if strings.Contains(target, w) {
			hits++
		}
	}

	return float32(hits) / float32(len(words))
}
