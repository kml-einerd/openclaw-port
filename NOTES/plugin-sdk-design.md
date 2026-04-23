# OpenClaw Plugin Architecture Analysis

## Executive Summary

OpenClaw's plugin system demonstrates a **capability-driven extensibility model** with three core patterns worth adapting for PM-OS:

1. **Multi-slot registration** (tools, providers, commands, hooks, channels, services) — plugins extend via named factories + dependency injection, not monolithic inheritance
2. **Package contract & version compatibility** — `openclaw.compat.pluginApi` + `openclaw.build` metadata in `package.json` enables safe external plugins without runtime coupling
3. **Zero-isolation/shared heap** — plugins execute in-process with shared context (config, auth, state), plus explicit `PluginRuntime`/`SecurityAuditCollector` surfaces for audit/veto

PM-OS should adopt the **multi-slot registry pattern** and **contract-based versioning**, but **NOT** the shared-heap approach—instead, wrap tools as subprocess/HTTP (status quo).

---

## 1. Plugin Contract (Package Metadata)

**File:** `packages/plugin-package-contract/src/index.ts`

OpenClaw enforces a **two-field contract** on external plugins:

```json
{
  "openclaw": {
    "compat": {
      "pluginApi": "^1.0.0"    // Required: semver range of plugin SDK API
    },
    "build": {
      "openclawVersion": "1.2.3"  // Required: host version at build time
    },
    "install": {
      "minHostVersion": "1.2.0"   // Optional: minimum runtime version
    }
  }
}
```

**Validation:** `validateExternalCodePluginPackageJson()` enforces `compat.pluginApi` + `build.openclawVersion`, warns if missing.

**Why it works:**
- Decoupled: plugin declares its API contract, host validates at load time
- Versioning: semver ranges on `pluginApi` allow gradual breaking changes
- No registry lock-in: plugins can be local, npm, or tarball

---

## 2. Plugin Entry & Registration API

**File:** `src/plugin-sdk/plugin-entry.ts`

The canonical plugin entry point is **`definePluginEntry()`**, which normalizes the metadata + registration function:

```typescript
export function definePluginEntry({
  id: string;                              // plugin id (e.g., "openai", "custom-tool")
  name: string;                            // display name
  description: string;                     // human description
  kind?: PluginKind;                       // optional classification
  configSchema?: OpenClawPluginConfigSchema | (() => ...) // Zod-like parser + UI hints
  reload?: PluginDefinition["reload"];     // optional hot-reload config
  nodeHostCommands?: ...;                  // Node.js subprocess commands
  securityAuditCollectors?: ...;           // security veto hooks
  register: (api: OpenClawPluginApi) => void  // Main registration callback
}): DefinedPluginEntry
```

**The API object** `OpenClawPluginApi` exposes:
- `api.tools.register(factory)` — register tool factory
- `api.providers.register(provider)` — register model provider (search, speech, TTS, video, etc.)
- `api.commands.register(command)` — register CLI command
- `api.hooks.register(event, handler)` — register hook listener
- `api.services.register(service)` — long-lived service (e.g., cache warmer, metric collector)
- `api.channels.register(channel)` — communication channel (Discord, Slack, etc.)

**Design insight:** Multi-slot dispatch. Plugins don't implement one interface; they declare capabilities via the registration API. A single plugin can add tools, commands, AND a provider.

---

## 3. Security Model

**File:** `src/plugins/registry-types.ts` + `types.ts`

OpenClaw does **NOT sandbox plugins**. Instead, it uses:

### a) **Config Schema + Validation**
Every plugin can define `configSchema` (Zod-like or custom validate function). OpenClaw applies the schema at load time:
```typescript
export type OpenClawPluginConfigSchema = {
  safeParse?: (value) => { success: boolean; data?; error? };
  parse?: (value) => unknown;
  validate?: (value) => { ok: true | false; errors: string[] };
  uiHints?: Record<string, PluginConfigUiHint>;
  jsonSchema?: JsonSchemaObject;
};
```
Config errors block plugin load.

### b) **Security Audit Collectors**
Plugins can register a `securityAuditCollector` callback that runs post-load:
```typescript
export type OpenClawPluginSecurityAuditCollector = (
  ctx: OpenClawPluginSecurityAuditContext
) => SecurityAuditFinding[];
```
Used for vendor-specific veto checks (e.g., "no SSRF URLs", "no hardcoded API keys").

### c) **Explicit Capability Listing**
`PluginRecord` tracks what each plugin added:
```typescript
export type PluginRecord = {
  toolNames: string[];
  providerIds: string[];
  channelIds: string[];
  cliCommands: string[];
  services: string[];
  configSchema: boolean;
  contracts?: PluginManifestContracts;
  // ... etc
};
```
Host code can audit/veto at registration time.

**Assessment:** No runtime isolation (same Node.js heap as host), but explicit declaration + audit hooks provide a **capability-aware permission model**. Plugins can't hide what they do.

---

## 4. Discovery & Loading

**File:** `src/plugins/registry.ts`

Plugins are loaded via **source paths** (can be local, npm, or HTTP tarball). Registry holds:
- `plugins: PluginRecord[]` — loaded plugins + metadata
- `tools: PluginToolRegistration[]` — flattened tool list
- `providers: PluginProviderRegistration[]` — flattened provider list
- etc.

**Load sequence:**
1. Discover plugin source (package.json or inline manifest)
2. Validate contract (pluginApi + openclawVersion)
3. Load entry point (call `definePluginEntry()`)
4. Run `register(api)` callback to populate registry
5. Audit via `securityAuditCollector`
6. Mark as loaded/activated

