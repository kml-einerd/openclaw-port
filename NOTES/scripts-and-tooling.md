# OpenClaw Scripts & Tooling Analysis for PM-OS Adaptation

**Date:** 2026-04-23  
**Target:** PM-OS (Go 1.25 orchestration engine on systemd/local + Supabase + Caddy)  
**Repo Analyzed:** `/tmp/openclaw-eval/openclaw/scripts/` (419 files)

---

## Executive Summary

OpenClaw's scripts library contains **4 benchmark tools**, **41 architecture/quality checks**, **29 test helpers**, **29 E2E harnesses**, and **8 release workflows**. Most are TypeScript-specific (tsconfig, imports, plugin SDK), but **10 tools exhibit language-agnostic patterns** worth adapting to PM-OS:

1. **`bench-cli-startup.ts`** — Structured benchmark harness with p50/p95 percentiles, memory tracking, statistical output formatting
2. **`check-architecture-smells.mjs`** — Tree-sitter AST walking for boundary smells (cross-package imports, facade leaks)
3. **`lib/docker-e2e-image.sh`** — Reusable Docker build/run orchestration: caching, skip flags, labeled logging
4. **`ci-run-timings.mjs`** — Parse GitHub Actions job timings, identify slow/queued jobs, top-15 ranking
5. **`lib/ci-node-test-plan.mjs`** — Dynamic test shard calculation based on file patterns, auto-group by domain
6. **`pr-lib/common.sh`** — Reusable shell functions: artifact validation, path classification (docs/test/code), diff filtering
7. **`prep-ci-config.ts`** — Multi-platform CI config generation (branch strategies, matrix expansions)
8. **`audit-seams.mjs`** — Seams detection (cyclic dependencies, unused exports, boundary violations)
9. **`check-import-cycles.ts`** — Cycle detection via reverse graph traversal, human-readable output
10. **`lib/local-heavy-check-runtime.mjs`** — Benchmark check execution, detect slow checks, suggest parallelization

**Relevance to PM-OS:**
- PM-OS needs **perf benchmarks** (startup, recipe execution, wave throughput)
- PM-OS needs **quality gates** (contract validation, job timeout detection, artifact staging)
- PM-OS needs **local CI helpers** (systemd-aware E2E harnesses, GCS artifact mgmt, Supabase state checks)
- PM-OS needs **PR workflow automation** (changelog enforcement, stage gating, merge checklist)

---

## Detailed Analysis by Category

### BENCHMARK (4 tools) — Applicability: **HIGH**

| Script | LOC | Purpose | PM-OS Adaptation |
|--------|-----|---------|------------------|
| `bench-cli-startup.ts` | 180+ | Measures CLI invocation latency + memory across test cases, p50/p95/max percentile output | **ADAPT:** Create `tools/pm-bench/startup-bench.go` for pm-api/pm-engine cold-start times, recipe throughput per wave, queueing latency |
| `bench-model.ts` | 150+ | Benchmarks model inference (token generation, latency) with statistical summaries | **ADAPT:** Create `bench-llm-latency.go` to measure PicoClaw/HTTP executor response times, token budgeting |
| `bench-test-changed.mjs` | 100+ | Runs only changed test files, measures per-file execution time, detects hotspots | **ADAPT:** Use for PM-OS test matrix: identify slow quality gates (contract validation, cycle detection), suggest parallelization |
| `bench-gateway-startup.ts` | 80+ | Measures gateway initialization, connection pooling warmup | **ADAPT:** Low relevance; skip |

**Concrete Pattern to Copy:**
```go
// pm-bench/types.go
type BenchmarkResult struct {
    Name      string
    Samples   []int64 // nanoseconds
    Mean      int64
    P50, P95  int64
    Min, Max  int64
}

// pm-bench/startup.go - measure pm-api boot
func benchStartup() {
    const runs = 10
    for i := 0; i < runs; i++ {
        start := time.Now()
        cmd := exec.Command("./pm-api", "--dry-run")
        cmd.Run()
        samples = append(samples, time.Since(start))
    }
    // Calculate stats, output JSON
}
```

