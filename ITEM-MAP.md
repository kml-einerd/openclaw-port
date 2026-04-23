# ITEM-MAP — OpenClaw → PM-OS por Tier

**Base extraction:** `EXTRACTION-PLAN.md` tem código pronto de referência para cada item. Consulte-o.

**Base openclaw source:** `/tmp/openclaw-source/` (clone você mesmo após ler PROMPT.md).

**Base target output:** `/tmp/openclaw-port-output/` (sua pasta de trabalho).

**Ordem recomendada:** Tier A → C → B → D. Motivo:
- A: snippets pequenos, baixo risco, aquecimento (~1 dia total)
- C: capabilities JSON, afeta só schema, isoladas (~3 dias)
- B: algoritmos reais com dependências entre si (~1-2 sem)
- D: CLI + tools + tests, consome o que B criou (~2-3 sem)

---

## Tier A — Copy-Paste Direto (8 items)

**Método:** Código openclaw traduzido quase literal para Go. Código pronto na `EXTRACTION-PLAN.md`.

| # | Nome | Origem (openclaw) | Target (port-output) | LOC | Tempo | Ref PLAN |
|---|---|---|---|---|---|---|
| A1 | Benchmark Percentile Stats | `scripts/bench-cli-startup.ts` | `tools/pm-bench/stats.go` | ~40 | 30min | §1.1 |
| A2 | Docker Build/Reuse Shell | `scripts/lib/docker-e2e-image.sh` | `scripts/e2e/docker-build.sh` | ~25 | 20min | §1.2 |
| A3 | Artifact Validation Helpers | `scripts/pr-lib/common.sh` | `scripts/pr/artifacts.sh` | ~50 | 45min | §1.3 |
| A4 | MCP Tool Name Sanitization | `src/mcp/` references | `pkg/mcp/sanitize.go` | ~30 | 30min | §1.4 |
| A5 | Skill YAML Frontmatter Schema | `skills/*/SKILL.md` examples | `pkg/skills/frontmatter_schema.go` + template | ~60 | 1h | §1.5 |
| A6 | Dangerous Tools Deny List | `src/mcp/dangerous-tools.ts` | `pkg/mcp/deny_list.go` | ~40 | 45min | §1.6 |
| A7 | Job Timing Summary | `scripts/ci-run-timings.mjs` | `tools/pm-audit/timing.go` | ~70 | 1h | §1.7 |
| A8 | Recency Half-Life Decay | `extensions/memory-core/` | `pkg/memory/recency.go` | ~25 | 30min | §1.8 |

**Total Tier A: ~340 LOC, ~5h**

---

## Tier B — Port-and-Adapt (8 items)

**Método:** Algoritmo TS conhecido, reescrito Go idiomático. Você LÊ o TS pra entender, depois REIMPLEMENTA.

| # | Nome | Origem (openclaw) | Target (port-output) | LOC | Tempo | Ref PLAN |
|---|---|---|---|---|---|---|
| B1 | Channel Inbound Router | `src/channels/` + `extensions/telegram/` | `pkg/channels/router.go` + `types.go` | ~300 | 2-3 dias | §2.1 |
| B2 | Telegram Webhook Handler | `extensions/telegram/inbound.ts` | `pkg/channels/telegram.go` | ~200 | 1-2 dias | §2.2 |
| B3 | Global Hook Event Bus | `src/hooks/` + global state | `pkg/hooks/event_bus.go` | ~120 | 1 dia | §2.3 |
| B4 | Memory Multi-Component Scorer | `extensions/memory-core/scorer.ts` | `pkg/memory/recall_score.go` | ~200 | 2 dias | §2.4, snippet #1 |
| B5 | MCP Loopback Gateway | `src/mcp/loopback/` | `pkg/mcp/loopback_gateway.go` | ~250 | 2-3 dias | §2.5 |
| B6 | Wave-Level Model Fallback | implícito em provider fallback | `pkg/recipe/fallback.go` + engine hook | ~150 | 2 dias | §2.6, snippet #7 |
| B7 | Episode Auto-Eviction Reconciler | `extensions/memory-core/retention.ts` | `pkg/memory/reconciler.go` | ~180 | 2 dias | §2.7, snippet #4 |
| B8 | E2E Harness pm-api + pm-engine | `src/e2e/gateway-harness.ts` | `pkg/testing/e2e_harness.go` | ~250 | 3 dias | §2.8, §6.2 |

