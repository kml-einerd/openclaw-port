# CHECKLIST — Critérios de Aceitação por Tier

**Cada item entregue DEVE passar em todos os gates do seu tier antes de ser marcado completo em PORT-REPORT.md.**

---

## Gates Universais (TODOS os tiers)

- [ ] `go build ./...` compila sem erro (no output)
- [ ] `go vet ./...` zero warnings
- [ ] Package doc comment presente
- [ ] Exports com doc comments
- [ ] Comment explicando origem openclaw (`// Adapted from openclaw/<path>`)
- [ ] Zero dependências externas não-listadas em RULES.md
- [ ] Zero `panic()` em library code (exceto main)
- [ ] Error handling com `fmt.Errorf("x: %w", err)`
- [ ] Context.Context primeiro arg em I/O

---

## Tier A — Copy-Paste Direto

Additional gates:

- [ ] Test happy path cobre função pura
- [ ] Test edge case (nil input, empty, boundary)
- [ ] Coverage ≥ 90% (função pura é fácil)
- [ ] Sem shared state mutável (funções stateless)
- [ ] Sem I/O em função core (I/O em wrapper separado se precisar)

Timebox cada item: **30min-1h**. Se passar disso, parou pra escopo crescer.

---

## Tier B — Port-and-Adapt

Additional gates:

- [ ] Testify suite com ≥ 5 test cases
- [ ] `go test -race` passa (concorrência real se item usa shared state)
- [ ] Coverage ≥ 70%
- [ ] `context.Context` respeitado (cancelamento propaga)
- [ ] Error types explícitos onde necessário (não apenas fmt.Errorf genérico)
- [ ] Interface mínima exposta (implementação unexported quando possível)
- [ ] Sem assumption de global state não-documentado
- [ ] Mocking em tests via interface, não subprocess/rede real

Timebox cada item: **1-3 dias**. Se passar disso, escopo cresceu ou decisão arquitetural — pare e reporte.

---

## Tier C — Capabilities JSON

Additional gates:

- [ ] Struct com tag `json:"snake_case"` matching recipe schema
- [ ] Field opcional — omit sem quebrar recipes existentes (backwards compatible)
- [ ] Validator function que roda em recipe load time
- [ ] Validator retorna `*recipe.ValidationError` ou similar typed error
- [ ] Test: recipe JSON válido passa, inválido falha com msg clara
- [ ] Test: recipe SEM o campo (omit) passa
- [ ] Documentação inline do campo (formato `JSON: {"field": "example"}`)
- [ ] Se capability interage com engine, stub de wiring (comment TODO para Akita)

Timebox cada item: **4h-2 dias**.

---

## Tier D — CLI / Tools / Tests / Skills

Depende do subtipo:

### D1 Scripts Tooling

- [ ] Shell script idempotente (rodar 2x = mesmo resultado)
- [ ] `set -euo pipefail` em bash
- [ ] `--help` flag em Go tools
- [ ] Go tools usam cobra se CLI, stdlib flag se trivial
- [ ] Coverage ≥ 60% (tool Go)
- [ ] Bash scripts testados com `shellcheck` se disponível

### D2 CLI Commands

- [ ] Registra em cobra via `func init()` no arquivo
- [ ] Flags documentadas (`--help` output claro)
- [ ] `--dry-run` suportado onde aplica
- [ ] Erros amigáveis (sem stack trace pro usuário)
- [ ] Test: mock Supabase/systemd calls, verifica saída esperada
- [ ] Coverage ≥ 60%

### D3 Testing Utilities

- [ ] Pode ser importado por outros pacotes (sem init globais)
- [ ] Factory functions têm opções functional style (`WithX(v)`)
- [ ] Self-tested (fixtures.go testa próprio factory)
- [ ] Sem dependência de filesystem real (temp dirs ok)
- [ ] Example usage em test ou godoc

### D4 Skill/Doc Patterns

- [ ] Template markdown válido (frontmatter YAML passa parse)
- [ ] README explica propósito + exemplo
- [ ] Glossário JSON com chave=valor válido
- [ ] FAQ template tem 3+ exemplos concretos

Timebox cada item Tier D: **1-4 dias**.

---

## PORT-REPORT.md — template obrigatório

Arquivo `/tmp/openclaw-port-output/PORT-REPORT.md` deve conter:

```markdown
# Port Report — OpenClaw → PM-OS

## Summary

- Items completed: X / ~35
- LOC Go produced: ~Y
- LOC bash/yaml/json: ~Z
- Average test coverage: W%
- External deps added beyond standard: lista ou "none"

## Per-Tier Status

### Tier A — Copy-Paste (8 items)
- [x] A1 bench_stats (40 LOC, 3 tests, 100% cov)
- [x] A2 docker-build.sh
- ...

### Tier B — Port-and-Adapt (8 items)
- [x] B1 channels/router.go (320 LOC, 15 tests, 82% cov)
- ...

### Tier C — Capabilities JSON (7 items)
- [x] C1 wave_fallback.go (85 LOC, 4 tests, 90% cov)
- ...

### Tier D — CLI / Tools / Tests / Skills (12 items)
- [x] D1.1 ci-gates.sh (85 LOC bash)
- ...

## Deviations

Per-item desvios do spec, com motivo.

### Example
#### B1 channels/router.go
- Expected webhook auto-routing by content-type; instead routes only by channel prefix path (`/webhooks/telegram/*`). TS version used complex middleware chain; Go version simpler.
- Akita: decide se quer content-type routing, trivial adicionar.

## TODOs para Akita

### High priority (blocking integration)
- [ ] Wire `pkg/channels/router.go` em `cmd/pm-api/main.go`
- [ ] Register pm-cli commands no cobra root
- [ ] Migration SQL para `mcp_cache` table (B5)

### Medium priority
- [ ] Consolidar `pkg/hooks/event_bus.go` (B3) com `pkg/events` existente
- [ ] Memory reconciler (B7) precisa cron registration

### Questions for Akita
- Should pm-cli doctor check Caddy config?
- Channel router: path-based vs subdomain-based webhooks?
- Integration com existente pkg/memory/scope.go — merge ou coexist?

## External dependencies added

| Dep | Version | Reason | File |
|---|---|---|---|
| github.com/charmbracelet/huh | vX | wizard prompts | cmd/pm-cli/wizard.go |

## Test execution

```
cd /tmp/openclaw-port-output
go test -race -cover ./...
```

<paste tail -30 output>

## Known issues

- D3.3 provider-replay: requires LLM_REPLAY_MODE env var, otherwise passes through
- B8 E2E harness: needs port 0 for ephemeral, bind fails on busy CI
```

---

## Regra de ouro

**Qualquer gate falhar = item não está pronto. Marcar em andamento com [ ] até passar.**

Se encontrar obstáculo arquitetural, não tente decidir sozinho — reporte em "Questions for Akita" e pule pro próximo item.
