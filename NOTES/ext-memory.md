# OpenClaw Memory Extensions Architecture

## Overview
OpenClaw implements **3 distinct memory patterns**, each serving different durability + recall needs. All compose via plugin infrastructure (no hardcoding).

## Pattern 1: Active Memory (Short-Term Conversation Context)
**Location:** `extensions/active-memory/`  
**Purpose:** Bounded, blocking recall during **single conversation turn** (before agent reply)  
**Lifecycle:**
- User sends message → Active Memory sub-agent spawns (timeout-bounded, typically 15s)
- Sub-agent receives: latest message + small recent tail (configurable: 2 user turns, 1 assistant turn)
- Sub-agent searches session memory + any plugged-in backend (LanceDB, QMD, etc.)
- Returns structured context (`🧩 active memory: ...`) injected into main agent prompt
- If timeout expires → fails open (no memory, conversation continues)

**Key Config:**
- `queryMode`: "message" | "recent" | "full" — controls context window for sub-agent
- `promptStyle`: "balanced" | "strict" | "recall-heavy" | "precision-heavy"  
- `timeoutMs`: 250–120,000 (default 15s for speed)
- `thinking`: off/minimal/low/medium/high/xhigh/adaptive — LLM effort level
- `cacheTtlMs`: short-lived cache (1–120s) to avoid redundant sub-agent calls

**Is it needed for PM-OS?** **YES**, if you want real-time context injection during task execution. Used when:
- Workers need recent task results injected mid-stream
- Conversation-like interaction (user ↔ agent ↔ context loop)

## Pattern 2: Memory Core (File-Backed Long-Term Search)
**Location:** `extensions/memory-core/`  
**Purpose:** Persistent, searchable memory index (JSON files + FTS)  
**Capabilities:**
- `memory_search` tool — lexical + vector search (pluggable embedding provider)
- `memory_get` tool — retrieve by ID
- "Dreaming" mechanism — automatic short-term → long-term promotion (compression + storage)
- CLI: `openclaw memory search`, `memory lint`, `memory reindex`
- No external DB required; backed by filesystem

**Composition:** Pluggable embedding providers (OpenAI, Ollama, local)  
**Storage:** `~/.openclaw/memory/` (JSON files + metadata)

**Is it needed for PM-OS?** **OPTIONAL**. Useful if you want:
- Searchable task output archive (read-only knowledge base)
- Dream cycle (auto-summarization of task batches)
- Lightweight (~100MB datasets), no separate embedding service

## Pattern 3: Memory Wiki (Knowledge Vault Compiler)
**Location:** `extensions/memory-wiki/`  
**Purpose:** Durable, human-authored knowledge vault (Obsidian-compatible, markdown-first)  
**Lifecycle:**
- Ingest markdown files (manual write or bridge from Memory Core events)
- Compile to navigable vault with backlinks, dashboards, claims metadata
- Store machine-readable digests (`.openclaw-wiki/cache/agent-digest.json`)
- Tools: `wiki_search`, `wiki_get`, `wiki_apply`, `wiki_lint`
- Modes: `isolated` (own sources) | `bridge` (read Memory Core) | `unsafe-local` (private paths)

**Vault Structure:**
```
agents/, concepts/, entities/, syntheses/, sources/, reports/
+ .openclaw-wiki/cache/  (machine digests, claims.jsonl)
```

**Bridge Mode:** Wiki can read public memory artifacts + events from active-memory plugin  
**Obsidian Integration:** Auto-sync to Obsidian vault, CLI commands for workspace manipulation

**Is it needed for PM-OS?** **NO (not immediately)**. Useful for:
- Knowledge capture + human review (enterprise docs, runbooks)
- Cross-agent knowledge sharing
- Obsidian workflows

---

## Composition & Backend Pattern

All three use **pluggable backend interfaces**:

| Extension | Storage | Embedding | Search |
|-----------|---------|-----------|--------|
| **active-memory** | Session (in-memory) | Optional (QMD, LanceDB, etc.) | Configurable (search/vsearch/query modes) |
| **memory-core** | FS (JSON files) | Pluggable (OpenAI, Ollama, local) | FTS + vector (via embedding provider) |
| **memory-wiki** | FS (markdown) | None (deterministic compile) | FS scan + backlinks + Obsidian CLI |

**Backend Adapter Pattern (memory-lancedb example):**
- `lancedb-runtime.ts` — wraps LanceDB driver
- `MemoryDB` class — connection pooling, lazy init, vector ops
- Config: embedding model (OpenAI), storage options (S3, local)
- Lifecycle hooks: auto-capture, auto-recall

---

## For PM-OS: Recommended Stack

### Tier 1 (Must Have)
- **pgvector REST backend** (Supabase) — long-term vector search (embeddings + semantic queries)
  - Maps to memory-core pattern (file-backed → Postgres-backed)
  - Query: `SELECT * FROM embeddings WHERE embedding <-> query_embedding LIMIT 10`

### Tier 2 (Nice to Have)
- **Active Memory layer** (lightweight task context injection)
  - Spawn short-lived sub-agent (Haiku, <5s) before each task batch
  - Query: "Which recent task outputs are relevant to this step?"
  - Inject: `CONTEXT: ...` into step instructions
  - **Tradeoff:** +5s per wave, -10% rework on follow-on tasks

### Tier 3 (Future)
- **Wiki compiler** (human knowledge vault, optional)
  - For policy/runbook capture (read-only knowledge base)
  - Not for runtime task execution

---

## Decision Matrix

| Need | Pattern | Tradeoff |
|------|---------|----------|
| Real-time context during tasks | Active Memory | +time (but bounded) |
| Search past task outputs | Memory Core (FS) + pgvector | Storage + index cost |
| Human-authored runbooks | Memory Wiki | Maintenance overhead |
| Conversation-like agent loop | Active Memory | Not for batch/orchestration |

**Bottom line:** PM-OS needs **pgvector + optional active-memory**. Wiki is out-of-scope for execution orchestration.

---

## Implementation Notes

1. **pgvector REST endpoint:** POST `/rpc/search_embeddings` with vector similarity (no custom LLM)
2. **Active Memory sub-agent:** Fork existing active-memory logic, adapt to task context (e.g., WorkItem title + instructions)
3. **Memory Core:** Reuse memory-core CLI + search tools, point to pgvector backend
4. **No LanceDB in production:** Lancedb is for local dev (stateless free-code agents); pgvector is the durable backend

---

Generated: 2026-04-23