**Total Tier B: ~1650 LOC, ~12-17 dias**

**Dependências internas:**
- B8 usa B3 (event bus)
- B4 + B7 compartilham types em pkg/memory/

---

## Tier C — Capabilities JSON (7 items)

**Método:** Adicionar campo novo em schema `recipe.Recipe` ou `recipe.Step` ou `recipe.Wave`. Struct field + JSON tag + validator + test.

**Importante:** você NÃO edita pm-os real. Você produz arquivo `pkg/recipe/schema_extension.go` que o Akita mergerá.

| # | Nome | Target (port-output) | LOC | Tempo | Ref PLAN |
|---|---|---|---|---|---|
| C1 | `fallback_models` na Wave | `pkg/recipe/wave_fallback.go` + test | ~80 | 1 dia | §3.1 |
| C2 | `requires` na Recipe (bins/env/config) | `pkg/recipe/requires.go` + preflight validator | ~120 | 1-2 dias | §3.2, snippet #9 |
| C3 | `trigger_phrases` na Recipe description | `pkg/recipe/trigger_phrases.go` (Raven hint) | ~60 | 1 dia | §3.3 |
| C4 | `retention_policy` na Quality config | `pkg/recipe/retention.go` | ~80 | 1 dia | §3.4 |
| C5 | `collection_hint` no Step | `pkg/recipe/collection_hint.go` | ~50 | 4h | §3.5 |
| C6 | `requires_tools` no Skill frontmatter | `pkg/skills/requires.go` + parser | ~100 | 1 dia | §3.6 |
| C7 | `webhook_trigger` no Sensor | `pkg/sensors/webhook_trigger.go` | ~150 | 2 dias | §3.7 |

**Total Tier C: ~640 LOC, ~8-10 dias**

---

## Tier D — CLI / Tools / Tests / Skills (12 items)

**Método:** Features novas redesenhadas para PM-OS. Inspira em openclaw mas implementa do zero.

### D1 — Scripts de Tooling (5)

| # | Nome | Target | LOC | Tempo | Ref PLAN |
|---|---|---|---|---|---|
| D1.1 | `scripts/ci-gates.sh` | `scripts/ci-gates.sh` | ~80 bash | 2h | §4.1 |
| D1.2 | `tools/pm-bench/startup.go` | `tools/pm-bench/startup.go` + cmd | ~200 | 1 dia | §4.2 |
| D1.3 | `docker-compose.e2e.yml` | `scripts/e2e/docker-compose.e2e.yml` | ~60 yaml | 2h | §4.3 |
| D1.4 | `tools/pm-audit/cycle.go` (import cycle) | `tools/pm-audit/cycle.go` | ~150 | 1 dia | §4.4 |
| D1.5 | `scripts/prepush-ci.sh` | `scripts/prepush-ci.sh` | ~40 bash | 1h | §4.5 |

### D2 — CLI Commands (4)

Assume `cmd/pm-cli/` com cobra. Cada comando é arquivo + test.

| # | Nome | Target | LOC | Tempo | Ref PLAN |
|---|---|---|---|---|---|
| D2.1 | `pm-cli doctor` | `cmd/pm-cli/doctor.go` + tests | ~300 | 3 dias | §5.1 |
| D2.2 | `pm-cli init --wizard` | `cmd/pm-cli/wizard.go` + tests | ~400 | 3-4 dias | §5.2 |
| D2.3 | `pm-cli bench` | `cmd/pm-cli/bench.go` + tests | ~150 | 1 dia | §5.3 |
| D2.4 | `pm-cli channels setup` | `cmd/pm-cli/channels.go` + tests | ~200 | 2 dias | §5.4 |

