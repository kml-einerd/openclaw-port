package planner

import (
	"testing"

	"openclaw-port/pkg/skills"

	"github.com/stretchr/testify/assert"
)

func TestMatchSkills(t *testing.T) {
	t.Parallel()

	frontmatters := []skills.SkillFrontmatter{
		{Name: "code-reviewer", Description: "Use when: PR feedback, code quality review."},
		{Name: "deploy-helper", Description: "Use when: deploying to production, CI/CD pipelines."},
		{Name: "data-analyst", Description: "Use when: analyzing data, creating reports."},
	}

	matches := MatchSkills("review code quality", frontmatters)
	assert.NotEmpty(t, matches)
	assert.Equal(t, "code-reviewer", matches[0].Name)
}

func TestMatchSkills_NoMatch(t *testing.T) {
	t.Parallel()

	frontmatters := []skills.SkillFrontmatter{
		{Name: "deploy-helper", Description: "Use when: deploying."},
	}

	matches := MatchSkills("xyzzy foobar", frontmatters)
	assert.Empty(t, matches)
}

func TestMatchSkills_EmptyQuery(t *testing.T) {
	t.Parallel()

	frontmatters := []skills.SkillFrontmatter{
		{Name: "anything", Description: "whatever"},
	}

	matches := MatchSkills("", frontmatters)
	assert.Empty(t, matches)
}
