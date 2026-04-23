# PM-OS — Contexto Alvo

**O que é PM-OS:** motor de orquestração paralela baseado em recipes. Multi-tenant, Go puro, stateless HTTP via Supabase PostgREST.

## Stack

- **Linguagem:** Go 1.25.0
- **Web:** `net/http` stdlib only (sem framework)
- **Storage:** Supabase PostgREST via HTTP (nunca Postgres direto)
- **Deploy:** Cloud Run (3 serviços — pm-api, pm-engine, pm-mcp)
- **Testes:** stdlib `testing` + `github.com/stretchr/testify` v1.11.1
- **Config deps:** `go.mod` do PM-OS permite `google.golang.org/api`, `github.com/robfig/cron/v3`, `github.com/anthropics/anthropic-sdk-go`, `golang.org/x/oauth2`, OpenTelemetry, `google/uuid`
- **Forbid:** ORMs, pydantic-style runtime validators, reflection-heavy libs, CGO

## Estrutura de pacotes existente (NÃO QUEBRAR)

```
pm-os/
├── cmd/
│   ├── pm-api/                 # coordinator service
│   └── pm-engine/              # executor service  (se existir)
├── pkg/
│   ├── core/                   # Plan, Wave, WorkItem, TaskResult, GateResult contracts
│   ├── recipe/                 # Recipe schema, validator, compiler, catalog, instantiate
│   ├── engine/                 # Engine struct, builder, runRecipeInternal
│   │   ├── adapters/           # Executor, Gate, Browser, Store implementations
│   │   └── tools/              # Tool Registry, python_bridge, builtin_math, etc.
│   ├── store/                  # Supabase PostgREST client + typed stores (runs, tasks, usage)
│   ├── quality/                # Gates, spot_check, opus_review, conflict_resolver
│   ├── planner/                # Raven enricher, brain matcher, knowledge
│   ├── git/                    # Isolated git workspace
│   ├── sensors/                # Cron/webhook/watcher triggers
│   ├── infra/                  # GCS, Logger
│   └── api/                    # HTTP handlers
```

## Contratos essenciais (leia antes de portar)

### `pkg/core/contracts.go` — tipos raiz

```go
type Plan struct {
    Waves []Wave `json:"waves"`
    // ...
}

type Wave struct {
    WaveID  string     `json:"wave_id"`
    Items   []WorkItem `json:"items"`
    // ...
}

type WorkItem struct {
    ID           string            `json:"id"`
    Title        string            `json:"title"`
    Type         string            `json:"type"`  // "llm" | "function" | "llm_call" | "review" | "clarify" | "llm_agent" | "llm_with_tools"
    Provider     string            `json:"provider"`
    Instructions []string          `json:"instructions"`
    Acceptance   string            `json:"acceptance"`
    Files        []string          `json:"files"`
    DependsOn    []string          `json:"depends_on"`
    Brain        string            `json:"brain"`
    Constraints  []string          `json:"constraints"`
    Contract     *Contract         `json:"contract,omitempty"`
}

type TaskResult struct {
    TaskID    string
    Status    string    // "success" | "failure" | "skipped"
    Output    string
    Error     string
    CostUSD   float64
    TokensIn  int
    TokensOut int
    DurationMs int64
}

type GateResult struct {
    Passed  bool
    Checks  []CheckResult
    Message string
}
```

### `pkg/recipe/schema.go` — contrato recipe

```go
type Recipe struct {
    Slug    string           `json:"slug"`
    Version string           `json:"version"`
    Mode    string           `json:"mode"`  // "output-only" | "git-workspace"
    Waves   []Wave           `json:"waves"`
    Params  map[string]any   `json:"params"`
    Quality *QualityConfig   `json:"quality,omitempty"`
    Dataset *DatasetConfig   `json:"dataset,omitempty"`
}

type Wave struct {
    Name            string  `json:"name"`
    MaxConcurrency  int     `json:"max_concurrency"`
    Steps           []Step  `json:"steps"`
}

type Step struct {
    ID           string          `json:"id"`
    Type         string          `json:"type"`
    Provider     string          `json:"provider,omitempty"`
    Instructions []string        `json:"instructions"`
    DependsOn    []string        `json:"depends_on,omitempty"`
    Verify       *VerifyConfig   `json:"verify,omitempty"`
    GateChecks   []GateCheck     `json:"gate_checks,omitempty"`
    // ... e outros campos
}
```

### `pkg/engine/engine.go` — builder pattern

```go
type Engine struct {
    planner      Planner
    executor     Executor
    gates        []Gate
    store        Store
    // ... outros campos com builder WithX()
}

func New(opts ...Option) *Engine { /* ... */ }
func (e *Engine) RunRecipe(ctx context.Context, runID string, r *recipe.Recipe) (*RunResult, error) { /* ... */ }
```

