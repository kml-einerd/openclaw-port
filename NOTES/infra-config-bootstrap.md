# OpenClaw Infra, Config & Bootstrap Patterns

## Directory Structure

### `/src/infra/`
Core infrastructure primitives: error formatting, home-dir resolution, dotenv loading, shell env fallback, TLS cert resolution for Node.

### `/src/config/`
Configuration system (90+ files). Handles:
- **Loading**: `io.ts` — main entry point (JSON5 parsing, env substitution, includes, validation, audit)
- **Paths**: `paths.ts` — resolve config file location, state dir
- **Schema**: Zod-based validation with plugins (allows provider-specific rules)
- **Persistence**: Write audit, backups, recovery (last-known-good snapshots)
- **Env substitution**: `${VAR}` placeholder resolution with pre-read snapshot capture
- **Includes**: Support for `include:` directive (config composition)
- **Agent dirs**: Multi-agent mode with duplicate-dir detection

### `/src/bootstrap/`
Minimal: Node.js startup TLS setup (`NODE_EXTRA_CA_CERTS`, `NODE_USE_SYSTEM_CA` resolution).

### `/src/hooks/`
Event-driven hook system:
- **Registry**: Global singleton (`globalThis[Symbol.for(...)]`) with Map<eventKey, handlers[]>
- **Event types**: agent bootstrap, gateway startup, message (received/sent/transcribed/preprocessed), session patch
- **Lifecycle**: Register → Trigger → Serial async execution

---

## Key Patterns for PM-OS

### Pattern 1: Config as State Machine (Snapshot + Recovery)
**What OpenClaw does:**
- Runtime config snapshot in memory (`getRuntimeConfigSnapshot()`)
- Last-known-good backup on disk (`io.observe-recovery.ts`)
- On read failure → auto-promote last-good snapshot
- Config write audit log (append-only, forensic trail)

**For PM-OS:**
- Adopt `ConfigSnapshot` type: `{ raw: string; parsed: object; hash: string; mtimeMs: number }`
- Store only on Supabase (`runs` table: `config_snapshot` JSONB field)
- On engine startup: read from DB, validate schema, fall back to last-good if broken
- Audit trail: append to `run_events` table on every config change

**Gap today:** PM-OS loads from `.env` files, no schema validation, no recovery path.

---

### Pattern 2: Env Substitution with Snapshot Isolation
**What OpenClaw does:**
- Pre-read env vars before config parse: `Record<string, string | undefined>`
- Preserve `${VAR}` markers in config (don't eval at read time)
- On write: re-substitute from original snapshot (safe to copy config across machines)
- `envSnapshotForRestore` option passed to writer

**For PM-OS:**
```go
type ConfigSnapshot struct {
  Raw      string                 // Original YAML/JSON
  EnvRefs  map[string]string      // ${FOO} → captured value
  Parsed   map[string]interface{} // Substituted config
}

// On startup: readConfigWithEnvCapture()
// On write: writeConfigPreservingEnvRefs(snapshot, envSnapshot)
```

**Gap today:** PM-OS inlines env vars at startup, can't replay config on another host.

---

### Pattern 3: Global Hooks + Local Event Bus
**What OpenClaw does:**
- `registerHook(eventKey, handler)` → global registry (survives bundle splits)
- Fire-and-forget (`triggerHook()` → serial async)
- Hook context is immutable (no state mutation)
- Example: `agent:bootstrap`, `gateway:startup`, `message:received`

**For PM-OS:**
```go
// In pkg/infra/hooks.go
type HookEvent struct {
  Type    string                 // "run", "task", "wave"
  Action  string                 // "started", "completed", "failed"
  Context map[string]interface{} // immutable
}

type HookHandler func(ctx context.Context, event HookEvent) error

// Global registry
var hookHandlers = make(map[string][]HookHandler)

// In engine.WithAfterWave(), AfterTask(), etc.
engine.TriggerHook(ctx, "wave:completed", waveContext)
```

**Use cases:**
- `run:started` → log to Telegram (existing)
- `task:completed` → save Episode to knowledge DB
- `wave:completed` → trigger OpusReview gate
- `run:failed` → send alert + recover next run

**Gap today:** Engine uses receiver-style callbacks (`engine.WithAfterWave(func...)`). Loose coupling via hooks is better for plugins.

---

## Current PM-OS Gaps

1. **No config schema validation** — accepts any `.env`, fails at runtime
2. **No recovery path** — if production config is corrupt, manual rollback
3. **No config audit** — who changed what, when, why?
4. **Env vars leaked into code** — can't replay config on new host
5. **No hook system** — callbacks are hardcoded in engine loop
6. **No structured logging bootstrap** — logger initialized ad-hoc
7. **No graceful shutdown hooks** — systemd kills process, no cleanup

---

## Recommended Phasing

- **Phase 1**: Adopt global hooks + event bus in engine (2–3 days)
- **Phase 2**: Add config snapshot + env substitution to Supabase (3–5 days)
- **Phase 3**: Schema validation + recovery (5–7 days)
- **Phase 4**: Structured logging bootstrap (2–3 days)

**Estimated total:** ~2–3 weeks for production-grade ops.
