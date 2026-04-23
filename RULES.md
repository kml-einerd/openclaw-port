# RULES — Restrições Hard + Convenções Go PM-OS

**Ler DEPOIS de `PM-OS-CONTEXT.md`, ANTES de escrever qualquer linha.**

---

## 1. Dependencies — lista fechada

### Permitido sem pedir

```go
// Go stdlib inteiro
"context", "encoding/json", "fmt", "io", "net/http", "os", "os/exec",
"sync", "time", "bytes", "strings", "strconv", "errors", "crypto/rand",
"crypto/hmac", "crypto/sha256", "encoding/base64", "encoding/hex",
"regexp", "sort", "math", "log", "path/filepath", "reflect", "unicode"

// External permitidos (já no PM-OS go.mod)
"github.com/google/uuid"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"github.com/anthropics/anthropic-sdk-go"
"github.com/robfig/cron/v3"
"golang.org/x/oauth2"
"google.golang.org/api"
"go.opentelemetry.io/otel"
```

### Precisa justificar em PORT-REPORT.md (mas pode usar)

```go
"gopkg.in/yaml.v3"                  // só pro Tier G (skills parser)
"github.com/PuerkitoBio/goquery"    // só pra HTML parse se necessário
```

### PROIBIDO sem autorização explícita

- Web frameworks: `gin`, `echo`, `fiber`, `chi`, `gorilla`
- ORMs: `gorm`, `ent`, `sqlx`, `sqlc`
- Validators: `go-playground/validator`
- Pydantic-equivalents ou runtime reflection-heavy
- Client libs exóticas: `qdrant-go`, `lancedb`, `chroma-go`
- Logging libs: `zap`, `logrus`, `zerolog` (use `pkg/infra/logger.go` pattern)
- HTTP libs alternativos: `resty`, `fasthttp`, `req`

## 2. Forbidden Python idioms

### Pydantic → struct + validate func

❌ ERRADO:
```go
type Fingerprint struct {
    UUIDStr string `validate:"required,uuid4"`
}
// usa go-playground/validator
```

✅ CERTO:
```go
type Fingerprint struct {
    UUIDStr string `json:"uuid_str"`
}

func (f *Fingerprint) Validate() error {
    if f.UUIDStr == "" {
        return errors.New("uuid_str required")
    }
    if _, err := uuid.Parse(f.UUIDStr); err != nil {
        return fmt.Errorf("uuid_str invalid: %w", err)
    }
    return nil
}
```

### Decorators → registration

❌ ERRADO (decorators não existem em Go):
```go
@LLMHook
func MyHook(ctx context.Context, req Request) error { ... }
```

✅ CERTO:
```go
func MyHook(ctx context.Context, req Request) error { ... }

func init() {
    hooks.RegisterLLM(MyHook)
}
// OU
hooks.Registry.RegisterLLM(MyHook)
```

### Metaclass / `__init_subclass__` → interface + init()

❌ ERRADO:
```go
// tentar replicar "auto-register subclasses"
type BaseEvent interface { ... }
// ...reflection magic
```

✅ CERTO:
```go
type Event interface {
    EventType() string
    Timestamp() time.Time
}

// Cada evento implementa, e init() do arquivo registra se precisar
type TaskStartedEvent struct { ... }
func (e TaskStartedEvent) EventType() string { return "task.started" }

func init() {
    events.RegisterType("task.started", func() Event { return &TaskStartedEvent{} })
}
```

### Asyncio → goroutines

❌ ERRADO (não existe em Go):
```go
async func Handler(evt Event) error { ... }
future := asyncio.CreateTask(Handler(evt))
```

✅ CERTO:
```go
type Handler func(ctx context.Context, evt Event) error

// fire-and-forget com supervisão
errCh := make(chan error, 1)
go func() {
    errCh <- handler(ctx, evt)
}()

// se precisar coletar resultado:
select {
case err := <-errCh:
    return err
case <-ctx.Done():
    return ctx.Err()
case <-time.After(timeout):
    return errors.New("handler timeout")
}
```

### ContextVars → context.Context

❌ ERRADO:
```go
var currentEventID string  // global ou thread-local-like
```

✅ CERTO:
```go
type ctxKey string
const ctxKeyEventID ctxKey = "pm-os:event_id"

// set
ctx = context.WithValue(ctx, ctxKeyEventID, evtID)

// get
func EventIDFromContext(ctx context.Context) (string, bool) {
    id, ok := ctx.Value(ctxKeyEventID).(string)
    return id, ok
}
```

### try/except → if err != nil

❌ ERRADO:
```go
defer func() {
    if r := recover(); r != nil {
        // try-except simulation
    }
}()
```

