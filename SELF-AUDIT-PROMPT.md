# SELF-AUDIT — Verificação de Completude OpenClaw Port

Você terminou o port. Antes de entregar, **rode este audit contra seu próprio output em `/tmp/openclaw-port-output/`**.

---

## PARTE 1 — Inventário de Items

Para CADA item listado em `ITEM-MAP.md`, responda:

```
[TIER A]
A1 bench_stats                      → STATUS | LOC | TEST_FILE | TEST_PASS | COVERAGE%
A2 docker-build.sh                  → STATUS | LOC | N/A (bash) | N/A | N/A
A3 artifact_validation              → STATUS
A4 mcp sanitize                     → STATUS
A5 skill frontmatter schema         → STATUS
A6 dangerous tools deny list        → STATUS
A7 job timing summary               → STATUS
A8 recency decay                    → STATUS

[TIER B]
B1 channels/router.go               → STATUS | LOC | tests | pass | cov
B2 channels/telegram.go             → STATUS
B3 hooks/event_bus.go               → STATUS
B4 memory/recall_score.go           → STATUS
B5 mcp/loopback_gateway.go          → STATUS
B6 recipe/fallback.go               → STATUS
B7 memory/reconciler.go             → STATUS
B8 testing/e2e_harness.go           → STATUS

[TIER C]
C1-C7                               → STATUS per item

[TIER D]
D1.1-D1.5 scripts                   → STATUS per item
D2.1-D2.4 CLI commands              → STATUS per item
D3.1-D3.3 testing utilities         → STATUS per item
D4.1-D4.5 skill/doc patterns        → STATUS per item
```

STATUS = `DONE` | `PARTIAL` | `SKIPPED` | `MISSING`

Para qualquer item não-DONE, explique em 1 linha.

---

## PARTE 2 — Rodar gates técnicos

Execute em `/tmp/openclaw-port-output/`:

```bash
# Gate 1 — compila
go mod tidy
go build ./... 2>&1 | tee build.log

# Gate 2 — vet
go vet ./... 2>&1 | tee vet.log

# Gate 3 — testa
go test ./... -count=1 2>&1 | tee test.log

# Gate 4 — race
go test -race ./... -count=1 -timeout 10m 2>&1 | tee race.log

# Gate 5 — cover
go test -cover ./... -coverprofile=coverage.out 2>&1 | tee cover.log
go tool cover -func=coverage.out | tail -10

# Gate 6 — shell scripts
find scripts -name "*.sh" -exec bash -n {} \; 2>&1 | tee shell-syntax.log
# Se shellcheck disponível:
find scripts -name "*.sh" -exec shellcheck {} \; 2>&1 | tee shellcheck.log || true

# Gate 7 — YAML/JSON válidos
find . -name "*.yaml" -o -name "*.yml" | xargs -I{} python3 -c "import yaml; yaml.safe_load(open('{}'))" 2>&1 | tee yaml-check.log || true
find . -name "*.json" | xargs -I{} python3 -c "import json; json.load(open('{}'))" 2>&1 | tee json-check.log || true
```

Cole output RESUMIDO no AUDIT-REPORT.md (erros + totais).

---

## PARTE 3 — Validar regras hard (RULES.md)

SIM/NÃO com evidência:

1. **Sem pydantic/decorators Python/asyncio?** `grep -rn "pydantic\|@\w\+(\|async def\|asyncio" pkg/ --include="*.go"` → 0 matches (exceto em strings/comments)
2. **Sem interface{} preguiçoso?** count `grep -rn "interface{}" pkg/` (aceitável APENAS em JSON boundaries)
3. **Context.Context primeiro param em I/O?** sample 5 funções I/O, confirme
4. **Doc comments em exports?** `grep -rn "^func [A-Z]\|^type [A-Z]" pkg/ | wc -l` vs `grep -rn "^//" pkg/ | wc -l` — razão deve ser ~1:1
5. **Deps permitidas?** listar `go list -m all | grep -v indirect` e compare com RULES.md seção 1
6. **Error wrapping %w?** `grep -rn "fmt.Errorf" pkg/ | grep -v "%w" | grep -v "_test.go" | wc -l` → 0
7. **snake_case.go files?** `find . -name "*-*.go" | wc -l` → 0
8. **Package doc comments?** `for pkg in $(find pkg -type d -not -path './pkg'); do grep -l "^// Package " $pkg/*.go 2>/dev/null | head -1 || echo "MISSING: $pkg"; done` → 0 MISSING
9. **Zero panic em library?** `grep -rn "panic(" pkg/ --include="*.go" | grep -v "_test.go" | wc -l` → ~0
10. **JSON tags snake_case?** sample 5 structs com tags, confirme convention

---

## PARTE 4 — Validar gates por Tier

