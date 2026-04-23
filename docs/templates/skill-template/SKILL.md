---
name: example-skill
description: |
  Use when: (1) example case 1, (2) example case 2.
  NOT for: unrelated tasks.
metadata:
  pmos:
    requires_tools: ["git"]
    requires_env: ["GITHUB_TOKEN"]
    requires_config: []
    install:
      - label: "Install git"
        command: "apt-get install git"
    primary_auth: "GITHUB_TOKEN"
    version: "1.0.0"
    emoji: "wrench"
---

# Example Skill

This is the body of the skill. It should contain detailed instructions
for the agent on how to perform the task.

## Steps

1. Step one description
2. Step two description

## Constraints

- Maximum 500 lines in this file
- Keep instructions clear and actionable
