---
summary: "Quick setup for PicoClaw executor integration"
read_when:
  - Setting up PicoClaw as primary LLM executor
  - Debugging PicoClaw auth or timeout errors
title: "PicoClaw Executor Guide"
---

# Document Frontmatter Pattern

All PM-OS documentation files under `/docs/` should include a YAML frontmatter
block with the following fields:

## Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Human-readable document title |
| `summary` | string | One-line summary of the document's purpose |

## Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `read_when` | string[] | List of situations when this doc is relevant |
| `tags` | string[] | Searchable tags for discovery |
| `last_updated` | date | ISO 8601 date of last review |

## Example

```yaml
---
summary: "How to configure quality gates for recipe validation"
read_when:
  - Setting up quality gates for a new recipe
  - Debugging why a recipe run was rejected
title: "Quality Gates Configuration"
tags: ["gates", "quality", "recipes"]
---
```

This enables Raven v2 to perform contextual documentation discovery
based on the current task context.
