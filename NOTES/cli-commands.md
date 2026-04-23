# OpenClaw CLI Architecture — Commands & Onboarding Patterns

## CLI Structure

**Entry Point:** `openclaw` binary (compiled from TypeScript)
- **Package.json:** Declares `"bin": { "openclaw": "openclaw.mjs" }`
- **Runtime:** Node.js 24 (recommended) or 22.16+
- **Architecture:** Modular ESM (ES modules), plugins/extensions via plugin-sdk

**Command Registration:**
- No Cobra/Yargs visible; appears to use custom CLI routing in `src/cli/` and `src/commands/`
- Commands organized by domain: `onboard-interactive.ts`, `configure.ts`, `doctor-config-flow.ts`, etc.
- Handler functions dispatch async tasks with error boundaries

---

## Onboarding Wizard Flow

### Structure: Multi-Step, Non-Linear, Config-Aware

1. **Entry:** `runInteractiveSetup(opts, runtime)` in `onboard-interactive.ts`
   - Creates `WizardPrompter` (Clack-based interactive UI)
   - Wraps `runSetupWizard()` with cancellation error handling
   - Restores terminal state on exit (signal handling built-in)

2. **Setup Flow:** `runSetupWizard()` → 705 LOC orchestration
   - **Detects existing config:** If `~/.openclaw/config.json` exists, offers: "Use existing", "Update", or "Reset"
   - **Reset Scopes:** config-only | config+creds+sessions | full (including workspace)
   - **Flow Choice:** QuickStart (defaults, minimal prompts) vs Manual (full control)
     - QuickStart suggests defaults for gateway bind, auth, tailscale, channels
     - Manual requires all choices explicitly

3. **Wizard Sections (Sequential):**
   - **Gateway config** (port, bind: loopback/LAN/custom/tailnet, auth mode: token/password)
   - **Channels setup** (WhatsApp, Telegram, Slack, Discord, etc.; filtered by quickstart allowlist)
   - **Search setup** (configurable search providers)
   - **Skills setup** (optional; skipped with `--skip-skills`)
   - **Plugin config** (sandbox backends, tool plugins; only in manual mode, not quickstart)
   - **Hooks setup** (session memory on /new)
   - **Finalize** → writes config, optionally launches TUI

### Key Interactions

- **Prompt Types** (via Clack library):
  - `select()` — single choice (autocomplete if `searchable: true`)
  - `multiselect()` — multiple choices (with tokenized search filter)
  - `text()` — free input with validation
  - `confirm()` — yes/no
  - `progress()` — spinner + progress indicator

- **Config Merge Logic:**
  - Compares existing config against new input
  - Deep merges to preserve user-set values across wizard reruns
  - `mergeWizardConfigOntoLatest(current, base, next)` — applies only new changes

- **Cancellation:**
  - `WizardCancelledError` thrown on user cancel (Ctrl+C)
  - Caught at top level; exit code 1, terminal state restored, no partial writes

- **Validation:**
  - Per-field validation in text inputs (e.g., port range, IP format)
  - Config validation after each wizard step
  - Legacy config migration (detects `~/.openclaw.yaml` → migrates to JSON)

---

## Health/Doctor Command Pattern

### `doctor-config-flow.ts` (Diagnostic & Repair)

- **Detect issues:** Legacy config keys, missing defaults, plugin incompatibilities
- **Repair workflow:**
  - Preflight check (read config, detect legacy issues)
  - Apply compatibility step (normalize old format)
  - Collect warnings (missing default account bindings, etc.)
  - Emit diagnostics → prompt user for fixes
  - Apply mutations → validate
- **Output:** Formatted notes with context; suggests `openclaw doctor --fix` to auto-repair
- **Philosophy:** Repair as a guided, non-destructive flow (all changes reviewed before write)

---

## Config Discovery & Storage

### Paths (XDG-aware)
- Primary: `~/.openclaw/config.json` (or user-configured via `--config`)
- Legacy: `~/.openclaw.yaml` (auto-migrated on detect)
- Workspace: Separate from config; stored in `agents.defaults.workspace` (default: `~/.openclaw`)