✅ CERTO (panic só para erros de programa, não de runtime):
```go
result, err := doThing()
if err != nil {
    return fmt.Errorf("doThing: %w", err)
}
```

### Dynamic typing → type parameters ou concrete types

❌ ERRADO:
```go
func Process(data interface{}) interface{} { ... }
```

✅ CERTO (generic):
```go
func Process[T any](data T) (T, error) { ... }
```

✅ CERTO (concreto):
```go
type Input struct { ... }
type Output struct { ... }
func Process(in Input) (Output, error) { ... }
```

✅ Aceitável (JSON boundary):
```go
// quando desserializando JSON genérico
func ParseJSON(raw json.RawMessage) (map[string]any, error) { ... }
```

### Mutable default args (Python bug) → construtor

❌ PYTHON (bug clássico):
```python
def __init__(self, metadata={}):  # shared mutable default
```

✅ GO:
```go
func NewX(metadata map[string]any) *X {
    if metadata == nil {
        metadata = make(map[string]any)
    }
    return &X{metadata: metadata}
}
```

---

## 3. Error handling — regras duras

### Wrapping

Sempre:
```go
if err != nil {
    return fmt.Errorf("operation name: %w", err)
}
```

### Sentinel errors para casos específicos

```go
var (
    ErrNotFound = errors.New("not found")
    ErrInvalidInput = errors.New("invalid input")
)

// Consumer check:
if errors.Is(err, ErrNotFound) { ... }
```

### Typed errors quando precisa dados

```go
type ValidationError struct {
    Field   string
    Reason  string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// Consumer:
var vErr *ValidationError
if errors.As(err, &vErr) {
    fmt.Println(vErr.Field)
}
```

### Nunca

- `panic()` em library code (apenas em `main()` ou assertion impossível)
- Ignorar erro com `_` sem comentário explicando por quê
- Retornar `(T, bool)` quando poderia retornar `(T, error)` com contexto

---

## 4. Concorrência — regras duras

### Sempre

- `context.Context` como primeiro argumento em qualquer I/O ou long-running
- `sync.RWMutex` para shared state (prefer R-locks quando possível)
- Goroutines leaked = bug. Sempre garantir cancelamento via context.
- `go test -race` passa em todos os arquivos com state concorrente

### Channels

- **Unbuffered** para sincronização (rendez-vous)
- **Buffered** para desacoplar producer/consumer
- Fechar canal é responsabilidade do producer
- Receive de canal fechado retorna valor zero + `ok == false`

### Worker pool pattern

```go
type Pool struct {
    workers int
    queue   chan Job
    wg      sync.WaitGroup
}

func (p *Pool) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(ctx)
    }
}

func (p *Pool) worker(ctx context.Context) {
    defer p.wg.Done()
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-p.queue:
            if !ok { return }
            job.Do(ctx)
        }
    }
}
```

---

## 5. Naming

| Elemento | Estilo |
|---|---|
| Packages | lowercase, curto, sem underscore (`events`, não `event_bus`) |
| Files | `snake_case.go` |
| Tests | `<name>_test.go` |
| Test helpers | `<name>_testhelper_test.go` |
| Exported types | `PascalCase` |
| Unexported types | `camelCase` |
| Methods | `PascalCase` (exp) ou `camelCase` (unexp) |
| Constants | `PascalCase` ou `UPPER_SNAKE_CASE` para enum strings |
| Interfaces | Capacidade: `Executor`, `Handler`, `Provider` (não `IExecutor`, não `ExecutorInterface`) |
| Error vars | `ErrXxx` sentinels |
| Error types | `XxxError` structs |
| Context keys | `ctxKeyXxx ctxKey = "pm-os:xxx"` |
| JSON tags | `snake_case` (espelha Supabase columns) |

---

## 6. Testes — regras duras

### Estrutura

```go
package events

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEventBus_Emit_HappyPath(t *testing.T) {
    t.Parallel()

    bus := NewEventBus()
    var received Event
    bus.Register("test.evt", func(ctx context.Context, e Event) error {
        received = e
        return nil
    })

    err := bus.Emit(context.Background(), TestEvent{ID: "x"})
    require.NoError(t, err)
    assert.NotNil(t, received)
    assert.Equal(t, "x", received.(TestEvent).ID)
}

func TestEventBus_Emit_HandlerError_Propagates(t *testing.T) { ... }
func TestEventBus_Emit_ConcurrentRegister_NoRace(t *testing.T) { ... }  // go test -race
func TestEventBus_Emit_NilEvent_ReturnsError(t *testing.T) { ... }      // edge
```

### Cobertura mínima

- Happy path: SEMPRE
- Erro esperado (validation, not found): SEMPRE
- Edge cases (nil, empty, zero): SEMPRE
- Concorrência (`go test -race`): quando state shared
- Integração com sistema externo: **mock interno no `_test.go`**, nunca subprocess ou rede real

