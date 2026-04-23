# Skill Template

This directory provides the canonical structure for a PM-OS Skill.

## Directory Layout

```
skills/
  my-skill/
    SKILL.md          # Frontmatter YAML + body (<500 lines)
    scripts/          # Executable code (NOT loaded into context)
      validate.go
    references/       # Docs loaded on-demand
      api-schema.md
    assets/           # Output templates
      template.json
```

## Frontmatter Fields

See `SKILL.md` in this directory for a working example.

## 3-Layer Loading

| Layer | Content | When loaded |
|-------|---------|-------------|
| Metadata | name + description (~100 words) | Always (pre-dispatch) |
| SKILL.md body | Instructions (~500 lines) | When match detected |
| References | Schemas, API docs (unlimited) | On-demand by agent |
| Scripts | Executable code | Executed without loading |
