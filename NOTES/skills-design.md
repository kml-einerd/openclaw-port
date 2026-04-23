# OpenClaw Skills System — Design Patterns Analysis

**Date:** 2026-04-23  
**Goal:** Extract patterns from 53 OpenClaw skills for PM-OS skills primitive  
**Scope:** File structure, metadata, discovery, composition, lifecycle

---

## Executive Summary

OpenClaw's skills system is a mature, **progressive-disclosure design** where metadata enables skill discovery, SKILL.md contains procedural instructions, and bundled resources (scripts, references, examples) stay isolated for selective loading. Unlike PM-OS's current CrewAI port (which is task-focused), OpenClaw skills are **agent-capability-focused** — they extend what agents CAN DO rather than what agents WILL EXECUTE.

### Top 5 Patterns to Adopt in PM-OS

1. **Progressive Disclosure (3-layer loading):** Metadata (~100 words) → SKILL.md (~500-1k words) → References/Scripts (unlimited, loaded as-needed). Reduces context bloat massively.

2. **Metadata Frontmatter (YAML):** Skill discovery driven by `name` + `description` fields only. Enables structured triggers: bin requirements, env vars, OS constraints, installation methods, emoji/icon.

3. **Bundled Resources Organization:** Three directories:
   - `scripts/` — deterministic code (Python/Bash/Go), executed without loading into context
   - `references/` — documentation loaded as-needed (schemas, API docs, detailed workflows)
   - `assets/` — templates, boilerplate, non-documentation output files
   - (Avoid `README.md`, `CHANGELOG.md`, auxiliary docs)

4. **Trigger Phrases in Description:** Description includes explicit phrases like "Use when: (1) X, (2) Y. NOT for: Z." Makes skill selection explicit and deterministic.

5. **Tool Declaration in Metadata:** Skills declare hard requirements:
   - `requires.bins` — external CLIs required (gh, summarize, codex, claude)
   - `requires.env` — environment variables (NOTION_API_KEY, OPENAI_API_KEY)
   - `requires.config` — config files or structured settings
   - Enables auto-prompting for missing dependencies and installation

---

## All 53 Skills — Categorized by Type

### **Integration Skills (External APIs/Services)** — 14 skills
1. **1password** — Password vault access
2. **discord** — Discord bot actions (messages, reactions, pins)
3. **gemini** — Google Gemini API integration
4. **gh-issues** — GitHub issues via gh CLI
5. **github** — GitHub operations (PRs, CI, code review)
6. **himalaya** — Email client operations
7. **notion** — Notion pages/databases API
8. **slack** — Slack messages, reactions, pins
9. **openai-whisper** — Speech transcription (local)
10. **openai-whisper-api** — Speech transcription (API)
11. **oracle** — Oracle database operations
12. **spotify-player** — Spotify playback control
13. **trello** — Trello board management
14. **xurl** — URL shortening

### **Agent/Coding Skills** — 4 skills
1. **coding-agent** — Spawn coding agents (Codex, Claude Code, Pi, OpenCode)
2. **skill-creator** — Author/audit skills and SKILL.md files
3. **node-connect** — Node.js process control (execute, debug)
4. **healthcheck** — Host security auditing and hardening

### **Content/Utility Tools** — 12 skills
1. **bear-notes** — Bear notes management
2. **blogwatcher** — Blog post monitoring
3. **canvas** — Canvas document editing
4. **nano-pdf** — PDF extraction/processing
5. **obsidian** — Obsidian vault operations
6. **session-logs** — Session logging and playback
7. **summarize** — Summarize URLs, files, transcripts
8. **tmux** — tmux session control
9. **video-frames** — Extract frames from video
10. **voice-call** — Voice call handling
11. **weather** — Weather API
12. **peekaboo** — Screenshot/screen capture

### **Specialized Domain/Admin Tools** — 10 skills
1. **apple-notes** — Apple Notes integration
2. **apple-reminders** — Apple Reminders management
3. **blucli** — Bluetooth CLI control
4. **bluebubbles** — Blue Bubbles messaging
5. **camsnap** — Camera snapshot capture
6. **imsg** — iMessage handling
7. **openhue** — Philips Hue lighting control
8. **sag** — System audio graph control
9. **sonoscli** — Sonos speaker control
10. **things-mac** — Things 3 task manager

