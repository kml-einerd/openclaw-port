# OpenClaw Agent Patterns — PM-OS Evaluation

**Date:** 2026-04-23  
**Scope:** `src/cron/isolated-agent/` (58 files)  
**Goal:** Extract 3 patterns worth porting to PM-OS task-first philosophy

---

## 1. Agent Architecture Overview

OpenClaw uses **isolated agent turns** (cron jobs) as the execution unit, modeled as:

```
CronJob
├── payload: { kind: "agentTurn", ... }
├── sessionTarget: "isolated" | shared
└── delivery: { channel, accountId, to, threadId }
    ↓
IsolatedAgent (stateless HTTP executor)
├── Session (mutable, persisted)
│   ├── sessionEntry (model, provider, skills snapshot, auth profile)
│   ├── transcriptPath (JSONL session file)
│   └── store (multi-key registry)
├── Executor (functional)
│   ├── createCronPromptExecutor() → runPrompt()
│   ├── executeCronRun() → retry loop w/ fallback
│   └── runWithModelFallback() → provider/model selection
└── Result
    ├── RunCronAgentTurnResult (output text, delivery trace, telemetry)
    └── CronDeliveryTrace (target, attempted, delivered flags)
```

**Key insight:** Agent lifecycle is **request-response**, not stateful loop. State (session, skills, auth) persists separately.

---

## 2. Three Patterns Worth Porting to PM-OS

### Pattern A: **Lazy-Loaded Runtime Modules** (Execution Hooks)

**What:** Runtimes (executor, delivery, auth) are dynamic imports cached per agent turn, not global.

**OpenClaw code:**
```typescript
let cronExecutorRuntimePromise: Promise<...> | undefined;

async function loadCronExecutorRuntime() {
  cronExecutorRuntimePromise ??= import("./run-executor.runtime.js");
  return await cronExecutorRuntimePromise;
}
```

**Why it works:**
- Splits build-time (types) from runtime (execution). Each turn can upgrade runtimes independently.
- Avoids circular dependencies: core agnostic, runtimes inject their behavior.
- Enables A/B testing: two turns can use different executor versions.

**Fit for PM-OS:** 
Recipe steps (llm_agent, llm_with_tools, verify, dataset) could have **pluggable executors** loaded on-demand. Forge workers could inject a different executor than production. TDD gate could patch executor mid-recipe without rebuild.

**Example:**
```go
// pm-os pkg/engine
type StepExecutorRuntime interface {
  Execute(ctx, step) -> (output, error)
}

var runtimes = map[string]func()StepExecutorRuntime{} // lazy load
```

---

### Pattern B: **Session State + Live Model Fallback**

**What:** Model/provider selection is **mutable**, **persisted**, **runtime-switchable** within a run.

**OpenClaw code:**
```typescript
export type CronLiveSelection = LiveSessionModelSelection;

export type MutableCronSession = {
  sessionEntry: MutableCronSessionEntry; // { model, provider, authProfileId }
  store: MutableSessionStore; // persisted key→entry map
};

// Mid-execution fallback:
try { await executor.runPrompt(...); }
catch (err instanceof LiveSessionModelSwitchError) {
  params.liveSelection.provider = err.provider;
  params.liveSelection.model = err.model;
  await params.persistSessionEntry(); // immediate flush
  continue; // retry with new model
}
```

**Why it works:**
- **Deterministic without LLM choice:** If Opus fails, retry is Sonnet (not arbitrary).
- **Persisted immediately:** Crash mid-retry? Next turn sees the fallback choice.
- **Finite retry:** MAX_MODEL_SWITCH_RETRIES=2 prevents runaway fallbacks.

**Fit for PM-OS:**
Wave-level fallback policy: if Haiku task fails → retry Sonnet → retry Opus. Gate can encode `on_fail: "fallback_model"`. Wave result carries `fallbackProvider` and `fallbackModel` for observability.

