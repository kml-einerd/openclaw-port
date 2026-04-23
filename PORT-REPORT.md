# Port Report — OpenClaw → PM-OS

## Summary

- Items completed: 35 / 35
- LOC Go produced: ~2400
- LOC bash/yaml/json/md: ~350
- Average test coverage: 93.2%
- External deps added beyond standard: `github.com/stretchr/testify`, `gopkg.in/yaml.v3`, `golang.org/x/tools`

## Per-Tier Status

### Tier A — Copy-Paste (8 items)
- [x] A1 bench_stats (40 LOC, 4 tests, 100% cov)
- [x] A2 docker-build.sh (25 LOC bash)
- [x] A3 artifact_validation.sh (25 LOC bash)
- [x] A4 MCP Tool Name Sanitization (50 LOC, 6 tests, 94.7% cov)
- [x] A5 Skill YAML Frontmatter Schema (50 LOC, 2 tests, 100% cov)
- [x] A6 Dangerous Tools Deny List (20 LOC, 1 test, 100% cov)
- [x] A7 Job Timing Summary (65 LOC, 2 tests, 94.1% cov)
- [x] A8 Recency Half-Life Decay (55 LOC, 2 tests, 100% cov)

### Tier B — Port-and-Adapt (8 items)
- [x] B1 Channel Inbound Router (40 LOC, 2 tests, 100% cov)
- [x] B2 Telegram Webhook Handler (70 LOC, 4 tests, 100% cov)
- [x] B3 Global Hook Event Bus (55 LOC, 2 tests, 100% cov)
- [x] B4 Memory Multi-Component Scorer (55 LOC, 3 tests, 100% cov)
- [x] B5 MCP Loopback Gateway (85 LOC, 2 tests, 97.4% cov)
- [x] B6 Wave-Level Model Fallback (55 LOC, 5 tests, 97.6% cov)
- [x] B7 Episode Auto-Eviction Reconciler (50 LOC, 1 test, 100% cov)
- [x] B8 E2E Harness pm-api + pm-engine (85 LOC, 3 tests, 58.5% cov)

### Tier C — Capabilities JSON (7 items)
- [x] C1 `fallback_models` na Wave (18 LOC, 1 test, 100% cov)
- [x] C2 `requires` na Recipe (28 LOC, 2 tests, 100% cov)
- [x] C3 `trigger_phrases` na Recipe description (18 LOC, 1 test, 100% cov)
- [x] C4 `retention_policy` na Quality config (25 LOC, 3 tests, 100% cov)
- [x] C5 `collection_hint` no Step (22 LOC, 2 tests, 100% cov)
- [x] C6 `requires_tools` no Skill frontmatter (45 LOC, 5 tests, 100% cov)
- [x] C7 `webhook_trigger` no Sensor (35 LOC, 4 tests, 100% cov)

### Tier D — CLI / Tools / Tests / Skills (12+ items)
- [x] D1.1 ci-gates.sh (20 LOC bash)
- [x] D1.2 pm-bench/startup.go (55 LOC Go, builds clean)
- [x] D1.3 docker-compose.e2e.yml (38 LOC yaml)
- [x] D1.4 pm-audit/cycle.go (75 LOC Go, builds clean)
- [x] D1.5 prepush-ci.sh (12 LOC bash)
- [x] D2.1 pm-cli doctor (95 LOC, 1 test, 81.5% cov)
- [x] D2.2 pm-cli init --wizard (65 LOC, 3 tests, 81.5% cov)
- [x] D2.3 pm-cli bench (75 LOC, 2 tests, 81.5% cov)
- [x] D2.4 pm-cli channels setup (55 LOC, 4 tests, 81.5% cov)
- [x] D3.1 Mock Factory Kit (35 LOC, self-tested via other pkg tests)
- [x] D3.2 Scenario Pack Registry (60 LOC, 2 tests, 58.5% cov)
- [x] D3.3 Provider Replay (40 LOC, 1 test, 58.5% cov)
- [x] D4.1 Skill dir structure template (README.md + SKILL.md)
- [x] D4.2 Raven Integration Hook / Skill Matcher (65 LOC, 3 tests, 90% cov)
- [x] D4.3 Doc frontmatter pattern (doc-frontmatter.md)
- [x] D4.4 Glossary PT-BR (glossary-pt-br.json, 20 entries)
- [x] D4.5 FAQ "First 60 Seconds" pattern (first-60s.md)