### D3 — Testing Utilities (3)

| # | Nome | Target | LOC | Tempo | Ref PLAN |
|---|---|---|---|---|---|
| D3.1 | Mock Factory Kit | `pkg/testing/fixtures.go` | ~250 | 2 dias | §6.1 |
| D3.2 | Scenario Pack Registry | `recipes/qa-scenarios/*.md` + runner | ~200 Go + templates | 2 dias | §6.3 |
| D3.3 | Provider Replay (deterministic LLM) | `tools/pm-test/provider-replay.go` | ~150 | 1-2 dias | §6.4 |

### D4 — Skill/Doc Patterns (3+)

| # | Nome | Target | LOC | Tempo | Ref PLAN |
|---|---|---|---|---|---|
| D4.1 | Skill dir structure template | `docs/templates/skill-template/` + README | ~50 md | 2h | §7.1, §7.2, §7.3 |
| D4.2 | Raven Integration Hook (skill match) | `pkg/planner/skill_matcher.go` | ~120 | 1 dia | §7.4 |
| D4.3 | Doc frontmatter pattern | `docs/templates/doc-frontmatter.md` | ~20 md | 30min | §8.1 |
| D4.4 | Glossary PT-BR | `docs/templates/glossary-pt-br.json` | ~100 json | 2h | §8.2 |
| D4.5 | FAQ "First 60 Seconds" pattern | `docs/templates/first-60s.md` | ~30 md | 1h | §8.3 |

**Total Tier D: ~2200 LOC + 2000 LOC docs, ~25-30 dias**

---

## Sumário geral

| Tier | Items | LOC estimado | Tempo |
|---|---|---|---|
| A | 8 | ~340 | ~5h |
| B | 8 | ~1650 | ~12-17 dias |
| C | 7 | ~640 | ~8-10 dias |
| D | 12+ | ~4200 (code + docs) | ~25-30 dias |
| **TOTAL** | **~35** | **~6830 LOC** | **~6-8 semanas** |

**Paralelização recomendada** (se múltiplas instâncias LLM disponíveis):
- Instância 1: Tier A + Tier C (items independentes, schema + snippets)
- Instância 2: Tier B items B1-B4 (channels + hooks + memory)
- Instância 3: Tier B items B5-B8 + Tier D D3 (mcp + e2e + tests)
- Depois single: Tier D D1 + D2 + D4 (ordem: scripts → CLI → docs)

**Tempo paralelo:** 2-3 semanas.

---

## Snippets já escritos em EXTRACTION-PLAN.md

Os 10 snippets na seção "SPECIFIC CODE SNIPPETS" do EXTRACTION-PLAN.md **já estão escritos em Go**. Use-os como ponto de partida literal, apenas adapte:

1. RecallScore Composite → serve B4
2. MCP NameRegistry → serve A4 + parte de B5
3. Hook Event Bus → serve B3
4. Episode Eviction Reconciler → serve B7
5. Benchmark Stats → serve A1 + D1.2
6. InboundRouter → serve B1
7. Wave Fallback → serve B6 + C1
8. Docker Sandbox Config Snippet → serve parte de D1.3
9. Recipe `requires` Pre-flight → serve C2
10. Scope-Aware Memory Retrieval → serve C5

**Comece pelos snippets.** Você já tem 60% do código Tier A/B base. O resto é expandir com tests + doc comments + error handling.

---

## Items EXPLICITAMENTE NÃO portar

Mesmo aparecendo em notes originais:

- Canvas/A2UI mobile (§10 DESCARTADO) — paradigma mobile não cabe PM-OS
- Gateway WebSocket broker platform abstraction — systemd local basta
- Plugin SDK in-process — subprocess já é superior
- Memory Wiki Obsidian — niche, defer
- LLM provider zoo (OpenAI/Gemini/etc) — manter AnthropicDirect único por enquanto
- Mobile native apps (iOS/macOS/Android) — web-first
- Mesh gateway multi-region
- Auto-reply logic src/auto-reply/ — complexo, defer

Se qualquer desses aparecer por engano, pule.
