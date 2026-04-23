# OpenClaw Memory System Analysis for PM-OS

**Research Date:** 2026-04-23  
**Scope:** OpenClaw memory-host-sdk patterns, conversation threading, context window management, retrieval ranking, embedding strategy  
**Goal:** Extract architecturally relevant patterns for PM-OS `pkg/memory/` enrichment

---

## Executive Summary

OpenClaw's memory system is a **layered, indexing-first architecture** that separates host process concerns (CLI, collection management) from vector/semantic retrieval. Key innovations vs PM-OS:

1. **Host-managed collections** via QMD CLI (query markdown) — decouples memory index lifecycle from app runtime
2. **Multi-component ranking** (recency + frequency + relevance + diversity + consolidation + conceptual) — goes beyond semantic similarity
3. **Session-level retention** with export-on-write — automatic cleanup & conversion to markdown
4. **Hierarchical scope + implicit context windows** — collection-level hints for retrieval disambiguation
5. **Embedding abstraction layer** — multi-backend (OpenAI, Gemini) with graceful fallback + health probing

**Top 3 Patterns Worth Adopting:**
- Composite scoring with multi-component weights (recency half-life decay, frequency counts, recall consolidation)
- Session exporter with configurable retention policy (auto-eviction + markdown export)
- Scope hierarchy with collection hints for fast doc location resolution (cache-backed)

**PM-OS Gaps Solved:**
- Currently: only Episode/Synapse, minimal ranking (no recency/frequency weighting)
- Currently: no session lifetime management or auto-cleanup
- Currently: scope is flat (/tenant/recipe/step) without collection-level clustering

**Go Implementation Effort:** ~1.5-2 weeks (core patterns only)
- Week 1: Multi-component scorer + session retention layer
- Week 2: Collection hinting + scope-aware eviction

---

## Detailed Analysis

### 1. Memory-Host-SDK Architecture (OpenClaw)

OpenClaw's memory lives in `extensions/memory-core/`. The SDK separates concerns:

```
QmdMemoryManager (qmd-manager.ts, ~2900 LOC)
  ├─ Manages SQLite index (file-watch + sync)
  ├─ Collection lifecycle (add/remove via qmd CLI)
  ├─ Session exporter (retention-aware markdown export)
  └─ Search delegation to QMD subprocess

MemorySearchManager (search-manager.ts)
  ├─ Caches QmdMemoryManager per-agent
  ├─ Falls back to MemoryIndexManager (builtin)
  └─ Gracefully handles missing qmd binary

Components:
  - Host process runs qmd CLI (query markdown) as subprocess
  - Collections stored in index DB, not app state
  - All indexing/embedding delegated to qmd daemon
```

**Key Insight:** OpenClaw treats memory indexing as **infrastructure** (like a DB), not application code. The host process is stateless; collections are persistent in the index.

---

### 2. Scope & Context Window Model

**PM-OS Current:**
```go
type Scope struct {
    TenantID string  // /tenant/X
    Recipe   string  // /recipe/Y
    Step     string  // /step/Z
}
// Matches by path prefix: /tenant/1/recipe/foo/step/bar matches /tenant/1
```

**OpenClaw Model:**
```typescript
// Collections (plural) + context limits per agent
type QmdConfig {
  collections: Array<{
    name: string      // "memory" | "sessions" | "custom"
    path: string      // filesystem root
    pattern: string   // glob pattern for watched files
    kind: MemorySource // "memory" | "sessions" | "custom"
  }>
  sessions: {
    enabled: boolean
    exportDir: string
    retentionDays: number  // <-- auto-cleanup
  }
  contextLimits: {
    maxResultsPerQuery: number
    maxContextWindowChars: number
  }
}

// Retrieval hints (collection-aware)
type DocLocationHint = {
  preferredCollection?: string  // "memory" | "sessions"
  preferredFile?: string        // "2026-04-03.md"
}
```

**OpenClaw Advantage:** Multiple named collections allow **semantic partitioning** (e.g., "sessions" collection for short-term, "memory" for long-term). PM-OS' flat scope doesn't distinguish collection purpose.

---

### 3. Ranking & Retrieval Strategy

OpenClaw uses **weighted multi-component scoring** for short-term promotion (dreaming.ts):

```typescript
type ScoringComponents = {
  frequency: number      // How often recalled in session
  relevance: number      // Semantic similarity to query
  diversity: number      // Coverage of different subtopics
  recency: number        // 0-1 decay over time (half-life model)
  consolidation: number  // Multiple unique query hits
  conceptual: number     // Abstract concept coverage
}

// Typical weights (from src/short-term-promotion.ts:59):
DEFAULT_WEIGHTS = {
  frequency: 0.2,
  relevance: 0.25,
  diversity: 0.15,
  recency: 0.15,
  consolidation: 0.1,
  conceptual: 0.15,
}

// Recency calculation:
recency = Math.max(0, 1 - (ageDays / (halfLifeDays * ln(2))))
// Half-life: 7 days = after 7 days, recency score = 0.5
```

