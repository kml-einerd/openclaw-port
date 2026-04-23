# OpenClaw Extraction Plan — PM-OS Concretude Pragmatica

**Data:** 2026-04-23  
**Fontes:** 19 notes em `/tmp/openclaw-eval/notes/*.md`  
**Filosofia:** Itens que Dis ganha em 1-5 dias cada. Zero filosofia.

---

## 1. COPY-PASTE DIRETO (zero ou minima adaptacao)

### 1.1 Benchmark Percentile Calculator
- **Origem:** `scripts-and-tooling.md` -> `bench-cli-startup.ts` (180 LOC)
- **LOC estimado:** 40 LOC Go
- **O que faz:** Calcula p50/p95/min/max/mean de arrays de tempos de execucao
- **Ajuste minimo:** Renomear pacote `openclaw` -> `pmbench`
- **Tempo integrar:** 30 min

```go
// tools/pm-bench/stats.go
func calculatePercentiles(samples []int64) (p50, p95, min, max, mean int64) {
    sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
    n := len(samples)
    min, max = samples[0], samples[n-1]
    for _, s := range samples { mean += s }
    mean /= int64(n)
    p50, p95 = samples[(n*50)/100], samples[(n*95)/100]
    return
}
```

### 1.2 Docker Build/Skip Pattern (E2E)
- **Origem:** `scripts-and-tooling.md` -> `lib/docker-e2e-image.sh` (70 LOC)
- **LOC estimado:** 35 LOC bash
- **O que faz:** Build Docker com flag SKIP_DOCKER_BUILD pra reusar imagem existente
- **Ajuste minimo:** Trocar nome da imagem
- **Tempo integrar:** 20 min

```bash
# scripts/e2e/docker-build.sh
docker_build_or_skip() {
  local image_name="$1"; local dockerfile="$2"
  if [ "${SKIP_DOCKER_BUILD:-0}" = "1" ]; then
    if ! docker image inspect "$image_name" >/dev/null 2>&1; then
      echo "ERROR: image not found: $image_name" >&2; exit 1
    fi
    echo "Reusing: $image_name"; return
  fi
  echo "Building: $image_name"
  docker build -t "$image_name" -f "$dockerfile" . || exit 1
}
```

### 1.3 Artifact Validation Shell Helpers
- **Origem:** `scripts-and-tooling.md` -> `pr-lib/common.sh` (150 LOC)
- **LOC estimado:** 50 LOC bash
- **O que faz:** `require_artifact()`, `check_recipe_artifacts()`, path classification
- **Ajuste minimo:** Trocar `validate-recipes/main.py` pelo validador Go
- **Tempo integrar:** 30 min

### 1.4 MCP Tool Name Sanitizer
- **Origem:** `mcp-integration.md` -> `src/agents/pi-bundle-mcp-names.ts`
- **LOC estimado:** 30 LOC Go
- **O que faz:** `server__tool` namespacing + collision detection + truncacao a 64 chars
- **Ajuste minimo:** Nenhum -- logica pura
- **Tempo integrar:** 2h (incluindo testes)

```go
// pkg/mcp/names.go
type NameRegistry struct { used map[string]bool }

func (r *NameRegistry) SafeName(serverName, toolName string) string {
    base := sanitize(serverName) + "__" + sanitize(toolName)
    if len(base) > 64 { base = base[:64] }
    if !r.used[base] { r.used[base] = true; return base }
    for i := 2; ; i++ {
        candidate := fmt.Sprintf("%s-%d", base[:min(60,len(base))], i)
        if !r.used[candidate] { r.used[candidate] = true; return candidate }
    }
}
```

### 1.5 Docker Sandbox Config Snippet
- **Origem:** `sandbox-security.md`
- **LOC estimado:** 30 LOC bash/Go
- **O que faz:** Template `docker run` com cap-drop, seccomp, memory, pids-limit
- **Ajuste minimo:** Substituir paths de workspace
- **Tempo integrar:** 1 dia (integracao com python_bridge)

```bash
docker run --rm \
  --cap-drop=ALL --cap-add=NET_BIND_SERVICE \
  --security-opt=no-new-privileges:true \
  --memory=512m --cpus=1 --pids-limit=256 \
  --network=none --user=1000:1000 --read-only \
  --tmpfs=/tmp:rw,size=100m \
  -v "${workspace}:/workspace:rw" \
  debian:bookworm-slim /bin/bash -c "$cmd"
```

### 1.6 CI Artifact Gate Script
- **Origem:** `scripts-and-tooling.md` -> `pr-lib/gates.sh` (100 LOC)
- **LOC estimado:** 80 LOC bash
- **O que faz:** Orquestra gates de qualidade pre-merge: build, test, lint, coverage
- **Ajuste minimo:** Substituir comandos vitest por `go test`, recipe validator Go
- **Tempo integrar:** 2h