**Example:**
```go
type Wave struct {
  Steps []Step
  FallbackModels []string // ["haiku", "sonnet", "opus"]
}

// engine/recipe_runner.go
if task fails {
  if len(wave.FallbackModels) > 0 {
    nextModel := wave.FallbackModels[attempt]
    task.Model = nextModel
    continue // retry with next model
  }
}
```

---

### Pattern C: **Subagent Delegation + Descendant Registry**

**What:** Agent can spawn child agents (subagents) and wait for results, with **parent-child session tracking**.

**OpenClaw code:**
```typescript
// During agent execution, agent can call sessions_spawn tool
// Spawn creates descendant with parent sessionKey reference

// After agent completes, check for descendants:
const descendants = listDescendantRunsForRequester(params.sessionKey)
  .filter(entry => entry.endedAt >= runStartedAt);

// If agent only sent interim ack ("on it"), wait for descendants:
if (shouldRetryInterimAck && !hasActiveDescendants) {
  const continuationPrompt = 
    "Your previous response was only an ack. Complete the task now.";
  await executor.runPrompt(continuationPrompt);
}
```

**Why it works:**
- **Session lineage:** Child runs linked to parent sessionKey, not just via data flow.
- **Interim ack detection:** Agent says "I'll do X", system detects and prompts for actual work.
- **Parallel delegation:** Subagents run async; parent waits with timeout.

**Fit for PM-OS:**
Recipe steps of type `"delegate"` invoke sub-recipes. Parent recipe:
1. Spawns child run with `depends_on: [parent_run_id]`
2. Polls child until done (with timeout)
3. Injects child result into parent's allOutputs map
4. Continues wave execution

**Example:**
```go
type Step struct {
  Type string // "delegate"
  SubRecipe string // "recipe.slug"
  DependsOn []string // parent run_id
}

// engine/recipe_runner.go
case "delegate":
  childRunID := uuid()
  childTask := &WorkItem{
    Title: step.SubRecipe,
    Type: "delegate",
    DependsOn: []string{runID},
  }
  childResult := await delegateSubRecipe(childTask, step.SubRecipe)
  allOutputs[step.ID] = childResult.Output
```

---

## 4. Summary: Fit for PM-OS Task-First Paradigm

| Pattern | OpenClaw Origin | PM-OS Benefit |
|---------|-----------------|---------------|
| **A: Lazy Runtimes** | Executor plugins | Forge workers inject custom executors; TDD patches mid-recipe |
| **B: Live Fallback** | Model selection | Wave fallback policy: Haiku→Sonnet→Opus; persisted choice |
| **C: Subagent Registry** | Session lineage | Delegation: sub-recipes as first-class steps, parent polls child |

**Recommended integration order:**
1. **Pattern B first** — Model fallback ties into existing Wave/Task architecture; low risk.
2. **Pattern C second** — Delegation is additive; uses existing dependency system (depends_on).
3. **Pattern A last** — Requires refactor to plugin-load step executors; highest value but touches core.

All three fit PM-OS **task-first** philosophy: tasks define work, system executes and retries automatically.

---

## 5. Key Files Examined

- `run.ts` (300 LOC) — Entry point, lazy module loading, session lifecycle
- `run-executor.ts` (400 LOC) — Prompt execution, retry loop, model fallback, subagent wait
- `run-session-state.ts` (150 LOC) — Session persistence, live selection mutation
- `subagent-followup.ts` (100 LOC) — Descendant tracking, interim ack retry logic
- `run.types.ts` (20 LOC) — Type contracts

Total examined: ~1K LOC core logic (tests not counted).

---

## 6. OpenClaw Context

- **Language:** TypeScript/Node
- **Agent model:** Stateless HTTP turns, mutable session store
- **Execution:** Cron jobs spawn isolated agent runs; runs can delegate to subagents
- **Persistence:** Sessions in JSON registry; transcripts as JSONL files
- **Tools:** Agent can call sessions_spawn, messaging tools, external APIs via skill registry