**PM-OS Current:** Only Episode stores raw data; no ranking beyond semantic vector similarity.

**OpenClaw Pattern:** Recency half-life decay is key — prevents old, never-evicted memories from drowning out recent context. The half-life parameter is **tunable per agent** (default 7 days, configurable in dreaming config).

---

### 4. Session Management & Retention

**OpenClaw SessionExporter (qmd-manager.ts L2100-2150):**

```go
type SessionExporterConfig struct {
  dir string                  // /agent-state-dir/qmd/sessions
  retentionMs *int64          // e.g., 30 days
  collectionName string       // e.g., "sessions-agent-123"
}

// Lifecycle:
// 1. Real-time: session data in .jsonl files (raw logs)
// 2. Periodic (configurable interval): export to .md + apply retention cutoff
// 3. On export: convert session entry to markdown, apply mtime check
// 4. Cleanup: delete .md files older than retentionMs
```

**Key Pattern:**
- Sessions are **auto-exported** from JSONL → Markdown at a background interval
- **Retention policy** is time-based (e.g., "keep last 30 days")
- Stale sessions are automatically **evicted** (not manually cleaned)
- No app-level intervention needed — SDK handles lifecycle

**PM-OS Missing:** No auto-cleanup for old Episodes. They accumulate in Supabase indefinitely.

---

### 5. Embedding Strategy & Multi-Backend Abstraction

**OpenClaw Embedding Layer (tools.shared.ts, tools.test.ts):**

```typescript
// Embedding provider abstraction:
// - OpenAI (preferred, quota-aware)
// - Gemini (fallback)
// - Custom endpoint (configurable)

// Health probing:
// "openclaw memory status --deep" → probes embedding provider readiness
// Graceful degradation: if embeddings unavailable (429, timeout), memory_search returns:
// {
//   error: "embedding provider timeout",
//   warning: "Memory search is unavailable due to an embedding/provider error.",
//   action: "Check embedding provider configuration and retry memory_search."
// }

// No blocking failures — search falls back to keyword matching (BM25)
```

**PM-OS Current:** No embedding provider abstraction; hardcoded to pgvector embeddings.

---

### 6. Vector Storage: pgvector vs Alternatives

**Finding:** OpenClaw does NOT use pgvector directly. Instead:
- **QMD (query markdown)** is the primary indexing backend
- QMD uses SQLite locally, embeds via external API calls
- No SQL database integration for vector storage
- Collections are filesystem-based with pattern matching

**Implication for PM-OS:**
- PM-OS uses Supabase pgvector (REST API only)
- OpenClaw's approach is more **stateless** (SQLite is local, replaceable)
- Trade-off: OpenClaw trades persistence for isolation; PM-OS trades isolation for multi-tenant state

---

## Comparison: PM-OS vs OpenClaw Memory

| Aspect | PM-OS | OpenClaw |
|--------|-------|----------|
| **Storage** | Supabase pgvector (REST) | SQLite (local) + QMD index |
| **Scope Model** | Flat hierarchy (/tenant/recipe/step) | Multi-collection (memory, sessions, custom) |
| **Ranking** | Semantic similarity only | Multi-component: recency, frequency, relevance, diversity, consolidation, conceptual |
| **Retention** | Manual (no eviction policy) | Auto-cleanup via retentionMs policy |
| **Session Mgmt** | No dedicated session layer | SessionExporter with markdown export |
| **Embedding** | pgvector (fixed) | Pluggable: OpenAI, Gemini, custom |
| **Collection Hints** | None | preferredCollection + preferredFile for disambiguation |
| **CLI Support** | None | QMD CLI with `collection list`, `vector search` |
| **Fallback** | None | Built-in MemoryIndexManager (keyword-based) |

---

## Implementation Roadmap for PM-OS

### Phase 1: Multi-Component Scoring (Week 1)

Replace simple vector similarity with composite score:

```go
type RecallScore struct {
    Frequency     float32  // raw count from recent queries
    Relevance     float32  // cosine(query_embedding, memory_vector)
    Diversity     float32  // coverage of distinct subtopics
    Recency       float32  // exp(-age_days / halfLifeDays)
    Consolidation float32  // count of unique queries hitting this memory
    Conceptual    float32  // LLM-based concept relevance (optional)
}

func (r *RecallScore) CompositeScore(weights RecallWeights) float32 {
    sum := r.Frequency*weights.Frequency +
           r.Relevance*weights.Relevance +
           r.Diversity*weights.Diversity +
           r.Recency*weights.Recency +
           r.Consolidation*weights.Consolidation +
           r.Conceptual*weights.Conceptual
    return sum / (weights.Sum())
}
```