### 1.7 Job Timing Summary (CI)
- **Origem:** `scripts-and-tooling.md` -> `ci-run-timings.mjs`
- **LOC estimado:** 50 LOC Go
- **O que faz:** Parse de timings de jobs, top-15 mais lentos, output formatado
- **Ajuste minimo:** Input de systemd journal em vez de GitHub Actions JSON
- **Tempo integrar:** 2h

---

## 2. PORT-AND-ADAPT (traducao pra Go idiomatico)

### 2.1 Memory Composite Scorer (Recency Half-Life)
- **Origem:** `memory-system.md` -> `extensions/memory-core/src/short-term-promotion.ts` (~500 LOC)
- **LOC estimado:** 200 LOC Go
- **O que faz:** Ranking multi-componente: frequency + relevance + diversity + recency + consolidation + conceptual
- **Ajuste minimo:** Storage fica em Supabase pgvector (nao SQLite local)
- **Tempo integrar:** 3-4 dias

```go
// pkg/memory/scorer.go
type RecallScore struct {
    Frequency, Relevance, Diversity float32
    Recency, Consolidation          float32
}

var DefaultWeights = RecallWeights{
    Frequency: 0.20, Relevance: 0.25, Diversity: 0.15,
    Recency: 0.15, Consolidation: 0.25,
}

func RecencyScore(ageDays float64, halfLifeDays int) float32 {
    return float32(math.Exp(-ageDays * math.Log(2) / float64(halfLifeDays)))
}

func (r RecallScore) Composite(w RecallWeights) float32 {
    return r.Frequency*w.Frequency + r.Relevance*w.Relevance +
           r.Diversity*w.Diversity + r.Recency*w.Recency +
           r.Consolidation*w.Consolidation
}
```

### 2.2 Episode Auto-Eviction (Retention Policy)
- **Origem:** `memory-system.md` -> `qmd-manager.ts` L2100-2150
- **LOC estimado:** 100 LOC Go
- **O que faz:** Deleta episodes por MaxAgeDays e MaxPerScope via Supabase HTTP
- **Ajuste minimo:** HTTP DELETE via Supabase PostgREST em vez de SQLite
- **Tempo integrar:** 2 dias

```go
// pkg/memory/retention.go
type RetentionPolicy struct {
    MaxAgeDays, MaxPerScope, HalfLifeDays int
}

func (s *SupabaseProvider) EvictStaleEpisodes(ctx context.Context, tenantID string, p RetentionPolicy) (int, error) {
    cutoff := time.Now().AddDate(0, 0, -p.MaxAgeDays).Format(time.RFC3339)
    // DELETE /episodes?created_at=lt.{cutoff}&tenant_id=eq.{tenantID}
}
```

### 2.3 Wave-Level Model Fallback Policy
- **Origem:** `src-agents.md` -> `run-executor.ts` Pattern B
- **LOC estimado:** 80 LOC Go
- **O que faz:** Se task falha em Haiku -> retry Sonnet -> retry Opus; persiste escolha
- **Ajuste minimo:** Integrar no loop `runRecipeInternal()` de `pkg/engine/recipe_runner.go`
- **Tempo integrar:** 2-3 dias

```go
// adicionar em pkg/core/contracts.go
type Wave struct {
    Steps          []Step
    FallbackModels []string  // ["haiku", "sonnet", "opus"]
}
```

### 2.4 Go Import Cycle Detector
- **Origem:** `scripts-and-tooling.md` -> `check-import-cycles.ts` (150 LOC)
- **LOC estimado:** 120 LOC Go
- **O que faz:** DFS no grafo de dependencias Go, detecta ciclos, output human-readable
- **Ajuste minimo:** `golang.org/x/tools/go/packages` em vez de TS AST
- **Tempo integrar:** 1-2 dias

### 2.5 MCP Tool Cache (TTL 30s por session)
- **Origem:** `mcp-integration.md` -> `src/gateway/mcp-http.ts`
- **LOC estimado:** 60 LOC Go
- **O que faz:** Cache de tool schemas por sessionKey + provider + accountId, TTL 30s
- **Ajuste minimo:** `sync.Map` em vez de Map JS
- **Tempo integrar:** 1 dia

### 2.6 Global Hook/Event Bus
- **Origem:** `infra-config-bootstrap.md` -> `src/hooks/`
- **LOC estimado:** 80 LOC Go
- **O que faz:** Registry global de handlers por tipo de evento, serial async, sem mutacao de estado
- **Ajuste minimo:** Nenhum -- padrao puro de observer
- **Tempo integrar:** 1 dia

```go
// pkg/infra/hooks.go
type HookEvent struct {
    Type, Action string
    Context      map[string]interface{}
}
type HookHandler func(ctx context.Context, event HookEvent) error

func RegisterHook(eventKey string, h HookHandler) { ... }
func TriggerHook(ctx context.Context, eventKey string, event HookEvent) { ... }
```

