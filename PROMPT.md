# PROMPT MESTRE — OpenClaw (TS) → PM-OS (Go)

**Público-alvo:** LLM especializada em tradução cross-language (Claude Sonnet 4.6+, GPT-5+, Cursor IDE).

**Objetivo:** Implementar 30+ items selecionados do [openclaw/openclaw](https://github.com/openclaw/openclaw) como Go idiomático em PM-OS. **NÃO é port literal.** É **adoção seletiva** classificada em 4 tiers, cada um com método de adoção diferente.

---

## ANTES DE COMEÇAR — leia 5 anexos nesta ordem

1. `PM-OS-CONTEXT.md` — arquitetura alvo. **OBRIGATÓRIO.**
2. `RULES.md` — convenções Go PM-OS + restrições hard.
3. `EXTRACTION-PLAN.md` — análise-fonte com conceitos detalhados + snippets de referência.
4. `ITEM-MAP.md` — lista dos 30+ items, cada um com origem OpenClaw + método tier + target PM-OS.
5. `CHECKLIST.md` — critérios aceitação por tier.

**Se qualquer anexo faltar, pare e avise.**

---

## Setup inicial

```bash
mkdir -p /tmp/openclaw-port-output
cd /tmp/openclaw-port-output
go mod init openclaw-port
git clone --depth 1 https://github.com/openclaw/openclaw.git /tmp/openclaw-source
```

Esta pasta `/tmp/openclaw-port-output/` é **seu output**. PM-OS real vive em outro lugar — você NÃO toca ele. Você produz Go code aqui, Akita copia depois.

---

## Tarefa — por tier

### Tier A — Copy-Paste Direto (8 items)

**Método:** Schemas YAML/JSON, snippets bash, funções puras Go. Pega código no openclaw source, traduz quase literal, coloca em PM-OS style.

**Exemplo (Tier A #1 — Benchmark Percentile Stats):**
- Origem: `openclaw/scripts/bench-cli-startup.ts`
- Target: `/tmp/openclaw-port-output/tools/pm-bench/stats.go`
- Tempo: 30 min
- Código pronto já em `EXTRACTION-PLAN.md` seção 1.1 — apenas adapte package name + escreve test

**Gates:** compila, test happy+edge, doc comment, no external deps.

### Tier B — Port-and-Adapt (8 items)

**Método:** Algoritmo TS conhecido vira Go idiomático. Você LÊ o TS no openclaw source pra entender, depois REESCREVE em Go sem copy.

**Exemplo (Tier B #1 — Channel Inbound Router):**
- Origem: `openclaw/src/channels/` e `openclaw/extensions/telegram/`
- Target: `/tmp/openclaw-port-output/pkg/channels/router.go` + `router_test.go`
- Tempo: 3-5 dias
- Ver `EXTRACTION-PLAN.md` seção 2.1 pra estrutura esperada

**Gates:** compila, testify test suite, `go test -race`, cobertura ≥70%, error handling `%w`, context.Context como primeiro arg.

### Tier C — Capabilities JSON (7 items)

**Método:** Adiciona campo novo em schema recipe PM-OS. Cada item é: JSON schema patch + Go struct field + validator.

**Exemplo (Tier C #1 — fallback_models):**
- Target: `/tmp/openclaw-port-output/pkg/recipe/schema_extension.go` + tests
- Adiciona campo `FallbackModels []string` em struct Wave
- Validator rejeita modelos desconhecidos
- Ver `EXTRACTION-PLAN.md` seção 3.1

**Gates:** valida JSON recipe real, schema compatibility, docs inline.

### Tier D — CLI / Tools / Tests / Skills (12 items)

**Método:** Features novas inspiradas no openclaw mas desenhadas pra PM-OS. Redesign, não port.

**Exemplo (Tier D #1 — pm-cli doctor):**
- Target: `/tmp/openclaw-port-output/cmd/pm-cli/doctor.go` + tests
- Inspira em `openclaw/src/cli/` + `openclaw/src/commands/`
- Implementa diagnostic checks + guided fixes (config, systemd, supabase conn)

**Gates:** runs end-to-end, friendly UX, flags documentadas, dry-run mode.

---

## Método TDD obrigatório

Cada item:

1. **RED:** escreve teste que falha
2. **GREEN:** implementação mínima passa
3. **REFACTOR:** limpa
4. Teste cobre happy path + edge cases + error paths + concorrência (se state shared)

Use `github.com/stretchr/testify` (já no PM-OS go.mod, permitido).

---

## Regras hard (detalhadas em `RULES.md`)

- Go 1.25+ stdlib-first
- Sem Gin, sem ORMs, sem asyncio/pydantic equivalents
- Sem panic em library code
- `context.Context` primeiro arg em I/O
- Error wrap com `fmt.Errorf("x: %w", err)`
- Naming: PascalCase exportado, camelCase unexportado, snake_case apenas em JSON tags (espelha Supabase colunas)
- Testes: `*_test.go` mesmo package, `t.Parallel()` onde possível, zero `time.Sleep` em testes
- Doc comments em TODOS exports

---

## Output esperado

Estrutura final em `/tmp/openclaw-port-output/`:

```
/tmp/openclaw-port-output/
├── PORT-REPORT.md              # resumo por item + deviations + TODOs
├── AUDIT-REPORT.md             # self-audit (gerado depois via SELF-AUDIT-PROMPT.md)
├── go.mod + go.sum
├── pkg/
│   ├── channels/
│   │   ├── router.go           # Tier B #1
│   │   ├── types.go            # struct Message, Channel interface
│   │   ├── telegram.go         # Tier B #2
│   │   ├── telegram_test.go
│   │   └── ...
│   ├── recipe/
│   │   └── schema_extension.go # Tier C itens 1-5
│   ├── memory/
│   │   ├── recall_score.go     # Tier A #8 + Tier B #4
│   │   ├── reconciler.go       # Tier B #7
│   │   └── ...
│   ├── mcp/
│   │   ├── name_registry.go    # Tier A #4 + snippet #2
│   │   ├── loopback_gateway.go # Tier B #5
│   │   ├── deny_list.go        # Tier A #6
│   │   └── ...
│   ├── hooks/
│   │   └── event_bus.go        # Tier B #3
│   ├── testing/
│   │   ├── fixtures.go         # Tier D #6.1
│   │   └── e2e_harness.go      # Tier D #6.2
│   └── ...
├── cmd/
│   └── pm-cli/
│       ├── doctor.go           # Tier D #5.1
│       ├── wizard.go           # Tier D #5.2
│       ├── bench.go            # Tier D #5.3
│       └── channels.go         # Tier D #5.4
├── tools/
│   ├── pm-bench/
│   │   ├── startup.go          # Tier A #1 + Tier D #4.2
│   │   └── stats.go
│   └── pm-audit/
│       └── cycle.go            # Tier D #4.4
├── scripts/
│   ├── ci-gates.sh             # Tier D #4.1
│   ├── prepush-ci.sh           # Tier D #4.5
│   └── e2e/
│       ├── docker-build.sh     # Tier A #2
│       └── docker-compose.e2e.yml # Tier D #4.3
├── recipes/
│   └── qa-scenarios/           # Tier D #6.3
│       ├── README.md
│       └── sample-scenario.md  # template
├── docs/
│   └── templates/
│       ├── skill-frontmatter.yaml  # Tier D #7.2
│       ├── doc-frontmatter.md      # Tier D #8.1
│       └── glossary-pt-br.json     # Tier D #8.2
└── migrations/
    └── (só se algum item exigir schema Supabase novo)
```

Cada arquivo `.go` começa com:
- Package doc comment
- Comment "Adapted from openclaw/`<origem>`" com link ao arquivo fonte
- Doc comments em todos exports

---

## PORT-REPORT.md — obrigatório

Conteúdo mínimo:

```markdown
# Port Report — OpenClaw → PM-OS

## Summary

- Items completed: X / 30
- LOC produced: ~Y (medir com `find . -name '*.go' | xargs wc -l`)
- Average test coverage: Z%
- External deps added beyond PM-OS standard: list or "none"

## Per-Tier Status

### Tier A — Copy-Paste (target 8 items)
- [x] A1 bench_stats (40 LOC, 3 tests, 100% coverage)
- [x] A2 docker-build.sh (bash)
- [x] A3 artifact_validation.go (...)
...

### Tier B — Port-and-Adapt (target 8 items)
- [x] B1 channels/router.go (300 LOC, 12 tests, 85% coverage)
...

### Tier C — Capabilities JSON (target 7 items)
...

### Tier D — CLI / Tools / Tests / Skills (target 12 items)
...

## Deviations

Lista items que desviaram da spec com motivo.

## TODOs para Akita

### High priority
- [ ] Wire channels/router.go no engine.RunRecipe
- [ ] Register pm-cli doctor em cobra root
- ...

### Medium priority
- ...

### Questions for Akita
- Should pm-cli doctor also check Caddy config?
- Memory recall_score — use cosine distance from pgvector OR in-Go computation?

## External deps

Tabela: dep | version | reason | which file uses it

## Test results

```
go test -race -cover ./...
```

<paste last 30 lines of output>

## Known issues

Lista items onde OpenClaw semantics não mapeou limpamente.
```

---

## Como isso complementa Akita

Eu (Akita) vou:

1. Ler seu `PORT-REPORT.md` primeiro
2. Rodar `SELF-AUDIT-PROMPT.md` resultado
3. Copiar item por item pro branch `feat/openclaw-adoption` do PM-OS
4. Resolver conflitos com packages existentes (pkg/channels? talvez já tenha; pkg/memory existe; pkg/mcp existe)
5. Integrar capabilities JSON no validator + compiler
6. Registrar CLI novos em cobra root
7. Rodar `go test -race -cover ./...` full pm-os, fixar novos breaks
8. Passar pelo Keel
9. Commit atômico por item
10. Merge em `main` após QA

Se você tentar integrar sozinho, vai conflitar com pkg/ existente PM-OS. **Seu escopo é só produzir Go código idiomático com tests, em `/tmp/openclaw-port-output/`.**

---

## Fail fast — quando parar

- Dependência OpenClaw não mapeável trivialmente → pare, anote em PORT-REPORT
- Conceito requer arquitetura PM-OS que você não conhece → pare, pergunte
- Múltiplas implementações possíveis Go → liste opções, peça guidance
- Test exige mock de sistema externo → implementa mock interno em `_test.go`, nunca subprocess

---

## Aviso final

Este é port **ADAPTATIVO + IDIOMÁTICO + SELETIVO**.

- NÃO copie código TS palavra-por-palavra — traduza idiom Go
- NÃO porte 3000 LOC TS como 3000 LOC Go — SIMPLIFIQUE para o essencial
- NÃO invente features que não estão no `ITEM-MAP.md`
- NÃO edite PM-OS real (`/home/agdis/pm-os/`) — produz em `/tmp/openclaw-port-output/`

Comece pelo Tier A (items pequenos, baixo risco). Progride pra B, C, D conforme `ITEM-MAP.md` ordem.

Commit messages padrão por item:
```
port(openclaw): add <item-name>

- Tier: A|B|C|D
- Source: openclaw/<path> (XXX LOC TS)
- Target: <path>/<name>.go (XXX LOC Go)
- Tests: X tests, Y% coverage
- Deviations: list or "none"
```
