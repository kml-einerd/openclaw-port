# OpenClaw Docs Structure Analysis

## Platform & Tooling

**Documentation Platform:** Mintlify (https://mintlify.com/docs.json schema)

- **Deployment:** Generated locale trees live in separate publish repo (`openclaw/docs`) while source English is in main product repo (`openclaw/openclaw`)
- **Sync workflow:** GitHub Actions mirrors English docs to publish repo, which regenerates locale translations (12 languages + glossaries)
- **Config:** Single `docs.json` (1638 lines) defines navigation tabs, redirects, branding, styling
- **Build:** Static site generation via Mintlify — Vercel or self-hosted

---

## Information Architecture

### Top-Level Navigation (5 Tabs)

1. **Guides** (Getting Started, Install, Deployment, Configuration)
2. **Features** (Channels, Nodes, Tools, Automation, Plugins, Skills)
3. **Concepts** (Architecture, Messages, Sessions, Memory, Models)
4. **Platforms** (macOS, Linux, Windows, iOS/Android)
5. **Models** (Provider index: Anthropic, OpenAI, 40+ integrations)

### Directory Structure

```
docs/
├── start/               # Onboarding (getting-started, wizard, hubs)
├── install/             # 28 install methods (npm, Docker, K8s, cloud platforms, bare metal)
├── channels/            # 15+ messaging platforms (Discord, Telegram, Slack, WhatsApp, Signal, etc.)
├── nodes/               # Mobile + Pi hardware (iOS/Android pairing, Canvas, Voice)
├── concepts/            # 35+ conceptual deep-dives (Agent loop, Memory, Sessions, Context engine)
├── providers/           # 50+ model providers + API integrations
├── tools/               # 40+ agent tools (web browser, search, media, code execution)
├── plugins/             # SDK docs + examples (channels, providers, skills)
├── automation/          # Cron, webhooks, tasks, taskflow, hooks
├── gateway/             # Core config, security, troubleshooting, health diagnostics
├── help/                # FAQ, Troubleshooting (symptom-based runbooks)
├── web/                 # Control UI, Web Chat, Canvas
├── platforms/           # OS-specific guides + mobile apps
├── reference/           # Credits, CLI reference, token usage
├── .i18n/               # Glossaries + navigation stubs for 12 locales
└── docs.json            # Single source of truth for nav/redirects
```

---

## Documentation Patterns Worth Stealing

### 1. **Fast Triage → Deep Runbooks (2-Tier Support)**

**Pattern:** FAQ/help pages offer quick symptom-based answers with clear escalation paths.

**Examples:**
- `/help/faq` → "First 60 seconds if something is broken" (lists 7 diagnostic commands in order)
- `/gateway/troubleshooting` → "Command ladder" (runs same 5 commands, then symptom-based sections below)
- Each section links back to related reference docs (configuration, security, health)

**Why it works:**
- Users stuck get unstuck fast without reading 50-page docs
- Clear escalation: `openclaw status` → `openclaw gateway status` → `openclaw logs --follow` → `openclaw doctor`
- All commands are actionable and idempotent

**PM-OS adoption:**
- Create `/help/first-60-seconds.md` with gate checks, recipe validation, run logs (3 commands)
- Link to deeper `/gateway/troubleshooting` with deterministic checks per stage (quality gates, executor retries, Supabase connectivity)
- Add quick-reference box at top of FAQ ("If stuck → run these 5 commands")

---

### 2. **Metadata + Smart Linking via Frontmatter**

**Pattern:** Every page has YAML frontmatter declaring when it's read + what problem it solves.

```markdown
---
summary: "Quick answers plus deeper troubleshooting for real-world setups"
read_when:
  - Answering common setup, install, onboarding, or runtime support questions
  - Triaging user-reported issues before deeper debugging
title: "FAQ"
---
```

**Why it works:**
- Search engines + docs site can recommend pages based on user intent
- Sidebar/TOC generation can be automated
- Content authors declare the "why" for each page
- Maintainers see which docs are duplicated vs uncovered

**PM-OS adoption:**
- Add `read_when` to every guide (especially recipes, quality gates, executor docs)
- Add `summary` explaining the single reason to read this page
- Generate a "Docs map" showing coverage gaps (pages with no `read_when` entries)

---

### 3. **Navigational Cards + Hub Pattern**

**Pattern:** Use `<Card>` + `<CardGroup>` to create scannable landing pages instead of wall-of-text TOCs.

Example from index.md:
```markdown
<Columns>
  <Card title="Get Started" href="/start/getting-started" icon="rocket">
    Install OpenClaw and bring up the Gateway in minutes.
  </Card>
  <Card title="Channels" href="/channels/telegram" icon="message-square">
    Channel-specific setup for Telegram, Discord, and more.
  </Card>
</Columns>
```

**Why it works:**
- Scannable, visual, icon-driven
- Reduces cognitive load (users pick their path, not overwhelmed by 20 links)
- Consistent pattern across all hub pages
- Cards are self-contained with description + icon

**PM-OS adoption:**
- Replace flat sidebar for `/recipes`, `/quality`, `/executors` with hub pages using cards
- Use icons to differentiate: 📝 recipes, ✅ gates, ⚙️ executors, 🔧 integrations
- Link to 4-6 most common pages per hub, not exhaustive list

---

### 4. **Per-Feature Getting Started (Steps + Quick Setup)**

**Pattern:** Every channel/provider/tool has its own "Quick setup" section (3-5 Steps component) before deep reference.

Example: `/channels/telegram.md`
```markdown
<Steps>
  <Step title="Create the bot token in BotFather">
    Open Telegram and chat with **@BotFather**...
  </Step>
  <Step title="Configure token and DM policy">
    [JSON config example]
  </Step>
  <Step title="Start gateway and approve first DM">
    [3 commands with expected output]
  </Step>
</Steps>
```

**Why it works:**
- Copy-paste friendly (code blocks + command examples)
- Visual progress indicator (Step 1/3/etc)
- Reader knows when they're done
- Details (API docs, advanced config) come *after* working example

**PM-OS adoption:**
- Wrap every executor, gate, provider in a `<Steps>` block
- Example: `/executors/picoclaw.md` → "5 steps to run a task via PicoClaw"
- Include expected success output at each step (not just "if error, see...")

---

### 5. **i18n as First-Class: Glossaries + Workflow Automation**

**Pattern:** Translation memory + glossaries live in source repo; locale nav stubs auto-synced, full translations generated via GH Actions.

**Files:**
- `glossary.<lang>.json` — 18+ term mappings (e.g. "Gateway" → "Gateway" for pt-BR, custom terms for zh-CN)
- `<lang>-navigation.json` — locale-specific nav overrides (e.g. zh-Hans gets expanded nav)
- `.tm.jsonl` — translation memory keyed by hash (prevents retranslating stable content)
- GH Actions → translate each locale on schedule + on release

**Why it works:**
- Glossaries keep product terms consistent across languages
- TM prevents paying for same translation twice
- Split repo keeps 12 locale trees out of main product repo (CI/build faster)
- Automation runs on schedule, not manual; scales to 20+ languages cheaply

**PM-OS adoption:**
- Create `docs/.i18n/glossary.pt-BR.json` with PM-OS terms (Recipe, Wave, Gate, Executor, Brain, Veto)
- Add GH Action to sync PT-BR docs daily (like OpenClaw does zh-CN + ja-JP)
- Use glossary to keep "Wave" vs "Onda" consistent across recipes/quality/gates docs

---

## Key Metrics

| Metric | Value |
|--------|-------|
| Total doc pages | ~150+ |
| Install methods documented | 28 |
| Channels supported | 15+ |
| Concepts deep-dives | 35+ |
| Model providers | 50+ |
| Languages + glossaries | 12 |
| Navigation structure | 5 tabs + grouped sidebar |
| Config schema size | 1638 lines JSON |

---

## OpenClaw's Secret: Mintlify + Discipline

1. **Mintlify:** Dead simple to maintain; git-driven; search out of the box; mobile-friendly; dark mode
2. **Metadata:** Every page declares its "why" (reduces orphaned docs)
3. **2-tier support:** FAQ → runbooks hierarchy (users find answer in 1 min, not 10)
4. **Per-feature Getting Started:** Each integration/provider has copy-paste steps before reference
5. **Glossaries:** Ensures translated docs don't drift (pt-BR team sees English term + approved pt-BR equivalent)
6. **Hub pattern:** Navigate by outcome (get a bot running, integrate a model) not by file structure
7. **Automation:** Let CI/GH Actions handle translate/sync, humans focus on accuracy

---

## Adoption Priority for PM-OS

**Phase 1 (Immediate):**
- [ ] Migrate from current `/docs` to Mintlify + publish separate docs site
- [ ] Create `docs/docs.json` with 3 tabs: Getting Started, Reference (Recipes/Gates/Executors), API
- [ ] Add `read_when` + `summary` metadata to 20+ existing guides

**Phase 2 (Month 2):**
- [ ] Build hub pages for Recipes, Quality Gates, Executors (with Card grid)
- [ ] Per-executor getting started (5 steps: setup → auth → test → integrate → troubleshoot)
- [ ] FAQ + 2-tier troubleshooting (first 60s → deep diagnostics)

**Phase 3 (Month 3):**
- [ ] PT-BR glossary + translation workflow (via GH Actions)
- [ ] Dedicated "Deployment" hub (local, GCP, Kubernetes, etc.)
- [ ] Search integration + analytics