### 2.7 Plugin Multi-Slot Registry
- **Origem:** `plugin-sdk-design.md` -> `src/plugins/registry-types.ts`
- **LOC estimado:** 150 LOC Go
- **O que faz:** Registry Tools + StepTypes + Executors + Gates + Hooks por plugin
- **Ajuste minimo:** Sem isolamento de heap (manter subprocess como esta)
- **Tempo integrar:** 3-4 dias

### 2.8 E2E Test Harness (spawn pm-api/engine)
- **Origem:** `qa-test-infra.md` -> `test/helpers/gateway-e2e-harness.ts`
- **LOC estimado:** 200 LOC Go
- **O que faz:** Spawn pm-api + pm-engine em portas efemeras, health-check, cleanup AfterAll
- **Ajuste minimo:** `exec.Command` Go em vez de Node `spawn`
- **Tempo integrar:** 3 dias

### 2.9 Recipe Scenario Pack (Markdown Registry)
- **Origem:** `qa-test-infra.md` -> `qa/scenarios/` + frontmatter
- **LOC estimado:** 50 LOC Go runner + N arquivos .md
- **O que faz:** Registry de cenarios de teste com frontmatter YAML
- **Ajuste minimo:** `.v2.json` snippets + runner Go
- **Tempo integrar:** 2 dias

### 2.10 LLM Provider Interface (Multi-provider)
- **Origem:** `llm-providers.md` -> `plugin-sdk/provider-stream-shared.ts`
- **LOC estimado:** 120 LOC Go base + 150 LOC por provider
- **O que faz:** Interface unificada Execute/Stream + ProviderModel com cost tracking
- **Ajuste minimo:** HTTP-only (sem SDK); OpenAI-compatible unifica Gemini, OpenRouter, Ollama
- **Tempo integrar:** 3-4 dias por provider novo

```go
// pkg/engine/adapters/provider.go
type LLMProvider interface {
    Name() string
    IsAvailable(ctx context.Context) bool
    Execute(ctx context.Context, wi WorkItem) (*TaskResult, error)
}

type ProviderModel struct {
    ID, Provider, API  string
    CostInput, CostOutput float64  // $/1K tokens
    MaxTokens          int
}
```

---

## 3. CAPABILITIES JSON (campos/primitives pra recipe schema)

### 3.1 `fallback_models` na Wave
```json
{
  "steps": [...],
  "fallback_models": ["claude-haiku-4-5", "claude-sonnet-4-6", "claude-opus-4-7"]
}
```
- **Habilita:** Retry automatico com modelo mais potente se task falha
- **Effort:** 2-3 dias

### 3.2 `requires` na Recipe (pre-flight checks)
```json
{
  "slug": "code-review",
  "requires": {
    "tools": ["git", "gh"],
    "env_vars": ["GITHUB_TOKEN"],
    "services": ["supabase", "gcs"]
  }
}
```
- **Habilita:** Pre-flight check antes de executar; falha rapido se deps ausentes
- **Effort:** 1 dia

### 3.3 `retention` na Recipe Quality Config
```json
{
  "quality": {
    "retention": { "max_age_days": 30, "max_per_scope": 100, "half_life_days": 7 }
  }
}
```
- **Habilita:** Auto-eviction de Episodes velhos por recipe
- **Effort:** 2 dias

### 3.4 `trigger_phrases` (convencao no description)
```json
{
  "description": "Review code.\nUse when: (1) PR submitted, (2) feedback requested.\nNOT for: design reviews."
}
```
- **Habilita:** Melhor matching no Raven v2; autodocumentacao
- **Effort:** 0 (convencao de texto, nao requer codigo)

### 3.5 `collection_hint` na Scope de memoria
```json
{
  "scope": { "tenant_id": "t1", "recipe": "code-review", "collection_hint": "sessions" }
}
```
- **Habilita:** Particionamento semantico de memoria (short-term vs long-term)
- **Effort:** 2 dias

### 3.6 Skill Metadata Frontmatter (YAML padrao)
```yaml
---
name: code-reviewer
description: |
  Use when: (1) PR feedback, (2) quality gate. NOT for: design reviews.
metadata:
  pmos:
    requires_tools: ["git", "gh"]
    requires_env: ["GITHUB_TOKEN"]
    emoji: "code"
    install:
      - label: "Install gh CLI"
        command: "brew install gh"
---
```
- **Habilita:** Discovery estruturado de skills; auto-prompting de deps faltantes
- **Effort:** 1 dia (convencao + parser YAML)

### 3.7 `sandbox` na Recipe Step
```json
{
  "id": "run-code",
  "type": "function",
  "sandbox": { "mode": "container", "memory": "512m", "cpus": 1, "network": "none" }
}
```
- **Habilita:** Isolamento Docker por step
- **Effort:** 2-3 dias

---

## 4. SCRIPTS DE TOOLING

### 4.1 `tools/pm-bench/startup.go`
- **Origem:** `bench-cli-startup.ts`
- **O que faz:** Mede cold-start de pm-api/pm-engine, p50/p95, budget enforcement (<200ms)
- **Como adaptar:** `exec.Command("./pm-api", "--dry-run")` em loop de 10 runs; output JSON