### `pkg/engine/tools/registry.go` — **já existe, você vai ESTENDER**

```go
type Handler func(ctx context.Context, input json.RawMessage) (string, error)

type Tool struct {
    Name        string
    Description string
    InputSchema json.RawMessage
    Handler     Handler
    Category    string
    // <- seus campos novos aqui: ResultAsAnswer, MaxUsageCount, etc.
}

type Registry struct { /* sync.RWMutex + map */ }
var GlobalRegistry = NewRegistry()
```

### `pkg/store/client.go` — Supabase client

```go
type SupabaseClient struct {
    baseURL string
    apiKey  string
    http    *http.Client
}

// Operações: GET, POST, PATCH via PostgREST REST
```

**Storage contract:** Todo mutable state PERSISTE imediatamente via Supabase HTTP. In-process state vive só na goroutine stack.

## Convenções obrigatórias

### Nomenclatura
- **Packages:** lowercase, short, single-word preferred (`hooks`, `events`, `hitl`, `mcp`)
- **Files:** `snake_case.go`, tests `_test.go`, helpers `_testhelper_test.go`
- **Types exportados:** PascalCase, claros (`EventBus` não `EB`, `Fingerprint` não `FP`)
- **Types unexportados:** camelCase iniciando com lowercase (`eventBusImpl`)
- **Funções:** PascalCase exportadas, camelCase unexportadas
- **Constantes:** PascalCase exportadas ou `UPPER_SNAKE_CASE` para enums tipo string
- **Interfaces:** Nome pela capacidade (`Executor`, `Gate`, `Store`, `HumanFeedbackProvider`)
- **Mapa JSON:** `snake_case` em tags espelhando colunas Supabase

### Error handling

```go
// Sempre
if err != nil {
    return fmt.Errorf("operation description: %w", err)
}

// Nunca panic fora de main()
// Nunca ignore erro com _ = salvo em casos justificados (defer close)
// Sempre %w para wrap (permite errors.Is/As)
```

### Concorrência

```go
// sync.RWMutex para shared state
type Registry struct {
    mu    sync.RWMutex
    items map[string]Item
}

// context.Context primeiro param em I/O
func (r *Registry) Fetch(ctx context.Context, id string) (*Item, error)

// Goroutines com cancelamento via context
go func() {
    select {
    case <-ctx.Done():
        return
    case result := <-ch:
        // process
    }
}()

// Channels unbuffered para sync, buffered para desacoplar
```

### Logging

```go
// Use pkg/infra/logger.go pattern
logger := infra.NewLogger().With("component", "hooks").With("run_id", runID)
logger.Info("starting hook")
logger.Errorf("hook failed: %v", err)

// NÃO use log.Printf direto em library code
// Severity: Info, Warn, Error; fmt variants: Infof, Warnf, Errorf
```

### Testes

```go
package events

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEventBus_Emit(t *testing.T) {
    bus := NewEventBus()
    // ... setup

    err := bus.Emit(context.Background(), evt)
    require.NoError(t, err)
    assert.Equal(t, 1, bus.HandlerCount())
}

// Padrões:
// - Sem fixtures globais (cada teste cria o que precisa)
// - Sem sleep em tests (use channels + timeouts)
// - t.Parallel() quando possível
// - go test -race deve passar sempre
```

## Diferenças semânticas CrewAI → PM-OS (decisões chave)

### 1. "Crew" não existe em PM-OS

CrewAI `Crew` = grupo de Agents com Process. PM-OS equivalente = `Recipe` (grupo de Steps com Waves).

**Porta:** onde CrewAI diz `crew`, renomeie pra `run` em PM-OS. Ex: `CrewStartedEvent` → `RunStartedEvent`.

### 2. "Agent" não existe em PM-OS

CrewAI `Agent` tem role/goal/backstory. PM-OS não tem. Equivalente mais próximo = `Step` com `provider`.

**Porta:** onde código depende de `Agent`, simplifique pra `Step` ou omit. Eventos de agent → não portar (exceto se genérico).

### 3. Decorators → Registration API

```python
# CrewAI (decorator)
@crewai_event_bus.on(EventName)
def handler(source, event):
    ...
```

```go
// PM-OS (registration)
bus.Register(EventName{}, func(ctx context.Context, source any, event Event) error {
    return nil
})
```

### 4. LanceDB/Qdrant → Supabase pgvector

CrewAI memory usa LanceDB embedded ou Qdrant. PM-OS usa Supabase PostgREST + `pgvector` extension.