### **Business/Admin Tools** — 6 skills
1. **clawhub** — OpenClaw instance discovery/management
2. **eightctl** — Eight Sleep bed control
3. **goplaces** — Google Places API
4. **mcporter** — Product import/export
5. **ordercli** — Order management
6. **wacli** — WhatsApp CLI

### **Flow/Orchestration** — 3 skills
1. **model-usage** — CodexBar model cost tracking
2. **taskflow** — Durable multi-step task orchestration
3. **taskflow-inbox-triage** — Concrete taskflow pattern (inbox inbox triage)

### **CLI/Entry Point Tools** — 2 skills
1. **gifgrep** — GIF search/extraction
2. **songsee** — Song/music recognition

### **Special/Experimental** — 2 skills
1. **sherpa-onnx-tts** — Text-to-speech via ONNX
2. **skill-creator** — (see above, under Agents)

---

## Skill Structure Anatomy

### Required: `SKILL.md`

**Frontmatter (YAML):**
```yaml
---
name: <skill-name>
description: >
  Use when: (1) specific case, (2) another case.
  NOT for: X, Y.
metadata:
  openclaw:
    emoji: "🔧"  # Visual identifier
    os: ["darwin", "linux"]  # Optional OS constraints
    requires:
      bins: ["gh", "summarize"]  # External CLIs
      env: ["GITHUB_TOKEN", "API_KEY"]  # Env vars
      config: ["channels.slack"]  # Config paths
    install:
      - id: "brew"
        kind: "brew"
        formula: "gh"
        bins: ["gh"]
        label: "Install GitHub CLI (brew)"
    primaryEnv: "GITHUB_TOKEN"  # Primary auth var
---
```

**Body (Markdown):**
- Concise instructions (<500 lines)
- Progressive disclosure: quick start → common commands → advanced
- Explicit trigger phrases for when to use
- Example commands/patterns
- DO NOT include: README.md, CHANGELOG.md, setup guides, auxiliary docs

### Optional: `scripts/` (Executable Code)

When the same code is rewritten repeatedly or deterministic reliability is needed:

```
scripts/
  ├── model_usage.py       # Python script (deterministic)
  ├── wait-for-text.sh     # Bash script
  ├── test_model_usage.py  # Unit tests
  └── frame.sh             # Shell utility
```

**Characteristics:**
- Executed WITHOUT loading into context window (deterministic)
- Can handle environment-specific adjustments
- May be read for patching if agent needs to adapt
- Include unit tests alongside

### Optional: `references/` (Reference Material)

Documentation loaded only as-needed into context:

```
references/
  ├── codexbar-cli.md      # CLI flags, cost JSON schema
  ├── finance.md           # Database schemas
  ├── api_docs.md          # API specifications
  └── policies.md          # Company policies
```

**Characteristics:**
- NOT automatically loaded (context efficiency)
- Referenced from SKILL.md with clear guidance on when to read
- Large files (>10k words) — include grep patterns in SKILL.md
- Keep SKILL.md body lean; move detailed info here

### Optional: `assets/` (Output Resources)

Files for final output, NOT loaded into context:

```
assets/
  ├── logo.png            # Brand assets
  ├── template.html       # HTML/React boilerplate
  ├── slides.pptx         # PowerPoint template
  └── frontend-template/  # Full project template
```

**Characteristics:**
- Used in final output (copied, modified, rendered)
- NOT loaded into context window
- Enables agents to work with files without hogging tokens

---

## Key Design Patterns

### Pattern 1: Metadata-Driven Discovery

Skill system triggers **based on metadata alone** — no need to load SKILL.md body to determine if skill is relevant:

```yaml
metadata:
  openclaw:
    requires:
      bins: ["gh"]  # Skill only available if `gh` CLI exists
      env: ["GITHUB_TOKEN"]  # Or if this env var is set
      config: ["channels.slack"]  # Or if Slack is configured
```

**PM-OS application:** Recipe step could declare `requires: ["git", "node", "go"]` to auto-inject them as constraints. Engine filters available skills pre-dispatch.

### Pattern 2: Trigger Phrase Specification

Rather than loose keywords, OpenClaw skills declare explicit trigger phrases:

```markdown
Use when:
  (1) checking PR status or CI
  (2) creating/commenting on issues
  (3) listing/filtering PRs or issues
  
NOT for:
  complex web UI interactions
  bulk operations across many repos
```

**PM-OS application:** Recipe step `instructions` could include a `trigger_phrases` field for LLM matching. Or, task router (Raven v2) uses these as hard constraints in brain veto injection.

### Pattern 3: Multi-Runtime Execution

Some skills abstract over multiple executors (Codex, Claude Code, Pi, OpenCode) with different calling conventions:

```bash
# Codex/Pi/OpenCode: requires PTY
bash pty:true command:"codex exec 'task'"

# Claude Code: no PTY, use --print --permission-mode bypassPermissions
bash command:"claude --permission-mode bypassPermissions --print 'task'"
```

**PM-OS similarity:** Engine already supports multiple executors (HTTPExecutor, PicoClawExecutor, FallbackExecutor, etc.). Could expose this as skill-level dispatch: "use executor X when Y".

### Pattern 4: Progressive Disclosure + Context Efficiency

OpenClaw minimizes context bloat via:

1. **Metadata only** in context by default (~100 words)
2. **SKILL.md loaded when triggered** (~500-1k words)
3. **References loaded on-demand** (unlimited, agent decides)
4. **Scripts executed as-is** (no loading)

**Token savings example:**
- skill-creator: 150-line SKILL.md + 5 reference/script files = agent doesn't see all 150 lines unless it needs them

**PM-OS application:** Could adopt the same for recipes:
- Recipe metadata: slug, title, brief description, required tools
- Recipe body: full step definitions
- Reference files: schema definitions, policy docs
- Only load recipe body + relevant references into work item context

### Pattern 5: Skill Composition & Chaining

Skills can orchestrate across other skills:

- **coding-agent** spawns agents that use other skills internally (e.g., git, gh, npm)
- **taskflow** orchestrates multi-step work with linked child tasks
- **skill-creator** validates and packages other skills

**PM-OS application:** Wave 8 Forge already decomposes recipes into microtasks. Could formalize skill composition: recipe step references another skill, engine auto-loads skill context + contracts.

---

## Differences from PM-OS CrewAI Port

| Aspect | OpenClaw | PM-OS (Current) |
|--------|----------|-----------------|
| **Scope** | Agent capability (tools, workflows) | Task execution (agents, workers) |
| **Unit of Work** | Skill (reusable capability) | WorkItem (per-wave task) |
| **Discovery** | Metadata-driven (name + description) | Catalog loaded at startup |
| **Execution** | Not owned by skill (agent/CLI invokes) | Engine owns Wave execution |
| **State** | Minimal (trigger conditions) | Rich (WorkItem, TaskResult, Contract) |
| **Composition** | Skills reference each other | Steps reference sub-recipes |
| **Context Loading** | Progressive (3-layer) | Monolithic (full recipe at start) |
| **Tool Binding** | Declarative metadata (bins, env) | Implicit (tasks assume tools exist) |

**Key insight:** OpenClaw skills are **middleware**, PM-OS skills are **execution contexts**. Adopting OpenClaw patterns means shifting PM-OS skills toward "capability declaration" rather than "task execution."

---

## Just-Copy Transfers

### 1. Metadata + Frontmatter Structure

Copy OpenClaw's YAML frontmatter into PM-OS skill metadata:

```yaml
---
name: my-skill
description: Use when X. NOT for Y.
metadata:
  pmos:
    emoji: "🔧"
    requires_tools: ["git", "node"]
    requires_env: ["API_KEY"]
    requires_config: ["path/to/config.json"]
    install_hints:
      - label: "Install Node.js"
        command: "brew install node"
    primary_auth: "API_KEY"
---
```

### 2. Progressive Disclosure (3-Layer)

Apply to PM-OS recipes:

- **Layer 1 (Metadata):** Slug, title, tools required, 1-line description
- **Layer 2 (Recipe Body):** Full step definitions, waves, contracts
- **Layer 3 (References):** Detailed schemas, API docs, examples (in `references/` subdir)

### 3. Bundled Resources Organization

Adopt same directory structure:

```
recipes/
  my-recipe/
    ├── recipe.v2.json      # Wave definitions
    ├── scripts/             # Deterministic code
    │   └── validate.go
    ├── references/          # Docs loaded as-needed
    │   └── schema.md
    └── assets/              # Output templates
        └── template.html
```

### 4. Trigger Phrase Specification

Include in recipe description:

```json
{
  "slug": "code-review",
  "description": "Review code changes.\nUse when: (1) PR submitted, (2) feedback requested.\nNOT for: design reviews, security audits (use separate recipes).",
  "waves": [...]
}
```

### 5. Tool Requirement Declaration

Add to recipe metadata:

```json
{
  "requires": {
    "tools": ["git", "node", "python"],
    "env_vars": ["GITHUB_TOKEN", "API_KEY"],
    "services": ["supabase", "gcs"]
  }
}
```

---

## Patterns NOT to Adopt

1. **OpenClaw's `pty:true` / `process` tool:** PM-OS is Go + HTTP; no terminal multiplexing needed.

2. **OpenClaw's Lobster language:** PM-OS uses JSON recipes; no DSL equivalent needed.

3. **OpenClaw's `channels` (Discord, Slack, Notion bindings):** PM-OS uses post-wave hooks (Telegram, webhooks) differently.

4. **OpenClaw's `metadata.openclaw.install` arrays:** PM-OS should use Supabase function registry + GCP Cloud Functions discovery instead.

5. **OpenClaw's plugin/hook system:** PM-OS has a different hook architecture (BeforeRun, AfterWave, AfterTask, AfterRun).

---

## Skill Discovery Mechanism

OpenClaw:
1. Agent requests capability ("summarize this URL")
2. System scans all SKILL.md frontmatter for `name` + `description` match
3. If metadata requirements met (`bins`, `env`, `config`), load SKILL.md body
4. Agent receives instructions + bundled resources location

**PM-OS equivalent:**
1. Recipe defines wave step with `type: "llm"` and title/instructions
2. DeterministicOptimizer (Raven v2) scans brain slugs for matching skills
3. If match found + requirements met, inject skill context into WorkItem.Constraints
4. PicoClaw/HTTPExecutor receives enriched WorkItem with skill guidance

---

## Summary: What Changes

### For PM-OS Skills Definition

**Before (CrewAI port):**
```json
{
  "role": "Code Reviewer",
  "goal": "Review code",
  "backstory": "...",
  "tools": ["git", "github"]
}
```

**After (OpenClaw-inspired):**
```yaml
---
name: code-reviewer
description: |
  Review code changes in PRs.
  Use when: (1) PR feedback requested, (2) quality gate.
  NOT for: design reviews, security audits.
metadata:
  pmos:
    requires_tools: ["git"]
    requires_env: ["GITHUB_TOKEN"]
    emoji: "🔍"
    install:
      - label: "Install GitHub CLI"
        command: "brew install gh"
---
# Code Review Skill

## Quick start
... (progressive disclosure)
```

### For Recipe Execution

**Before:** Engine loads full recipe into context per-dispatch.

**After:** Engine loads:
1. Recipe metadata only (fast check)
2. Full recipe body if executed
3. Only referenced `references/` files as-needed

### For Skill Composition

**Before:** Skills are independent.

**After:** Skills declare dependencies on other skills (e.g., code-review depends on git, github skills).

---

## Recommendations for PM-OS

1. **Adopt metadata frontmatter** immediately (low cost, high benefit for discovery)
2. **Implement `requires` field** for tools/env/config (enables pre-flight checks)
3. **Split large recipes into references/** (post-Wave-7, for Wave 8 optimization)
4. **Formalize trigger phrases** in recipe description (improve Raven v2 brain matching)
5. **Keep scripts/ unchanged** (PM-OS already uses this pattern for free-code provider)

**Estimated effort:** 1 day (metadata + requires), 3 days (references split + discovery system).

---

## References

- OpenClaw Skills: `/tmp/openclaw-eval/openclaw/skills/` (53 examined)
- Key patterns: coding-agent, skill-creator, model-usage, taskflow, github, notion
- PM-OS Recipe format: `pkg/recipe/schema.go`
- PM-OS Engine: `pkg/engine/recipe_runner.go`