### 4.2 `tools/pm-bench/recipe-throughput.go`
- **Origem:** `bench-model.ts`
- **O que faz:** Mede latencia de recipe, wave throughput, token budgeting
- **Como adaptar:** POST /api/v2/run + poll /api/runs/{id}; medir start->end por wave

### 4.3 `scripts/ci-gates.sh`
- **Origem:** `pr-lib/gates.sh`
- **O que faz:** Orquestra pre-merge: recipe validation + cycles + coverage + perf budget
- **Como adaptar:** Trocar vitest por `go test ./...`

```bash
#!/usr/bin/env bash
set -euo pipefail
go run ./cmd/validate-recipes/... recipes/*.v2.json || exit 1
go run tools/pm-audit/cycle.go ./... || exit 1
./scripts/coverage-check.sh || exit 1
go run tools/pm-bench/startup.go --budget 200ms || exit 1
echo "All gates: PASS"
```

### 4.4 `tools/pm-audit/cycle.go`
- **Origem:** `check-import-cycles.ts`
- **O que faz:** Detecta ciclos de import Go, output human-readable com path do ciclo
- **Como adaptar:** `golang.org/x/tools/go/packages` + DFS

### 4.5 `scripts/e2e/docker-compose.e2e.yml`
- **Origem:** `docker-compose.yml`
- **O que faz:** Stack completo: Supabase Docker + pm-api + pm-engine + Caddy
- **Como adaptar:** Volumes pra /home/pmos; pm-api/pm-engine como Go binaries

---

## 5. CLI COMMANDS NOVAS

### 5.1 `pm-cli doctor`
- **Proposito:** Diagnostica e repara configuracao PM-OS
- **Comportamento:**
  1. Valida credenciais: Supabase, Anthropic, GCS (curl test)
  2. Checa schema de recipes: `go run cmd/validate-recipes/main.go`
  3. Detecta runs presas (`status=running` > 30min)
  4. Verifica systemd units: `systemctl is-active pmos-api pmos-engine`
  5. `--fix`: auto-repara (restart services, reset stale runs)
- **Flags:** `--fix`, `--verbose`, `--format=json`
- **Effort:** 3 dias

### 5.2 `pm-cli init --wizard`
- **Proposito:** Setup interativo de nova instancia PM-OS (charmbracelet/huh para prompts)
- **Comportamento:** Section 1: API keys | Section 2: Workspace | Section 3: Executor choice
- **Flags:** `--skip-validation`, `--reset`, `--quick`
- **Effort:** 4 dias

### 5.3 `pm-cli bench`
- **Proposito:** Benchmarks de performance do stack
- **Comportamento:**
  - `pm-cli bench startup` -- mede cold-start pm-api (p50/p95/p99)
  - `pm-cli bench recipe [slug]` -- mede end-to-end de recipe
  - `pm-cli bench executor` -- compara latencia por executor
- **Flags:** `--runs=N`, `--format=json|table`, `--budget=Xms`
- **Effort:** 2-3 dias

### 5.4 `pm-cli mcp list`
- **Proposito:** Gerencia MCP servers em runtime (add/remove/list sem restart)
- **Comportamento:**
  - `pm-cli mcp list` -- lista servers + status
  - `pm-cli mcp show [name]` -- config + tools disponiveis
  - `pm-cli mcp set [name] [json]` -- adiciona/atualiza
  - `pm-cli mcp unset [name]` -- remove
- **Effort:** 2 dias

### 5.5 `pm-cli memory`
- **Proposito:** Gerencia knowledge base de Episodes
- **Comportamento:**
  - `pm-cli memory search "[query]"` -- busca semantica em Episodes
  - `pm-cli memory status` -- count por scope, tamanho
  - `pm-cli memory evict --max-age-days=30` -- forca eviction
- **Effort:** 2 dias

---

## 6. TESTING UTILITIES

### 6.1 `pkg/testing/fixtures.go` -- Mock Registry
- **Origem:** `qa-test-infra.md` -> `contracts-testkit.ts`
- **O que faz:** `NewMockExecutor()`, `NewMockGate()`, `NewMockStore()` com `assert.Called()`
- **LOC estimado:** 150 LOC Go
- **Tempo integrar:** 2 dias

```go
// pkg/testing/mock_executor.go
type MockExecutor struct {
    ExecuteFn func(context.Context, core.WorkItem) (*core.TaskResult, error)
    Calls     []core.WorkItem
}
func (m *MockExecutor) Execute(ctx context.Context, wi core.WorkItem) (*core.TaskResult, error) {
    m.Calls = append(m.Calls, wi)
    if m.ExecuteFn != nil { return m.ExecuteFn(ctx, wi) }
    return &core.TaskResult{Output: "mock output", Status: "completed"}, nil
}
func (m *MockExecutor) AssertCalledTimes(t *testing.T, n int) {
    assert.Len(t, m.Calls, n)
}
```

