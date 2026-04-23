# OpenClaw Context Engine & Flows Analysis

## Findings

### Context Engine (Conversation-Thread Context)
**Type:** Pluggable transcript manager for multi-turn Agent conversations.

**Core Abstraction (`ContextEngine` interface):**
- `bootstrap()` — load session history from disk
- `maintain()` — post-turn compaction/pruning (explicit or background)
- `ingest()` / `ingestBatch()` — append messages; dedup checks
- `assemble()` — rebuild ordered message list under token budget
- `compact()` — reduce footprint via summaries, old-turn pruning
- `afterTurn()` — lifecycle hook post-execution (triggers async compaction)

**Context = conversation thread:** messages flow sequentially; engine maintains order, estimates tokens, watches budget.

**Compaction policy:** Token-driven threshold triggers; Opus-based summarization; rewrite old turns via `rewriteTranscriptEntries()` helper. Runtime owns DAG updates.

**Subagent lifecycle:** `prepareSubagentSpawn()` isolates child session state; `onSubagentEnded()` cleans up.

---

### Flows (Configuration/Setup Declarative Pipelines)
**Type:** NOT step chains—configuration contribution registries.

**What they do:**
- Define UI choices (auth-choice, health-check, model-picker, setup screens)
- Contributions grouped by kind (channel, core, provider, search) + surface
- Options carry metadata: labels, hints, assistant priorities, docs links

**Example:** `ProviderSetupFlow` resolves available LLM providers → renders as UI menu → user picks one.

**Flows are:** setup wizards, not execution DAGs. No inter-step data flow.

---

## Informing PM-OS Handoff Primitive Design

### Directly Adoptable Patterns

1. **Message-as-first-class value** (not just pass-by-reference): OpenClaw ingest messages as immutable entries, track IDs. PM-OS `allOutputs` map is mutable; migrate to append-only transcript model for auditability.

2. **Pluggable compaction interface** (not hardcoded loop): OpenClaw `ContextEngine.compact()` is interchangeable. PM-OS could expose `WaveCompactor` interface—allow recipes to inject custom token-reduction strategies (e.g., Forge optimizes for TDD assertions, production recipes optimize for cost).

3. **Token budget as runtime context** (proactive overflow): `ContextEngineRuntimeContext` carries `tokenBudget`, `currentTokenCount`. PM-OS step executor could respect budget—fail fast before hitting overage, trigger pre-wave compaction.

4. **Subagent isolation via lifecycle hooks** (no bleed): `prepareSubagentSpawn()` + `onSubagentEnded()` pattern. PM-OS Forge workers could use—each worker gets isolated context window, parent rolls back on child crash.

### What Does NOT Transfer

- **Flows ≠ PM-OS recipes.** Flows are config menus; recipes are task DAGs. Don't conflate.
- **Subagent prep cost:** OpenClaw isolates *before* spawn; PM-OS already spawns freely. Use if Forge cost balloons.

---

## Recommendation

Adopt patterns 1–3 incrementally. Pattern 4 only if forge workers show runaway context bloat. Skip flows entirely—PM-OS has no UI generation need.
