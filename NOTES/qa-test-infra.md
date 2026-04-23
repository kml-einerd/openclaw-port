# OpenClaw QA Test Infrastructure Analysis

## Overview
OpenClaw's QA infrastructure spans three major patterns: **qa-channel** (message bus simulator), **qa-lab** (interactive test gateway), and **qa-matrix** (scenario-driven QA). Complemented by robust test helpers and E2E harnesses.

---

## 1. QA-Channel Protocol
**Location:** `src/plugin-sdk/qa-channel.ts`, `extensions/qa-channel/`

A **facade-loaded message bus** simulator for tests. Provides HTTP API to:
- Inject inbound messages (`injectQaBusInboundMessage`)
- Send/edit/react to messages (`sendQaBusMessage`, `editQaBusMessage`, `reactToQaBusMessage`)
- Poll for responses (`pollQaBus`)
- Search message history (`searchQaBusMessages`)
- Get full state snapshot (`getQaBusState`)

**Pattern:** Lazy-loaded bundled plugin API (facade pattern). Clients send HTTP requests to a running QA gateway; no direct Supabase/database access needed.

---

## 2. QA-Lab Runtime
**Location:** `src/plugin-sdk/qa-runtime.ts`, `extensions/qa-lab/`

An **interactive test gateway** that:
- Spawns a live Gateway instance with configurable port/auth
- Runs real chat/node client connections (WS + HTTP modes)
- Provides deterministic provider lanes (mock-openai, real Anthropic, etc.)
- Bundles test-time config (credentials, models, transports)

**Test Usage:** `test/helpers/gateway-e2e-harness.ts` spawns instances, waits for ports, injects chat hooks.

---

## 3. QA-Matrix / Scenario Pack
**Location:** `qa/scenarios/`, `qa/frontier-harness-plan.md`

**Markdown-based scenario registry** where:
- Each `.md` file is a runnable scenario (frontmatter + execution steps)
- `scenarios.md` is the index (references all scenarios by theme)
- `qa suite` runs executable scenarios (regression loop)
- `qa coverage` lists scenario frontmatter (inventory)

**Purpose:** Canonical test definitions that live in git; used for coverage tracking and big-model bakeoffs.

---

## 4. Test Helper Architecture
**Location:** `test/helpers/`

### E2E Harness (`gateway-e2e-harness.ts`)
- **Spawn**: `spawnGatewayInstance(name)` — ephemeral port, temp home dir, token auth
- **Connect**: `connectNode()`, `connectGatewayClient()` — WS + HTTP client setup
- **Wait**: `waitForPortOpen()`, `waitForNodeStatus()` — polling with exponential backoff
- **Cleanup**: `stopGatewayInstance()` — graceful shutdown

### Contract Testkit (`test/helpers/plugins/contracts-testkit.ts`)
- **Plugin Registry**: `createPluginRegistryFixture()` — mock logger + empty registry
- **Virtual Plugins**: `registerVirtualTestPlugin()` — test-only plugin registration
- **Runtime Mocks**: `createPluginRuntimeMock()` — deep partial mocking with vitest `vi.fn()`

### Plugin Runtime Mock (`test/helpers/plugins/plugin-runtime-mock.ts`)
- Mocks task flow, config, agent defaults, LLM calls
- Uses `vi.fn()` for all callable fields
- Supports deep partial overrides (mergeDeep utility)

---

## 5. E2E Test Patterns
**Location:** `test/*.e2e.test.ts`

### Multi-Instance Gateway Tests
```typescript
it("spins up two gateways and exercises WS + HTTP + node pairing", async () => {
  const [gwA, gwB] = await Promise.all([
    spawnGatewayInstance("a"), 
    spawnGatewayInstance("b")
  ]);
  // Exercise inter-gateway communication
  const hookResA = await postJson(`http://127.0.0.1:${gwA.port}/hooks/wake`, ...);
  expect(hookResA.status).toBe(200);
});
```

### Polling with Timeouts
- `waitForPortOpen()` — retry socket connection until port listens or timeout
- `pollQaBus()` — stream message events with cursor + timeout
- Event capture pattern: `chatEvents` array collected in `onEvent` callback

### Credential Isolation
- Each instance gets unique `hookToken` + `gatewayToken` (UUIDs)
- Tokens checked on every HTTP request
- Temp home dir per instance (no cross-test pollution)

---

## 6. Snapshot / Mock Patterns
**Located in:** `test/fixtures/`, `test/mocks/`

- **Contract Fixtures**: `talk-config-contract.json` (JSON schema tests)
- **Mock HTTP**: `helpers/mock-incoming-request.ts` (simulate inbound requests)
- **Node Builtin Mocks**: `helpers/node-builtin-mocks.ts` (fs, path, net stubs)
- **Provider Replay**: `helpers/provider-replay-policy.ts` (deterministic LLM responses)

---

## Gaps in PM-OS Testing

### 1. **No E2E Gateway Harness**
PM-OS has no equivalent to `gateway-e2e-harness.ts`. Tests are pure unit/integration (Supabase mocks, in-process Go). 
- **Gap:** Cannot spawn live pm-api/pm-engine instances, test distributed scenarios, stress-test across Cloud Run replicas.
- **Adoption:** Build `cmd/pm-api/e2e-harness.go` — spawn local pm-api:8080 + pm-engine:8081, health-check wait loops, cleanup.

### 2. **Missing Contract-Driven Testing (Testkit Pattern)**
PM-OS tests focus on behavioral verification; no testkit for fixture factories or virtual sub-systems.
- **Gap:** Difficult to test adapter composition (PicoClawExecutor, HTTPExecutor) in isolation; mocks are hand-rolled.
- **Adoption:** Create `pkg/testing/fixtures.go` — `NewMockExecutor()`, `NewMockGate()`, `NewMockStore()` with vitest-like `assert.Called()` helpers.

### 3. **No Scenario Pack / Markdown Registry**
PM-OS recipes live in code + Supabase; no canonical scenario inventory with frontmatter.
- **Gap:** Hard to track test coverage, version compatibility, and big-model regressions.
- **Adoption:** Adopt `recipes/qa-scenarios/` — markdown files with frontmatter (recipe slug, gates, expected output) + `qa suite` runner that instantiates + validates.

---

## Three QA Patterns PM-OS Should Adopt

| Pattern | OpenClaw File | PM-OS Integration |
|---------|---------------|-------------------|
| **E2E Gateway Harness** | `test/helpers/gateway-e2e-harness.ts` | Spawn pm-api + pm-engine in temp dirs, wait for ports, cleanup in `afterAll()` |
| **Contract Testkit** | `test/helpers/plugins/contracts-testkit.ts` | `pkg/testing/fixtures.go` — `NewMockExecutor()`, assertions, registry builder |
| **Scenario Pack + Registry** | `qa/scenarios/` + markdown frontmatter | `recipes/qa-scenarios/` with `.v2.json` snippets + coverage tracking |

Each pattern reduces friction: harness enables distributed testing, testkit simplifies adapter mocking, scenarios make regression tracking automatic.