### 6.2 `pkg/testing/e2e_harness.go` -- E2E Process Harness
- **Origem:** `gateway-e2e-harness.ts`
- **O que faz:** Spawn pm-api + pm-engine em portas efemeras, health-check, cleanup AfterAll
- **LOC estimado:** 200 LOC Go
- **Tempo integrar:** 3 dias

### 6.3 `recipes/qa-scenarios/` -- Scenario Pack
- **Origem:** `qa/scenarios/` + frontmatter
- **O que faz:** `.md` files com frontmatter YAML (slug, expected output, gates)
- **LOC estimado:** 30 LOC Go runner + N arquivos .md

```markdown
---
name: recipe-hello-world
recipe_slug: hello-world
expected_status: completed
expected_output_contains: "Hello World"
gates: [not_empty, min_length_10]
---
Tests minimal recipe execution with single LLM step.
```

---

## 7. SKILL PATTERNS

### 7.1 Estrutura de Diretorio SKILL.md padrao PM-OS

```
skills/
  my-skill/
    SKILL.md          # Frontmatter YAML + body (<500 linhas)
    scripts/          # Code executavel (NAO carregado no context)
      validate.go
    references/       # Docs carregados on-demand
      api-schema.md
    assets/           # Templates de output
      template.json
```

### 7.2 Frontmatter Fields Canonicas

```yaml
---
name: skill-name
description: |
  Use when: (1) case1, (2) case2.
  NOT for: X, Y.
metadata:
  pmos:
    requires_tools: ["git", "gh"]
    requires_env: ["GITHUB_TOKEN"]
    requires_config: []
    install:
      - label: "Install gh CLI"
        command: "brew install gh"
    primary_auth: "GITHUB_TOKEN"
    version: "1.0.0"
---
```

### 7.3 Discovery Mechanics

1. Raven v2 escaneia `skills/*/SKILL.md` frontmatter por `name` + `description`
2. Checa `requires_tools` contra PATH do worker
3. Checa `requires_env` contra env vars disponiveis
4. Match: injeta skill body como `SKILL_CONTEXT:` em `WorkItem.Constraints`
5. References carregadas on-demand

### 7.4 3-Layer Loading (Context Efficiency)

| Layer | Conteudo | Quando carrega |
|-------|---------|----------------|
| Metadata | name + description (~100 words) | Sempre (pre-dispatch) |
| SKILL.md body | Instrucoes (~500 linhas) | Quando match detectado |
| References | Schemas, API docs (ilimitado) | On-demand por agent |
| Scripts | Codigo executavel | Executado sem carregar |

---

## 8. DOCS/I18N PATTERNS

### 8.1 Frontmatter `read_when` + `summary` para docs PM-OS

```markdown
---
summary: "Quick setup for PicoClaw executor integration"
read_when:
  - Setting up PicoClaw as primary LLM executor
  - Debugging PicoClaw auth or timeout errors
title: "PicoClaw Executor Guide"
---
```

Aplicar em todos os guias em `/docs/`: recipes, quality gates, executors.

### 8.2 Glossario PT-BR (PM-OS terms)

Criar `docs/.i18n/glossary.pt-BR.json`:
```json
{
  "Recipe": "Recipe",
  "Wave": "Wave",
  "Gate": "Gate (verificador)",
  "Executor": "Executor",
  "Brain": "Brain (classificador)",
  "Veto": "Veto",
  "Episode": "Episode (memoria de tarefa)",
  "Reconciler": "Reconciler (verificador periodico)"
}
```

### 8.3 2-Tier Troubleshooting Pattern

```markdown
# Primeiro 60 segundos se algo quebrou
1. `systemctl status pmos-api pmos-engine`
2. `pm-cli doctor`
3. `curl http://localhost:8080/health`
4. `pm-cli mcp list`
5. `journalctl -u pmos-api -n 50`
```

---

## 9. SPECIFIC CODE SNIPPETS (Top 10)

### Snippet 1: Benchmark Percentile Stats (~40 LOC)
```go
// tools/pm-bench/stats.go
package pmbench

import "sort"

type Stats struct {
    P50, P95, P99  int64
    Min, Max, Mean int64
    Samples        int
}

func Calculate(rawNs []int64) Stats {
    n := len(rawNs)
    if n == 0 { return Stats{} }
    s := make([]int64, n)
    copy(s, rawNs)
    sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
    var sum int64
    for _, v := range s { sum += v }
    return Stats{
        P50: s[(n*50)/100], P95: s[(n*95)/100], P99: s[(n*99)/100],
        Min: s[0], Max: s[n-1], Mean: sum / int64(n), Samples: n,
    }
}
```

### Snippet 2: Global Hook Registry (~80 LOC)
```go
// pkg/infra/hooks.go
package infra

import (
    "context"
    "sync"
)

type HookEvent struct {
    Type    string
    Action  string
    Context map[string]interface{}
}