No network registry; plugins are self-contained bundles with embedded contracts.

---

## 5. Memory-Host-SDK: Context/State Isolation

**File:** `src/memory-host-sdk/`

**Not a plugin security model**, but a **context-passing convention** for long-running plugins. Exports:
- `engine`: config, agent scope, memory search, embeddings
- `runtime`: environment variables, logger, file-safe I/O
- `secret`: credential resolution
- `files`: workspace file access
- `query`: vector store querying
- `status`: operation status tracking

**Purpose:** Plugins that need workspace state (tools, services, channels) receive a `PluginRuntime` object with scoped access. Enables plugin state isolation **by convention**, not sandboxing.

**PM-OS relevance:** Not directly applicable (PM-OS is stateless HTTP), but the pattern of **passing context objects instead of global state** is worth copying.

---

## Key Patterns Worth Adapting for PM-OS

### Pattern 1: Multi-Slot Registry
PM-OS currently has `pkg/engine/tools/registry.go` (Tool struct with Name/Description/Handler/Category). Expand to:
```go
type PluginRegistry struct {
  Tools        map[string]*Tool
  Steps        map[string]*StepType       // "llm", "function", "custom"
  Executors    map[string]Executor        // custom executors
  Gates        map[string]Gate            // custom quality gates
  Hooks        map[string][]Hook          // BeforeRun, AfterWave, AfterStep
  Providers    map[string]Provider        // LLM model providers
}
```
Let plugins register multiple capability types via a single `Register()` callback:
```go
func register(api *PluginAPI) {
  api.Tools.Register("my-tool", tool)
  api.Steps.Register("my-step-type", stepHandler)
  api.Gates.Register("my-gate", gateChecker)
}
```

### Pattern 2: Contract-Based Versioning
Replace manual registry.Register() calls with a `plugin.json` manifest:
```json
{
  "id": "my-plugin",
  "name": "My Plugin",
  "version": "1.0.0",
  "pmos": {
    "apiVersion": "^2.0.0",
    "builtWith": "2.1.3"
  },
  "capabilities": ["tools", "steps", "gates"]
}
```
Load plugins via:
```go
plugin := LoadPlugin("./my-plugin/plugin.json")
if !plugin.CompatWith(pm.Version) {
  return fmt.Errorf("incompatible: %v", plugin)
}
```
No code generation; just load and validate.

### Pattern 3: Subprocess Tool Isolation
**Do NOT copy in-process heap sharing.** Instead, extend `python_bridge.go` to support:
- Tool subprocess with NDJSON stdin/stdout (like PicoClaw)
- Tool receives `ToolContext` with recipe/run metadata
- Tool returns `ToolResult` with output + cost
- Death detection + restart on crash

Example tool plugin manifest:
```json
{
  "type": "subprocess",
  "entrypoint": "tool.py",
  "timeout": "30s",
  "restart": true
}
```

---

## Security Implications for PM-OS

**OpenClaw:** Shared heap (Node.js), explicit audit hooks + capability listing.
**PM-OS recommendation:** Keep current subprocess isolation (tools as processes), add:
1. **Manifest validation** (contract checking at load time)
2. **Capability audit** (log what each plugin registers)
3. **Startup gates** (fail fast if contract mismatch)

No new sandboxing needed; subprocess boundary is sufficient.

---

## Should PM-OS Build Its Own Plugin-SDK vs. Wrap MCP?

### Option A: Custom Plugin-SDK (Go)
**Pros:**
- Matches PM-OS architecture (recipe-first, subprocess tools)
- Contract validation is simple JSON parsing
- Registration is just function calls

**Cons:**
- New SDK to maintain
- No existing ecosystem

### Option B: Wrap MCP as Plugin-SDK
**Pros:**
- Reuse existing ecosystem (Claude, Anthropic, community MCP servers)
- No maintenance burden

**Cons:**
- MCP is JSON-RPC, not optimized for PM-OS flow control
- Contract model is different (capabilities list, not API versioning)
- Overkill for internal tools

### Recommendation: **Custom light SDK** (Option A)
Keep it minimal: `plugin.json` + loader + registry. No subprocess requirement—leave that to tool executors. MCP can be ONE type of tool (wrapper around MCP server subprocess), not the plugin model itself.

---

## Effort Estimate: Plugin System for PM-OS

1. **Plugin manifest + contract validation** (plugin.json schema)
   - Define schema, add validator
   - Effort: 2-3 days (one dev)

2. **Plugin loader + registry refactor**
   - Extend current `pkg/engine/tools/registry.go`
   - Add plugin discovery (local dirs, npm)
   - Effort: 3-5 days

3. **Example plugin** (custom step type or gate)
   - Demonstrate register() callback
   - Effort: 1 day

4. **Documentation + testing**
   - Plugin dev guide, contract examples
   - Effort: 2-3 days

**Total:** ~10-15 days (one dev) for a working, tested system. **Not a prerequisite for v2.x**—can ship as Wave 8+ enhancement.

---

## References

- **OpenClaw plugin SDK:** `packages/plugin-sdk/` (TypeScript interfaces + helpers)
- **External plugin contract:** `packages/plugin-package-contract/src/index.ts`
- **Registry types:** `src/plugins/registry-types.ts` + `types.ts`
- **Memory isolation:** `src/memory-host-sdk/` (context-passing pattern)
- **Example extension:** `extensions/vllm/` (simple provider plugin)