### Não faça

- `time.Sleep()` em teste (use channels + timeouts)
- Fixtures globais (crie no test)
- `TestMain()` a menos que seja essencial
- Testes que dependem de ordem (`t.Parallel()` onde possível)
- Testes que batem em API real (use `httptest.NewServer` local)

---

## 7. Doc comments

### Package

```go
// Package events provides the PM-OS event bus with sync and async handlers,
// dependency-ordered execution plan, and context scoping.
//
// Ported from lib/crewai/src/crewai/events/ with adaptation to Go idioms:
// - asyncio handlers → goroutines
// - contextvars → context.Context values
// - singleton via sync.Once
package events
```

### Type

```go
// EventBus dispatches events to registered handlers with dependency ordering.
// Handlers registered in scoped mode are activated only while the scope is active.
//
// EventBus is safe for concurrent use.
type EventBus struct { ... }
```

### Func/Method

```go
// Register adds a handler for the given event type. Handlers are keyed by
// EventType() and deduplicated by function pointer.
//
// Returns an error if the event type is empty or the handler is nil.
func (b *EventBus) Register(eventType string, h Handler) error { ... }
```

### Regra

- TODOS os exports doc-commented
- Imperativo presente: "Register adds...", "Emit dispatches..."
- Primeira frase termina com ponto, forma substantiva do nome
- Não repetir o nome do símbolo desnecessariamente

---

## 8. Arquivos que você NÃO deve tocar

Você está gerando arquivos NOVOS em `/tmp/crewai-port-output/pkg/**`. Não modifique, edite, ou assuma que pode reescrever arquivos existentes do PM-OS — você não tem acesso a eles. Se um arquivo portado precisar interagir com código PM-OS existente, assuma a API pelos exemplos em `PM-OS-CONTEXT.md` e deixe TODO no código + nota em `PORT-REPORT.md`.

Exemplo:
```go
// TODO(akita): wire to existing pkg/engine/tools/registry.go GlobalRegistry
// after merge. This package currently uses local registry for tests.
```

---

## 9. Output format

### Ordem dentro de cada arquivo .go

1. Package doc comment
2. `package <name>`
3. Imports (stdlib, then external, blank line separated)
4. Package-level constants
5. Package-level variables (incl sentinel errors)
6. Exported types (ordered: interfaces → structs → enums)
7. Unexported types
8. Constructor functions (`New*`)
9. Methods on types (receiver-grouped)
10. Package-level functions
11. Unexported helpers

### Exemplo

```go
// Package fingerprint provides deterministic and random unique identifiers
// for PM-OS recipes and work items.
//
// Ported from lib/crewai/src/crewai/security/fingerprint.py.
package fingerprint

import (
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// MaxMetadataSize limits metadata payload to prevent DoS.
const MaxMetadataSize = 10_000

// ErrInvalidMetadata is returned when metadata fails validation.
var ErrInvalidMetadata = errors.New("invalid metadata")

// Fingerprint uniquely identifies a PM-OS entity across runs.
type Fingerprint struct {
    UUIDStr   string         `json:"uuid_str"`
    CreatedAt time.Time      `json:"created_at"`
    Metadata  map[string]any `json:"metadata,omitempty"`
}

// New creates a random Fingerprint with the given metadata.
func New(metadata map[string]any) (*Fingerprint, error) {
    if err := validateMetadata(metadata); err != nil {
        return nil, err
    }
    return &Fingerprint{
        UUIDStr:   uuid.New().String(),
        CreatedAt: time.Now(),
        Metadata:  metadata,
    }, nil
}

// GenerateFromSeed creates a deterministic Fingerprint from a seed string.
// Same seed always produces the same UUID. Uses UUID5 with PMOSNamespace.
func GenerateFromSeed(seed string, metadata map[string]any) (*Fingerprint, error) { ... }

// Validate returns an error if the fingerprint is malformed.
func (f *Fingerprint) Validate() error { ... }

func validateMetadata(m map[string]any) error { ... }
```

---

## 10. Regras finais

- **Se a tradução exigir decisão arquitetural**, pare e anote em `PORT-REPORT.md`.
- **Se o arquivo Python usa lib Python sem equivalente Go**, liste a lib + solução sugerida em `PORT-REPORT.md`.
- **Se a tradução quebra semântica**, documente em comment + `PORT-REPORT.md`.
- **Nunca invente funcionalidade** que não está no source Python.
- **Nunca "melhore"** a arquitetura. Traduza. Akita refatora depois.
- **Commit message padrão:** `port(<pkg>): translate <file>.py to Go` + corpo com deviations.