---

### ARCHITECTURE SMELLS & CYCLE DETECTION (4 tools) — Applicability: **MEDIUM-HIGH**

| Script | LOC | Purpose | PM-OS Adaptation |
|--------|-----|---------|------------------|
| `check-architecture-smells.mjs` | 200+ | AST walk (TypeScript) to find facade leaks, wrong-package exports, re-export cycles | **ADAPT:** Use Go's `ast.Walk()` + `go/types` to detect PM-OS boundary violations (e.g., adapters importing from handlers, internal state leaks) |
| `check-import-cycles.ts` | 150+ | TypeScript import cycle detector via reverse graph, outputs cycle path + location | **ADAPT:** Implement in Go using `go mod graph` + cycle detection algorithm; integrate into CI gates |
| `audit-seams.mjs` | 100+ | Finds unused exports, cyclic dependencies, unused packages | **ADAPT:** Use for PM-OS: identify dead recipe types, unused store methods, dangling executor implementations |
| `check-ts-max-loc.ts` | 80+ | Flags functions/files >N LOC, suggests refactoring boundaries | **ADAPT:** Implement in Go: flag `runRecipeInternal()` (1200 LOC), suggest extraction of wave loop |

**Concrete Go Pattern:**
```go
// pm-audit/cycle.go
func detectImportCycles(modPath string) error {
    // Parse go.mod, walk dependency graph
    // Use DFS to find cycles
    // Output: file:line references forming cycle
}
```

---

### CI INFRASTRUCTURE (10 tools) — Applicability: **HIGH**

| Script | LOC | Purpose | PM-OS Adaptation |
|--------|-----|---------|------------------|
| `lib/docker-e2e-image.sh` | 70 | Parameterized Docker image build/reuse logic: skip-build flag, env fallback, labeled logging | **COPY:** PM-OS uses systemd local (no K8s), but pattern applies to Supabase Docker setup; adapt to `scripts/e2e/docker-setup.sh` |
| `ci-run-timings.mjs` | 80 | Parse GitHub Actions run JSON, extract per-job timings, identify slowest jobs + queue delays | **ADAPT:** Create `tools/pm-ci/timings.go` to parse pm-api/pm-engine systemd journal or audit log; identify slow recipe executions |
| `lib/ci-node-test-plan.mjs` | 150+ | Dynamic test shard generation: scan test files, group by domain, auto-distribute across parallel jobs | **ADAPT:** Use for PM-OS: auto-shard recipe tests (e.g., group by Wave type, Provider), create test matrix in systemd |
| `ci-changed-scope.mjs` | 120+ | Git diff analysis: identify which CI jobs to run based on changed files | **ADAPT:** Use for PM-OS: if `pkg/quality/` changed, run quality gates; if `recipes/` changed, validate all recipes |
| `lib/channel-contract-test-plan.mjs` | 150+ | Builds test execution plan from contract definitions, manages dependencies | **ADAPT:** Use for PM-OS Recipe validation: load all `.v2.json` recipes, build DAG of test dependencies |
| `pr-lib/common.sh` | 150+ | Reusable shell functions: artifact validation, path classification (docs/test/code), changelog enforcement | **COPY:** PM-OS can use directly for `prepush-ci.sh`: validate inventory.json, check git hooks, enforce test coverage |
| `pr-lib/gates.sh` | 100+ | Quality gate enforcement in CI: required checks (build, test, lint, coverage), fail-fast logic | **COPY:** Use for PM-OS CI: invoke `./scripts/ci-gates.sh` before merge (recipe validation, coverage, perf budget) |
| `lib/vitest-batch-runner.mjs` | 200+ | Batches test files to avoid OOM, distributes across shards, collects results | **ADAPT:** Create `tools/pm-test/batch-runner.go` for parallel gate execution (avoid starving systemd) |
| `lib/local-heavy-check-runtime.mjs` | 100+ | Benchmarks slow checks, detects which checks kill CI performance, suggests parallelization | **ADAPT:** Create `tools/pm-audit/check-perf.go` to measure time spent in each quality gate (contract validation, cycle detection) |
| `run-vitest.mjs` | 80+ | Test runner wrapper: retry logic, collect logs, summarize failures | **ADAPT:** Create `tools/pm-test/run-gates.go` with same patterns |