type HookHandler func(ctx context.Context, event HookEvent) error

var (
    hookHandlers = make(map[string][]HookHandler)
    hookMu       sync.RWMutex
)

func RegisterHook(eventKey string, h HookHandler) {
    hookMu.Lock(); defer hookMu.Unlock()
    hookHandlers[eventKey] = append(hookHandlers[eventKey], h)
}

func TriggerHook(ctx context.Context, eventKey string, event HookEvent) {
    hookMu.RLock()
    handlers := make([]HookHandler, len(hookHandlers[eventKey]))
    copy(handlers, hookHandlers[eventKey])
    hookMu.RUnlock()
    for _, h := range handlers {
        go func(fn HookHandler) { fn(ctx, event) }(h)
    }
}

func ClearHooks() {
    hookMu.Lock(); defer hookMu.Unlock()
    hookHandlers = make(map[string][]HookHandler)
}
```

### Snippet 3: MCP Safe Tool Name (~50 LOC)
```go
// pkg/mcp/names.go
package mcp

import (
    "fmt"
    "regexp"
    "strings"
)

var nonAlphanumRe = regexp.MustCompile(`[^A-Za-z0-9_-]`)

type NameRegistry struct{ used map[string]bool }

func NewNameRegistry() *NameRegistry { return &NameRegistry{used: make(map[string]bool)} }

func sanitize(s string) string {
    s = nonAlphanumRe.ReplaceAllString(s, "-")
    if len(s) > 30 { s = s[:30] }
    return strings.TrimRight(s, "-")
}

func minInt(a, b int) int { if a < b { return a }; return b }

func (r *NameRegistry) SafeName(serverName, toolName string) string {
    base := sanitize(serverName) + "__" + sanitize(toolName)
    if len(base) > 64 { base = base[:64] }
    if !r.used[base] { r.used[base] = true; return base }
    for i := 2; ; i++ {
        candidate := fmt.Sprintf("%s-%d", base[:minInt(60, len(base))], i)
        if !r.used[candidate] { r.used[candidate] = true; return candidate }
    }
}
```

### Snippet 4: Recency Half-Life Decay (~30 LOC)
```go
// pkg/memory/scorer.go
package memory

import "math"

func RecencyScore(ageDays float64, halfLifeDays int) float32 {
    if halfLifeDays <= 0 { halfLifeDays = 7 }
    score := math.Exp(-ageDays * math.Log(2) / float64(halfLifeDays))
    return float32(math.Max(0, math.Min(1, score)))
}

type RecallScore struct {
    Frequency, Relevance, Diversity, Recency, Consolidation float32
}

type RecallWeights struct {
    Frequency, Relevance, Diversity, Recency, Consolidation float32
}

var DefaultWeights = RecallWeights{
    Frequency: 0.20, Relevance: 0.25, Diversity: 0.15,
    Recency: 0.15, Consolidation: 0.25,
}

func (r RecallScore) Composite(w RecallWeights) float32 {
    return r.Frequency*w.Frequency + r.Relevance*w.Relevance +
        r.Diversity*w.Diversity + r.Recency*w.Recency +
        r.Consolidation*w.Consolidation
}
```

### Snippet 5: Episode Retention Policy (~60 LOC)
```go
// pkg/memory/retention.go
package memory

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

type RetentionPolicy struct {
    MaxAgeDays, MaxPerScope, HalfLifeDays int
}

var DefaultRetentionPolicy = RetentionPolicy{MaxAgeDays: 90, MaxPerScope: 500, HalfLifeDays: 7}

