# OpenClaw Gateway-Daemon Architecture Analysis

## Overview
OpenClaw separates concerns into two layers: **Gateway** (stateless WebSocket/HTTP server) and **Daemon** (background service manager for different OS platforms: systemd/launchd/Windows Task Scheduler).

---

## Gateway Architecture (`/src/gateway/`)

### Pattern: **Stateless Request-Response Broker**

**Key Design:**
- **Node.js HTTP + WebSocket server** listening on configurable port (default from config)
- Single instance per machine
- Manages multiple concurrent client connections (Set<GatewayWsClient>)
- Protocol: binary framing with request/response IDs for multiplexing
- WebSocket handshake includes auth negotiation (token/password/device signature)

**Router/Request Handling:**
- Method-based dispatch: `method` field in request routes to handler
- Connection-based handlers: each WebSocket gets isolated request context
- No routing table—all methods discovered at boot via `gatewayMethods[]` array
- Scopes system: each method has operator-level permission requirements (least-privilege model)

**WebSocket Connection Management** (`server/ws-connection.ts`):
```
1. Accept connection → sanitize headers
2. Rate-limit auth attempts (per-IP, per-device)
3. Handshake: exchange auth token/password/device signature
4. Attach message handler → bidirectional RPC loop
5. Broadcast presence events (other clients notified of joins/leaves)
6. On disconnect → cleanup client state, remove from presence
```

**Statefulness:**
- **Stateless**: no in-process request routing
- **Connection state only**: auth token, scopes, device ID tracked per WebSocket
- **Presence state** (transient): System-presence table updated on connect/disconnect (fire-and-forget to backend)
- **Health state** (volatile): version counters for presence/health events broadcast to clients

**Rate Limiting & DoS Protection:**
- Pre-auth connection budget (max connections before handshake completes)
- Per-IP auth brute-force limiter
- Payload size limits: 64KB max, 1MB max per message (configurable)
- Loopback address exempt from browser-origin fallback limits

---

## Daemon Architecture (`/src/daemon/`)

### Pattern: **Platform Abstraction Layer** (Bridge Pattern)

**Purpose:** Abstract OS-specific service registration so the gateway can be started automatically.

**Supported Runtimes:**
1. **systemd** (Linux) — user service via `~/.config/systemd/user/`
2. **launchd** (macOS) — user launch agent via `~/Library/LaunchAgents/`
3. **Windows Task Scheduler** — scheduled task auto-restart on login/system boot

**Service Lifecycle:**
```
stage()    → prepare unit file / plist / task XML (no write yet)
install()  → write to filesystem + register with OS (systemctl --user enable, launchctl load, etc.)
start()    → immediately run (systemctl --user start, launchctl start, schtasks /run)
restart()  → stop + start (or reschedule if needed)
stop()     → graceful shutdown
uninstall()→ deregister from OS + delete unit file
isLoaded() → check if registered with OS
readRuntime()→ query actual process state (PID, memory, CPU, uptime)
readCommand()→ extract ExecStart / ProgramArguments from stored config
```

**Configuration Persistence:**
- **ExecStart/ProgramArguments** stored in unit file (systemd) or plist (launchd)
- Environment variables passed inline (systemd Environment=) or via EnvironmentFile=
- Gateway entrypoint discovery: looks for `dist/index.js`, `dist/entry.mjs`, etc.

**Key Files:**
- `systemd.ts` — systemd user services
- `launchd.ts` — macOS launch agents (wrapper around launchctl)
- `schtasks.ts` — Windows scheduled tasks
- `service.ts` — unified interface (GatewayService)
- `node-service.ts` — sets env vars specific to Node.js runtime

**No Multi-Daemon Pool:**
- Single daemon per machine (single systemd service named `openclaw-gateway@*.service`)
- No load balancing between daemons
- Daemon restarts on crash via OS restart policy (systemd Restart=on-failure)

---

## Comparison: OpenClaw vs PM-OS Architecture

| Aspect | OpenClaw | PM-OS |
|--------|----------|-------|
| **Gateway Role** | Stateless WebSocket/HTTP broker | HTTP REST coordinator (pm-api) |
| **Routing** | Method-based dispatch, no explicit router | handler functions (`handlers_*.go`) by path |
| **Auth Model** | Token/password/device signature (in-connection) | API key in header + tenant extraction (per-request) |
| **State** | Per-connection (WebSocket), transient presence | Per-run in Supabase (RunStore), in-process maps (allOutputs) |
| **Multi-Executor** | N/A (gateway is pure broker) | Supports multiple executors (HTTPExecutor, PicoClawExecutor, FallbackExecutor) |
| **Service Mgmt** | Daemon abstraction (systemd/launchd/schtasks) | Systemd only (no abstraction layer) |
| **Persistence** | Minimal (presence only) | Full (Supabase PostgREST for all mutable state) |
| **Payload Handling** | Binary framing (size-bounded) | JSON + optional file uploads to GCS |
| **Concurrency Model** | Per-connection message handlers | Wave-based task parallelism (depends_on DAG) |

---

## 3 Architectural Patterns PM-OS Can Adopt

### 1. **Platform Abstraction for Service Management**
**Pattern:** Bridge pattern for OS-specific daemon registration (systemd/launchd/Windows)

**Current PM-OS:** Only systemd via direct system commands
**Benefit:** Enable pm-api/pm-engine to auto-start on Windows/macOS desktops without manual installation steps
**Implementation:** Create `pkg/infra/service-manager.go` interface with platform-specific implementations
**Overlap:** OpenClaw's `daemon/service.ts` → PM-OS `service.go` adapters (StartService, StopService, ReadStatus)

---

### 2. **Per-Connection Auth Binding with Presence Broadcasting**
**Pattern:** Device-identity scoping + presence state management for multi-client scenarios

**Current PM-OS:** Stateless per-request API key check; no notion of "client sessions"
**Benefit:** Support CLI tools (wave-cli, pm-agent) that maintain long-lived connections; broadcast run status to multiple watchers in real-time
**Implementation:** Add connection pool in pm-api (like OpenClaw's `Set<GatewayWsClient>`); upgrade WebSocket support with device-identity binding
**Overlap:** OpenClaw's device signature auth + presence events → PM-OS could add SSE/WebSocket subscriptions to `/api/v2/runs/{id}` for live updates

---

### 3. **Rate-Limiting Hierarchy: Pre-Auth + Per-Scope**
**Pattern:** Budget-based connection limiting before auth completes, then per-method scope enforcement

**Current PM-OS:** Quota middleware tracks per-tenant usage; no pre-auth budget or per-scope rate limits
**Benefit:** Mitigate brute-force attacks and handle bursty workloads gracefully (wave-cli dispatching 100+ tasks at once)
**Implementation:** Add pre-auth connection counter (like OpenClaw's `PreauthConnectionBudget`); extend quota middleware with scope-based limits (e.g., "plan" method = 10/min, "run" method = 50/min)
**Overlap:** OpenClaw's `AuthRateLimiter` + `PreauthConnectionBudget` → PM-OS `QuotaMiddleware` v2 with pre-auth gates

---

## Summary
**Overlap:** ~60% on auth/connection handling, 40% on persistence model
- Gateway = pure broker (OpenClaw advantage: cleaner state machine, easier to scale horizontally)
- Engine = orchestrator (PM-OS advantage: integrated quality gates, knowledge injection, git integration)
- **Integration point:** PM-OS could add a WebSocket gateway for real-time client subscriptions (wave-cli watching run output live), adopting OpenClaw's presence broadcasting + device-identity binding.