**Implementation:**
- Add `RecallScore` struct to `pkg/memory/types.go`
- Implement composite scorer in `pkg/memory/scorer.go` (new file)
- Update `Episode` to track `RecallCount`, `UniqueQueries`, `LastRecalledAt`
- Modify retrieval to sort by CompositeScore instead of vector similarity

---

### Phase 2: Session Retention & Auto-Eviction (Week 1-2)

Add retention policy to Episode lifecycle:

```go
type RetentionPolicy struct {
    MaxAgeDays     int       // keep episodes < N days old
    MaxPerScope    int       // keep only last N episodes per scope
    HalfLifeDays   int       // for recency decay
}

func (ep *Episode) IsStale(policy RetentionPolicy) bool {
    age := time.Since(ep.CreatedAt).Hours() / 24
    return age > float64(policy.MaxAgeDays)
}

// Background reconciler runs periodically:
func (s *SupabaseProvider) EvictStaleEpisodes(ctx context.Context, policy RetentionPolicy) error {
    // DELETE FROM episodes WHERE age > policy.MaxAgeDays
    // KEEP ONLY last N per (tenant_id, recipe_slug)
}
```

**Storage Schema Change:**
- Add `retention_days`, `half_life_days` to recipe config (if not present, use defaults)
- Add migration to add `created_at` index on episodes table
- Add reconciler loop to `cmd/pm-api/main.go` (runs every 5min, checks retention)

---

### Phase 3: Collection Hinting & Scope Clustering (Week 2)

Enhance Scope to support collection names:

```go
type Scope struct {
    TenantID       string           // required
    Recipe         string           // optional, groups memories
    Step           string           // optional, narrows scope
    CollectionHint string           // optional: "sessions" | "long-term"
}

type DocHint struct {
    PreferredCollection string
    PreferredFile       string
}

func (s *Scope) WithCollectionHint(hint string) *Scope {
    s.CollectionHint = hint
    return s
}
```

**API Change:**
- Add optional `collection_hint` to retrieval request
- Modify `search()` to prefer matching collection before falling back to scope prefix
- Cache doc location hints (like OpenClaw does)

---

## Adoption Priority

**Must Have (High Impact, <1 week):**
1. Recency component + half-life decay
2. Frequency tracking (count recalls per episode)
3. Auto-eviction based on MaxAgeDays

**Should Have (Medium Impact, 1-2 weeks):**
4. Multi-component scoring (all 6 components)
5. SessionExporter-like auto-markdown export
6. Retention policy config per recipe

**Nice to Have (Low Priority):**
7. Collection-level hinting
8. Embedding provider abstraction (Gemini fallback)
9. Memory health probing CLI

---

## Technical Debt & Gotchas

1. **pgvector Persistence:** PM-OS is multi-tenant + cloud-first; OpenClaw's local SQLite won't work. Stick with pgvector but add composite scoring on application layer.

2. **Eviction Timing:** OpenClaw background-exports sessions incrementally. PM-OS should use Supabase background job (or keep a simple reconciler in pm-engine).

3. **Scope Collision:** If Recipe + Step + Collection hint all exist, prefer explicit collection over scope prefix. Clear precedence rules needed.

4. **Cost:** More frequent writes (tracking RecallCount, LastRecalledAt) will increase Supabase usage. Monitor against budget limits.

---

## References

- `extensions/memory-core/src/memory/qmd-manager.ts` — Session lifecycle (L2100+)
- `extensions/memory-core/src/short-term-promotion.ts` — Multi-component scoring (L50-562)
- `extensions/memory-core/src/dreaming.ts` — Recency half-life config (L105+, L486+)
- `extensions/memory-core/src/memory/search-manager.ts` — Scope-aware retrieval + caching
- `/home/agdis/pm-os/pkg/memory/` — Current PM-OS implementation (baseline)

---

## Effort Estimate: **1.5–2 weeks** (core patterns only)

- **Week 1:** Recency decay + frequency tracking + basic auto-eviction (~40 LOC schema, 200 LOC scorer, 100 LOC reconciler)
- **Week 2:** Multi-component scoring, collection hinting, markdown export (~300 LOC exporter, 150 LOC retrieval logic)
- **Testing & refinement:** 2-3 days (integration tests for eviction, scoring correctness)

**Not included:** Embedding provider abstraction, QMD CLI integration, full session markdown export (lower ROI for PM-OS context).