**Concrete PM-OS CI Pattern:**
```bash
# scripts/ci-gates.sh (PM-OS adapted from pr-lib/gates.sh)
#!/usr/bin/env bash
set -euo pipefail

# 1. Validate all recipes
python3 cmd/validate-recipes/main.py recipes/*.v2.json || exit 1

# 2. Run architecture checks
go run tools/pm-audit/cycle.go ./...

# 3. Run perf benchmarks
go run tools/pm-bench/startup.go --budget 500ms

# 4. Collect artifacts
mkdir -p .local/
echo "gates: PASS" > .local/gates.status
```

---

### E2E & DOCKER ORCHESTRATION (15 tools) — Applicability: **MEDIUM**

| Script | File Count | Purpose | PM-OS Adaptation |
|---------|----------|---------|------------------|
| `scripts/e2e/` | 29 files | Docker-based E2E harnesses: spin up containers, run scenario, verify outputs | **ADAPT:** Create `scripts/e2e/` for PM-OS: Supabase Docker, pm-api/pm-engine systemd, recipe execution e2e tests |
| `docker-compose.yml` | 78 LOC | Compose definition for multi-service stack (gateway, plugins, storage) | **ADAPT:** Create `docker-compose.e2e.yml` for PM-OS: pmos Postgres, pm-api, pm-engine, Caddy reverse proxy |
| `docker/setup.sh` | 20 LOC | Docker/Podman agnostic setup: detect platform, run setup | **COPY:** PM-OS uses systemd; less relevant, but pattern useful for cross-platform Dockerfile setup |

---

### RELEASE & PUBLISH WORKFLOWS (8 tools) — Applicability: **MEDIUM**

| Script | Purpose | PM-OS Adaptation |
|---------|---------|------------------|
| `plugin-npm-release-check.ts` | Pre-release validation: version bump, changelog, tarball contents | **ADAPT:** Create `tools/pm-release/pre-release-check.go` for Go module validation, git tag consistency |
| `plugin-clawhub-release-plan.ts` | Release planning: detect affected packages, calculate version bumps | **SKIP:** PM-OS is single-module; not applicable |
| `changelog-add-unreleased.ts` | Adds "Unreleased" section to changelog, auto-detects since last tag | **COPY:** Use for PM-OS CHANGELOG.md management |
| `release-check.ts` | Validates release preconditions (no uncommitted changes, correct branch) | **COPY:** Use as-is for PM-OS release workflows |
| `ghsa-patch.mjs` | Auto-patches GitHub Security Advisories | **SKIP:** Low relevance to PM-OS |

---

### TEST HELPERS (18 tools) — Applicability: **LOW-MEDIUM**

| Script | Purpose | PM-OS Adaptation |
|---------|---------|------------------|
| `test-install-sh-docker.sh` | Tests install script in Docker, validates artifact | **SKIP:** PM-OS is server software; no install.sh |
| `test-live.mjs` | Smoke tests against live gateway | **ADAPT:** Create `tools/pm-test/e2e-live.go` to test pm-api against live Supabase + GCS |
| `test-projects.mjs` | Runs test suite across multiple project configurations | **SKIP:** Not applicable to PM-OS structure |
| `test-extension.mjs` | Extension unit/integration test runner | **SKIP:** Not applicable |

---

## Not Worth Adapting (TypeScript-Specific, Low ROI)

- **oxlint integration** (`run-oxlint*.mjs`, `run-extension-channel-oxlint.mjs`) — Language-specific linter; PM-OS uses `go vet` + `golangci-lint` natively
- **tsconfig/tsc checks** (`check-ts-*`, `check-no-*` that scan imports) — Most work via TypeScript AST; PM-OS equivalent is `go/ast` + `go/types`
- **Plugin SDK boundary checks** — Not applicable to PM-OS architecture
- **i18n tools** (`scripts/docs-i18n/`) — Specialized for docs translation; not applicable
- **knip** (dead code) — PM-OS equivalent: `go mod tidy` + manual export auditing