## Deviations

### B8 E2E Harness
- Coverage at 58.5% because `StartHarness` spawns real OS processes that can't be fully tested without real PM-OS binaries. The `WaitForHealth` function is fully tested via `httptest`.

### D1.2 / D1.4 (cmd/ binaries)
- Moved from `tools/` to `cmd/` to avoid package name conflicts (`main` vs library). The `ci-gates.sh` script references updated paths.

### D2.1-D2.4 CLI commands
- Implemented as library code in `pkg/cli/` rather than directly in `cmd/pm-cli/` with cobra registration. This allows full testing without cobra dependency. Akita should wire these into the cobra command tree.

### D4.2 Skill Matcher
- Uses simple keyword overlap scoring rather than vector embeddings. This is a usable baseline; Raven v2 will replace it with semantic similarity.

## TODOs para Akita

### High priority (blocking integration)
- [ ] Wire `pkg/channels/router.go` into `cmd/pm-api/main.go` HTTP mux
- [ ] Register `pkg/cli/` commands in cobra root (`cmd/pm-cli/`)
- [ ] Migration SQL for `mcp_cache` table (B5 LoopbackGateway)
- [ ] Add `fallback_models` field to existing `recipe.Wave` struct

### Medium priority
- [ ] Consolidate `pkg/hooks/event_bus.go` (B3) with existing `pkg/events` if present
- [ ] Memory reconciler (B7) needs cron/systemd timer registration
- [ ] Replace skill_matcher keyword scoring with embedding-based similarity
- [ ] Wire `WaitForHealth` into real E2E test setup

### Questions for Akita
- Should pm-cli doctor check Caddy config?
- Channel router: path-based vs subdomain-based webhooks?
- Integration with existing `pkg/memory/scope.go` — merge or coexist?
- `collection_hint` valid values — are "sessions/episodes/knowledge/scratchpad" correct?

## External deps

| dep | version | reason | which file uses it |
|---|---|---|---|
| github.com/stretchr/testify | v1.11.1 | test assertions | all _test.go files |
| gopkg.in/yaml.v3 | v3.0.1 | YAML frontmatter parsing | pkg/skills/, pkg/testing/scenarios.go |
| golang.org/x/tools | v0.44.0 | Go package loading for cycle detection | cmd/pm-audit/main.go |

## Test execution

```
ok  openclaw-port/pkg/channels     coverage: 100.0% of statements
ok  openclaw-port/pkg/cli          coverage: 81.5% of statements
ok  openclaw-port/pkg/hooks        coverage: 100.0% of statements
ok  openclaw-port/pkg/mcp          coverage: 97.4% of statements
ok  openclaw-port/pkg/memory       coverage: 100.0% of statements
ok  openclaw-port/pkg/planner      coverage: 90.0% of statements
ok  openclaw-port/pkg/recipe       coverage: 97.6% of statements
ok  openclaw-port/pkg/sensors      coverage: 100.0% of statements
ok  openclaw-port/pkg/skills       coverage: 100.0% of statements
ok  openclaw-port/pkg/testing      coverage: 58.5% of statements
ok  openclaw-port/tools/pm-audit   coverage: 94.1% of statements
ok  openclaw-port/tools/pm-bench   coverage: 100.0% of statements
```

Build validation:
```
go build ./...    — PASS (zero errors)
go vet ./...      — PASS (zero warnings)
```

## Known issues

- D1.4 cycle detector: requires `golang.org/x/tools/go/packages` which needs `go` in PATH at runtime
- B8 E2E harness: needs real PM-OS binaries for full integration; unit tests use mock processes
- D2.3 bench: `cmd` binary used as stub in Windows tests; Linux CI should use actual pm-api binary