**Porta:** crie `pkg/memory/storage/pgvector_storage.go` que implementa `Backend` interface. Schema esperado:

```sql
CREATE TABLE memory_episodes (
    id uuid PRIMARY KEY,
    tenant_id text NOT NULL,
    scope text NOT NULL,
    content text NOT NULL,
    embedding vector(1536),
    metadata jsonb,
    created_at timestamptz DEFAULT now()
);
CREATE INDEX ON memory_episodes USING ivfflat (embedding vector_cosine_ops);
```

Cliente: REST via `pkg/store/client.go`, queries similares a `.filter()` PostgREST.

### 5. Asyncio → goroutines + channels

```python
# CrewAI
async def handler(event):
    await process(event)

asyncio.create_task(handler(evt))
```

```go
// PM-OS
type Handler func(ctx context.Context, event Event) error

go func() {
    if err := handler(ctx, evt); err != nil {
        logger.Errorf("handler failed: %v", err)
    }
}()
```

### 6. Pydantic → struct + validate func

```python
class Fingerprint(BaseModel):
    uuid_str: str
    metadata: Annotated[dict, BeforeValidator(_validate_metadata)]
```

```go
type Fingerprint struct {
    UUIDStr   string         `json:"uuid_str"`
    Metadata  map[string]any `json:"metadata,omitempty"`
}

func (f *Fingerprint) Validate() error {
    if err := validateMetadata(f.Metadata); err != nil {
        return fmt.Errorf("metadata: %w", err)
    }
    return nil
}
```

### 7. ContextVars → context.Context

```python
# CrewAI
_current_event_id = contextvars.ContextVar("current_event_id")
_current_event_id.set(evt.id)
```

```go
// PM-OS
type ctxKey string
const ctxKeyEventID ctxKey = "event_id"

ctx = context.WithValue(ctx, ctxKeyEventID, evt.ID)
id, _ := ctx.Value(ctxKeyEventID).(string)
```

## Dependencies permitidas (PM-OS go.mod)

```go
// Stdlib only preferred. External allowed:
"github.com/google/uuid"                       // UUIDs
"github.com/stretchr/testify"                  // testing (already there)
"github.com/anthropics/anthropic-sdk-go"       // Anthropic API
"github.com/robfig/cron/v3"                    // cron
"golang.org/x/oauth2"                          // OAuth
"google.golang.org/api"                        // GCS
"go.opentelemetry.io/otel"                     // tracing
```

**NÃO adicionar sem autorização:**
- Web frameworks (gin/echo/fiber)
- ORMs (gorm/ent)
- Pydantic-style validators (no validator.v10 unless asked)
- Clients exóticos (qdrant-go, lancedb-go) — use Supabase REST
- litellm equivalent (já removido CrewAI side too)

## Cultura PM-OS — coisas que o autor valoriza

- **Determinismo.** Mesma entrada → mesma saída. Evitar `time.Now()` em paths testáveis (use `clock` interface injetável se precisar).
- **Fail fast.** Erro de validação = rejeite com mensagem clara, não silencioso.
- **Multi-tenant first.** Todo state precisa `tenant_id`. Nunca global estado compartilhado entre tenants sem auth.
- **Observable.** Todo evento importante loga + emite metric. Sem silêncio em falha.
- **TDD hard.** Acceptance é teste que roda, não comentário.
- **Pequenos commits atômicos.** 1 arquivo portado + teste = 1 commit.
- **Sem features fantasmas.** Se CrewAI não tem, não inventa.

## Arquivos do PM-OS que você PODE ler (referência)

No repo PM-OS público você pode espiar:
- `pkg/engine/tools/registry.go` — pattern de registry
- `pkg/engine/tools/python_bridge.go` — subprocess bridge
- `pkg/core/contracts.go` — contratos
- `pkg/recipe/schema.go` — schema recipe

Use como guia de estilo.

## Arquivos do PM-OS que você NÃO DEVE modificar

- `cmd/pm-api/main.go` — Akita wires integration
- `pkg/engine/engine.go` — Akita integra
- `pkg/engine/wave_executor.go` — Akita adiciona case por step type
- `pkg/recipe/schema.go` — Akita adiciona campos (exceto novos arquivos como `task_output.go`)
- `pkg/recipe/validator.go` — Akita estende

**Crie arquivos novos, não edite os existentes.**

## Resumo

- Transcreve 60-65 arquivos selecionados do CrewAI pra Go idiomático PM-OS
- TDD obrigatório por arquivo
- Não inventa funcionalidade, não refatora arquitetura
- Reporta desvios em `PORT-REPORT.md`
- Deixa integração pro Akita