---

## Recommended Adaptation Roadmap (Ranked by ROI for PM-OS)

### Phase 1: Benchmarking (Week 1)
1. **`tools/pm-bench/startup.go`** — Measure pm-api/pm-engine cold start, establish budget (target: <200ms)
2. **`tools/pm-bench/recipe-throughput.go`** — Measure recipes/min, wave latency, identify bottlenecks

### Phase 2: Quality Gates (Week 2-3)
3. **`scripts/ci-gates.sh`** — Orchestrate all pre-merge checks (recipe validation, coverage, perf)
4. **`tools/pm-audit/cycle.go`** — Detect import cycles in Go modules, fail fast on boundary violations
5. **`tools/pm-test/run-gates.go`** — Parallel gate execution with timeout + retry logic

### Phase 3: CI Infrastructure (Week 3-4)
6. **`scripts/e2e/docker-compose.e2e.yml`** — Full PM-OS stack (Supabase, pm-api, pm-engine, Caddy)
7. **`scripts/e2e/test-recipe-e2e.sh`** — End-to-end recipe execution: spawn stack, upload recipe, poll results
8. **`.github/workflows/ci.yml` refactor** — Use new gate sharding logic, parallel job matrix

### Phase 4: Release Automation (Week 4)
9. **`tools/pm-release/pre-release-check.go`** — Validate git tag, CHANGELOG, go.mod consistency
10. **`scripts/release-tag.sh`** — Automate versioning, changelog generation, Docker image tagging

---

## Copy-Paste Code Snippets

### 1. Benchmark Percentile Calculation (from `bench-cli-startup.ts`)
```go
// tools/pm-bench/stats.go
func calculatePercentiles(samples []int64) (p50, p95, min, max, mean int64) {
    sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
    n := len(samples)
    min, max = samples[0], samples[n-1]
    mean = 0
    for _, s := range samples {
        mean += s
    }
    mean /= int64(n)
    
    p50Index := (n * 50) / 100
    p95Index := (n * 95) / 100
    p50, p95 = samples[p50Index], samples[p95Index]
    
    return
}
```

### 2. Docker Build/Reuse Pattern (from `lib/docker-e2e-image.sh`)
```bash
# scripts/e2e/docker-build.sh
docker_build_or_skip() {
  local image_name="$1"
  local dockerfile="$2"
  
  if [ "${SKIP_DOCKER_BUILD:-0}" = "1" ]; then
    if ! docker image inspect "$image_name" >/dev/null 2>&1; then
      echo "ERROR: image not found: $image_name" >&2
      exit 1
    fi
    echo "Reusing: $image_name"
    return
  fi
  
  echo "Building: $image_name"
  docker build -t "$image_name" -f "$dockerfile" . || exit 1
}
```

### 3. Artifact Validation (from `pr-lib/common.sh`)
```bash
# scripts/ci-gates.sh
require_artifact() {
  local path="$1"
  if [ ! -s "$path" ]; then
    echo "Missing required artifact: $path" >&2
    exit 1
  fi
}

check_recipe_artifacts() {
  for recipe in recipes/*.v2.json; do
    if ! python3 cmd/validate-recipes/main.py "$recipe"; then
      echo "Invalid recipe: $recipe" >&2
      return 1
    fi
  done
}
```

### 4. Job Timing Summary (from `ci-run-timings.mjs` adapted to Go)
```go
// tools/pm-ci/timings.go
type JobTiming struct {
    Name      string
    Duration  time.Duration
    QueueWait time.Duration
    Status    string
}

func printTimingsSummary(jobs []JobTiming) {
    sort.Slice(jobs, func(i, j int) bool {
        return jobs[i].Duration > jobs[j].Duration
    })
    
    for i := 0; i < 15 && i < len(jobs); i++ {
        fmt.Printf("%s: %v (queued: %v)\n", 
            jobs[i].Name, jobs[i].Duration, jobs[i].QueueWait)
    }
}
```