### Read-Modify-Write Pattern
- `readConfigFileSnapshot()` → snapshot (before state)
- Mutate in memory
- `writeConfigFile(nextConfig)` → atomic file write
- On conflict (concurrent edits): `ConfigMutationConflictError` thrown
- Merge strategy: detect drift, preserve user edits across reruns

---

## CLI Patterns Worth Adapting for PM-OS

### 1. **Clack-Based Interactive Prompts**
   - **What:** Polished TUI library with autocomplete, multiselect, progress
   - **Why:** OpenClaw achieves professional UX without complex terminal state management
   - **For PM-OS:** Replace raw `bufio.Scanner` CLI with Clack-like prompting for:
     - Recipe selection (searchable multiselect)
     - Parameter input (typed text inputs with validation)
     - Confirmation (yes/no for risky ops like `--delete-all`)
   - **Adoption:** JavaScript wrapper over Clack for Go via subprocess or embed Node engine

### 2. **Wizard Sections + Deep Config Merging**
   - **What:** Setup wizard broken into logical sections (gateway → channels → skills), with deep merge strategy
   - **Why:** Users can re-run wizard and update subset of config without losing values
   - **For PM-OS:** `pm-cli init --wizard` flow:
     - Section 1: API key setup (Anthropic, GCS, Supabase)
     - Section 2: Workspace config (local vs remote, git repo)
     - Section 3: Recipe catalog (fetch from catalog URL or local)
     - Section 4: Executor choice (PicoClaw, Anthropic Direct, OpenRouter)
     - Deep merge: detect if re-running, preserve unchanged values

### 3. **Doctor/Repair Command with Diagnostics**
   - **What:** `openclaw doctor --fix` detects stale configs, broken auth, missing plugins
   - **Why:** Users don't need to manually debug; CLI guides repair
   - **For PM-OS:** `pm-cli doctor` :
     - Preflight: Validate recipe syntax, check credentials in Supabase, verify GCS bucket access
     - Diagnostics: Report missing migrations, outdated recipe schema, stale run records
     - Repair: Auto-fix schema drift, prompt for missing credentials, suggest `--reset-db`
   - **Pattern:** Non-destructive, all changes reviewed before mutation

### 4. **Terminal State Restoration + Signal Handling**
   - **What:** `restoreTerminalState()` ensures Ctrl+C leaves terminal usable (even if wizard crashes)
   - **Why:** Users can safely interrupt long setup without terminal corruption
   - **For PM-OS:** Wrap all interactive CLI with signal handlers:
     - Catch SIGINT → restore cursor visibility, flush logs, exit gracefully
     - Catch SIGTERM → cancel in-flight operations, save state, exit
   - **Adoption:** Go's `os/signal` + manual terminal state restoration (via termios or similar)

---

## Shell Completion & Update Pattern

### Not Directly Visible in Codebase
- OpenClaw docs mention `fish`/`bash` shell completion support
- Completion generators separate from command handlers (see `completion-fish.ts`)
- Update mechanism mentioned in README ("Updating") — likely `npm update` or `npm latest` check

### For PM-OS
- Implement `pm-cli --generate-completion bash|zsh|fish` to output shell functions
- Include `pm-cli update` command that checks GitHub releases, compares versions, prompts to upgrade

---

## Summary: 3 Patterns for PM-OS

| Pattern | OpenClaw Example | PM-OS Adaptation |
|---------|------------------|------------------|
| **Interactive UI** | Clack-based select/multiselect with search | `pm-cli init --wizard` with recipe/executor selection |
| **Config Persistence** | Deep merge on re-run; preserve user values | `pm-cli config set/get` with merge strategy for partial updates |
| **Diagnostics + Repair** | `openclaw doctor --fix` auto-repairs config | `pm-cli doctor` validates credentials, recipe schema, GCS access; suggests fixes |

All three reduce friction for users: no manual config editing, self-service troubleshooting, safe recovery from mistakes.