func (s *SupabaseMemoryStore) EvictStaleEpisodes(
    ctx context.Context, tenantID string, p RetentionPolicy,
) error {
    if p.MaxAgeDays <= 0 { return nil }
    cutoff := time.Now().AddDate(0, 0, -p.MaxAgeDays).Format(time.RFC3339)
    url := fmt.Sprintf("%s/episodes?created_at=lt.%s&tenant_id=eq.%s",
        s.baseURL, cutoff, tenantID)
    req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
    req.Header.Set("Authorization", "Bearer "+s.apiKey)
    resp, err := s.httpClient.Do(req)
    if err != nil { return fmt.Errorf("evict episodes: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return fmt.Errorf("evict episodes: status %d", resp.StatusCode)
    }
    return nil
}
```

### Snippet 6: Wave Fallback Model Policy (~50 LOC)
```go
// em pkg/engine/recipe_runner.go
func (e *Engine) executeStepWithFallback(
    ctx context.Context, step recipe.Step, wave recipe.Wave,
    allOutputs map[string]string,
) (*core.TaskResult, error) {
    models := wave.FallbackModels
    if len(models) == 0 { models = []string{step.Provider} }
    var lastErr error
    for attempt, model := range models {
        wi := e.buildWorkItem(step, allOutputs)
        wi.Provider = model
        result, err := e.executor.Execute(ctx, wi)
        if err == nil { return result, nil }
        lastErr = err
        e.log.Warnf("step %s fallback attempt %d/%d (model=%s): %v",
            step.ID, attempt+1, len(models), model, err)
        if attempt < len(models)-1 {
            time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
        }
    }
    return nil, fmt.Errorf("step %s failed after %d fallbacks: %w",
        step.ID, len(models), lastErr)
}
```

### Snippet 7: Docker Sandbox Wrapper (~40 LOC bash)
```bash
# tools/python_bridge/sandbox.sh
#!/usr/bin/env bash
set -euo pipefail

SANDBOX_MODE="${PM_SANDBOX_MODE:-off}"
SANDBOX_MEMORY="${PM_SANDBOX_MEMORY:-512m}"
SANDBOX_CPUS="${PM_SANDBOX_CPUS:-1}"
WORKSPACE="${PM_WORKSPACE:-/tmp/pm-workspace}"
IMAGE="${PM_SANDBOX_IMAGE:-debian:bookworm-slim}"

run_sandboxed() {
  local script="$1"; shift
  if [ "$SANDBOX_MODE" = "off" ]; then python3 "$script" "$@"; return; fi
  docker run --rm \
    --cap-drop=ALL \
    --security-opt=no-new-privileges:true \
    "--memory=${SANDBOX_MEMORY}" "--cpus=${SANDBOX_CPUS}" \
    --pids-limit=256 --network=none \
    --user=1000:1000 --read-only \
    --tmpfs=/tmp:rw,size=100m \
    -v "${WORKSPACE}:/workspace:rw" \
    "$IMAGE" python3 /workspace/script.py "$@"
}
```

### Snippet 8: MCP Tool Cache (~60 LOC)
```go
// pkg/mcp/cache.go
package mcp

import (
    "fmt"
    "sync"
    "time"
)

type toolCacheEntry struct {
    tools     []Tool
    expiresAt time.Time
}

type ToolCache struct {
    mu      sync.Mutex
    entries map[string]*toolCacheEntry
    ttl     time.Duration
}

func NewToolCache(ttl time.Duration) *ToolCache {
    if ttl <= 0 { ttl = 30 * time.Second }
    return &ToolCache{entries: make(map[string]*toolCacheEntry), ttl: ttl}
}

func (c *ToolCache) Get(sessionKey, provider, accountID string, isOwner bool) ([]Tool, bool) {
    c.mu.Lock(); defer c.mu.Unlock()
    key := fmt.Sprintf("%s|%s|%s|%v", sessionKey, provider, accountID, isOwner)
    e, ok := c.entries[key]
    if !ok || time.Now().After(e.expiresAt) { delete(c.entries, key); return nil, false }
    return e.tools, true
}

func (c *ToolCache) Set(sessionKey, provider, accountID string, isOwner bool, tools []Tool) {
    c.mu.Lock(); defer c.mu.Unlock()
    key := fmt.Sprintf("%s|%s|%s|%v", sessionKey, provider, accountID, isOwner)
    c.entries[key] = &toolCacheEntry{tools: tools, expiresAt: time.Now().Add(c.ttl)}
}
```

### Snippet 9: Mock Executor para Testes (~50 LOC)
```go
// pkg/testing/mock_executor.go
package pmtesting

import (
    "context"
    "testing"
)

type MockExecutor struct {
    ExecuteFn func(context.Context, core.WorkItem) (*core.TaskResult, error)
    Calls     []core.WorkItem
    ReturnErr error
    ReturnOut string
}

func NewMockExecutor(output string) *MockExecutor {
    return &MockExecutor{ReturnOut: output}
}

func (m *MockExecutor) Execute(ctx context.Context, wi core.WorkItem) (*core.TaskResult, error) {
    m.Calls = append(m.Calls, wi)
    if m.ExecuteFn != nil { return m.ExecuteFn(ctx, wi) }
    if m.ReturnErr != nil { return nil, m.ReturnErr }
    return &core.TaskResult{Output: m.ReturnOut, Status: "completed"}, nil
}

func (m *MockExecutor) AssertCalledTimes(t *testing.T, n int) {
    t.Helper()
    if len(m.Calls) != n {
        t.Errorf("expected Execute called %d times, got %d", n, len(m.Calls))
    }
}

func (m *MockExecutor) AssertCalledWith(t *testing.T, title string) {
    t.Helper()
    for _, wi := range m.Calls {
        if wi.Title == title { return }
    }
    t.Errorf("Execute not called with title %q", title)
}
```

### Snippet 10: CI Artifact Gate (~50 LOC bash)
```bash
# scripts/ci-gates.sh
#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
pass() { echo -e "${GREEN}ok $1${NC}"; }
fail() { echo -e "${RED}FAIL $1${NC}"; exit 1; }

require_file() { [ -f "$1" ] && [ -s "$1" ] || fail "Missing: $1"; pass "Artifact: $1"; }

gate_recipes() {
  go run ./cmd/validate-recipes/... recipes/*.v2.json && pass "Recipes valid" || fail "Recipes"
}
gate_tests() {
  go test ./... -timeout 60s && pass "Tests passed" || fail "Tests"
}
gate_coverage() {
  ./scripts/coverage-check.sh && pass "Coverage ok" || fail "Coverage"
}
gate_build() {
  go build ./cmd/pm-api ./cmd/pm-engine && pass "Build ok" || fail "Build"
}

echo "=== PM-OS CI Gates ==="
gate_build; gate_recipes; gate_tests; gate_coverage
echo "=== All gates PASS ==="
```

---

## 10. DESCARTADO

| Item | Motivo descarte |
|------|-----------------|
| A2UI Canvas (iOS/Android) | Platform gap total: requer WebView nativo + bridge iOS/Android |
| OpenClaw Gateway WebSocket broker | PM-OS ja tem HTTP REST + SSE; reescrever seria regressao |
| Flows (config menus) | Flows = wizard UI de configuracao; PM-OS nao tem UI generation |
| Memory Wiki (Obsidian vault) | Knowledge vault human-authored; sem relevancia pra execucao de recipes |
| Lobster DSL | PM-OS usa JSON recipes; nenhum beneficio de DSL adicional |
| pty:true / process tool | Terminal multiplexing; PM-OS e HTTP/subprocess sem TTY |
| QMD CLI + SQLite local | PM-OS e multi-tenant cloud-first; pgvector Supabase e superior |
| Plugin-npm-release-check | PM-OS e Go module, nao npm package |
| Daemon launchd/schtasks | PM-OS usa systemd local; Windows/macOS nao e target atual |
| Channel integrations completas (Telegram/Discord porter) | Cada canal = 3 dias; prioridade baixa vs core engine; implementar depois |
| gVisor sandbox | Overhead 2-3x + nao suportado no Cloud Run; Docker socket e suficiente |
| Plugin shared-heap (Node.js) | PM-OS ja usa subprocess boundary; melhor isolamento |
| i18n GH Actions automated | Docs nao estao em ingles ainda; prematuro |
| LanceDB embedding backend | PM-OS usa pgvector REST; LanceDB e para local/stateless |
| Interactive Clack TUI (Node) | Requer Node.js; usar charmbracelet/huh Go em vez |
| OpenClaw client iOS/Android | PM-OS e B2B server orchestration; mobile nao e target |
| Subagent registry completo | Complexidade alta; PM-OS depends_on ja e suficiente |
| QA-Lab TS runtime | Requer TypeScript gateway; adaptar para e2e harness Go (listado na secao 2) |
| Clack-based CLI (Node) | Substituir por charmbracelet/bubbletea/huh em Go nativo |

---

## RESUMO: TOP 15 por ROI

Ordenado por (tempo adapta curto) x (valor alto) descendente:

| # | Item | Tempo | Valor |
|---|------|-------|-------|
| 1 | MCP Tool Name Registry (Snippet 3) | 2h | Elimina colisoes silenciosas em MCP |
| 2 | Global Hook/Event Bus (Snippet 2) | 1 dia | Desacopla callbacks do engine loop; base pra plugins |
| 3 | Skill Metadata Frontmatter + `requires` | 1 dia | Discovery estruturado, pre-flight checks automaticos |
| 4 | scripts/ci-gates.sh (Snippet 10) | 2h | Gates pre-merge em 1 arquivo orchestrado |
| 5 | Recency Half-Life Decay + Composite Score (Snippets 4) | 2 dias | Memory ranking dramaticamente melhor |
| 6 | MCP Tool Cache 30s TTL (Snippet 8) | 1 dia | Elimina 30 schema calls por wave |
| 7 | Wave Fallback Model Policy (Snippet 6 + JSON 3.1) | 2 dias | Auto-retry com modelo superior sem mudar recipes |
| 8 | Episode Auto-Eviction (Snippet 5 + JSON 3.3) | 2 dias | Episodes nao acumulam indefinidamente no Supabase |
| 9 | pm-cli doctor | 3 dias | Onboarding + troubleshooting self-service |
| 10 | Mock Executor para testes (Snippet 9) | 2 dias | Testes de adaptadores sem hand-rolled mocks |
| 11 | Docker Sandbox python_bridge (Snippet 7) | 2 dias | Isolamento real de code execution untrusted |
| 12 | Benchmark p50/p95 (Snippet 1 + pm-cli bench) | 1 dia | Baseline de performance; detecta regressions automaticamente |
| 13 | E2E Test Harness pm-api/pm-engine | 3 dias | Testes distribuidos end-to-end possiveis |
| 14 | Recipe Scenario Pack (`recipes/qa-scenarios/`) | 2 dias | Coverage tracking automatico de recipes |
| 15 | pm-cli init --wizard | 4 dias | Onboarding melhor para novos usuarios |

---

*Gerado: 2026-04-23*
*Total analisado: 19 notes, ~8.000 LOC de analises*
*Items selecionados: 38 concretos, 19 descartados*