### Tier A (8 items)
- [ ] Cada item: cobertura ≥ 90% (funções puras)
- [ ] Cada item: sem I/O no core
- [ ] Bash scripts passaram syntax check

### Tier B (8 items)
- [ ] Cada item: coverage ≥ 70%
- [ ] Cada item: test -race passa
- [ ] Cada item: testify suite ≥ 5 tests
- [ ] Interfaces mínimas expostas
- [ ] Mocking via interface (não subprocess)

### Tier C (7 items)
- [ ] Cada capability: struct com JSON tag snake_case
- [ ] Cada capability: field opcional (omit ok)
- [ ] Cada capability: validator retorna typed error
- [ ] Cada capability: test com recipe JSON válido + inválido + omit
- [ ] Documentação inline com exemplo JSON

### Tier D (12 items)
- [ ] Scripts idempotentes (rodar 2x = mesmo resultado)
- [ ] CLI commands têm `--help`
- [ ] CLI commands têm `--dry-run` onde aplica
- [ ] Testing utilities self-tested
- [ ] Skill/doc templates têm README explicando

---

## PARTE 5 — Leak checks

```bash
# Items paradigma errado?
grep -rn "AgentConfig\|CrewConfig\|Flow\b" pkg/ --include="*.go" | wc -l    # ~0

# Decorator attempts (TS-ism)?
grep -rn "// @decorator\|reflect.StructTag.*Func" pkg/ | wc -l  # ~0

# Globals mutáveis não-documentados?
grep -rn "^var [a-z]" pkg/ --include="*.go" | grep -v "_test.go" | grep -v "// " | wc -l  # review se > 5

# Imports de openclaw source acidentais?
grep -rn "openclaw" --include="*.go" | grep -v "Adapted from" | wc -l  # 0

# Node.js runtime refs?
grep -rn "require(\|process\.env" --include="*.go" | wc -l  # 0
```

---

## PARTE 6 — Validação cruzada com snippets EXTRACTION-PLAN

Para cada snippet numerado em EXTRACTION-PLAN §9 (Snippets #1-10), confirme:

1. RecallScore Composite → existe em pkg/memory/recall_score.go? API bate com snippet? Tests cobrem?
2. MCP NameRegistry → existe em pkg/mcp/sanitize.go ou similar? Namespacing `server__tool-N` implementado?
3. Hook Event Bus → pkg/hooks/event_bus.go existe? Subscribe/Emit funcionam?
4. Episode Eviction Reconciler → pkg/memory/reconciler.go? Roda em goroutine com ticker?
5. Benchmark Stats → tools/pm-bench/stats.go? Função `CalculatePercentiles()` matching signature?
6. InboundRouter → pkg/channels/router.go? Interface `ChannelHandler` exposta?
7. Wave Fallback → pkg/recipe/fallback.go ou wave_fallback.go? Struct Wave estendida?
8. Docker Sandbox → config struct em pkg/recipe/sandbox.go + script em scripts/e2e/?
9. Recipe `requires` Pre-flight → pkg/recipe/requires.go? RunPreflight() func existe?
10. Scope-Aware Memory → pkg/memory/scope_retrieval.go (extensão)? Preferred collection hint funciona?

Report ok/faltando cada um.

---

## PARTE 7 — Output final

Gerar `/tmp/openclaw-port-output/AUDIT-REPORT.md`:

```markdown
# Audit Report — OpenClaw → PM-OS Port

## Summary
- Items DONE: X / ~35
- Items PARTIAL: Y
- Items SKIPPED: Z
- Items MISSING: W
- Build: PASS|FAIL
- Test pass: X/Y packages
- Coverage avg: W%
- Race detector: PASS|FAIL

## Part 1 — File inventory
<tabela completa>

## Part 2 — Technical gates
<resumo 7 comandos>

## Part 3 — Hard rules
<10 SIM/NÃO + evidence>

## Part 4 — Per-tier gates
<checkboxes + test paths>

## Part 5 — Leak check
<counts 5 greps>

## Part 6 — Snippets cross-check
<10 itens ok/falta>

## Known gaps
<TODOs pro Akita>

## Confidence score
<0-100 + justificativa>
```

---

## Regra de ouro

**Seja brutalmente honesto.** Akita vai rodar os mesmos comandos — se você mentir, vai pegar. Melhor reportar `PARTIAL` claro que vender "DONE" falso.

**Se gate crítico falhar:** reporte em "Known gaps" e entregue mesmo assim. Akita decide se manda de volta ou conserta.

**Se passou tudo:** declare confidence 90+ com justificativa.

Output final: **1 arquivo `/tmp/openclaw-port-output/AUDIT-REPORT.md`** + confirmação explícita dos gates executados.