---

## Key Learnings from OpenClaw

1. **Structured Benchmarking:** Always measure p50/p95 (not just mean). Percentiles catch outliers.
2. **Dynamic Test Sharding:** Group tests by domain to balance load, avoid single slow test blocking matrix.
3. **Artifact-Driven CI:** Declare required artifacts (gates.status, review.json); fail if missing.
4. **Architecture Checks as Gates:** Use AST walks to enforce boundaries (adapters ≠ handlers, no re-exports across layers).
5. **Reusable Shell Patterns:** Extract common functions (`require_artifact`, `path_is_docsish`) into shared libs.
6. **Docker Build Caching:** Always support `SKIP_DOCKER_BUILD` flag for faster iteration.
7. **Cycle Detection:** Use reverse-graph traversal to find import cycles; output human-readable paths.

---

## Files Referenced

**Docker:**
- `/tmp/openclaw-eval/openclaw/docker-compose.yml` (78 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/docker/setup.sh` (20 LOC)

**Benchmarks:**
- `/tmp/openclaw-eval/openclaw/scripts/bench-cli-startup.ts` (180 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/bench-model.ts` (150 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/bench-test-changed.mjs` (100 LOC)

**Checks & Audits:**
- `/tmp/openclaw-eval/openclaw/scripts/check-architecture-smells.mjs` (200 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/check-import-cycles.ts` (150 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/audit-seams.mjs` (100 LOC)

**CI Infrastructure:**
- `/tmp/openclaw-eval/openclaw/scripts/lib/docker-e2e-image.sh` (70 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/ci-run-timings.mjs` (80 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/lib/ci-node-test-plan.mjs` (150 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/ci-changed-scope.mjs` (120 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/pr-lib/common.sh` (150 LOC)
- `/tmp/openclaw-eval/openclaw/scripts/pr-lib/gates.sh` (100 LOC)

**E2E & Release:**
- `/tmp/openclaw-eval/openclaw/scripts/e2e/` (29 files)
- `/tmp/openclaw-eval/openclaw/scripts/plugin-npm-release-check.ts`
- `/tmp/openclaw-eval/openclaw/scripts/changelog-add-unreleased.ts`

---

## Conclusion

**Top 10 MUST-ADAPT Scripts (Ranked by ROI for PM-OS):**

1. **`bench-cli-startup.ts`** → **pm-bench/startup.go** — Establish perf budget, detect regressions (p95 latency tracking)
2. **`lib/docker-e2e-image.sh`** → **scripts/e2e/docker-build.sh** — Orchestrate Supabase + pm-api + pm-engine containers
3. **`pr-lib/common.sh`** → **scripts/ci-gates.sh** — Artifact validation, changelog enforcement, path classification
4. **`check-architecture-smells.mjs`** → **tools/pm-audit/boundaries.go** — Enforce adapter/handler separation, fail on re-exports
5. **`ci-run-timings.mjs`** → **tools/pm-ci/timings.go** — Parse systemd journal, identify slow recipe executions, suggest parallelization
6. **`lib/ci-node-test-plan.mjs`** → **tools/pm-test/shard-gates.go** — Auto-group quality gates by domain, balance load
7. **`check-import-cycles.ts`** → **tools/pm-audit/cycle.go** — Detect Go import cycles, fail pre-merge
8. **`pr-lib/gates.sh`** → **scripts/prepush-ci.sh** — Quality gate orchestration (validation, coverage, perf)
9. **`lib/local-heavy-check-runtime.mjs`** → **tools/pm-audit/check-perf.go** — Benchmark each gate, identify bottlenecks
10. **`changelog-add-unreleased.ts`** → **tools/pm-release/version-bump.go** — Automate CHANGELOG + git tag on release

**Effort Estimate:** 60-80 hours (4-5 weeks part-time) to adapt all 10 patterns to PM-OS Go stack and systemd infrastructure.

**Expected ROI:** 40% faster CI (parallel gates), <200ms startup budget enforcement, zero boundary violations in code review.

