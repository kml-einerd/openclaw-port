# OpenClaw Canvas + A2UI Analysis

## What IS a Canvas in OpenClaw?

Canvas = **Live, server-driven HTML rendering surface** on iOS/Android OpenClaw nodes.
- **Not a React component** — it's a native WebView/viewport on mobile
- **Not DOM manipulation** — it's HTML file serving with live-reload + action bridging
- **Stateless server, stateful client:** Server serves HTML + injects a message-bridge script; client (mobile OS) renders and reports user interactions back via native bridge

## How It Works (Request-Response Flow)

```
Agent → canvas.a2ui.pushJSONL(jsonl)
         ↓
OpenClaw Node (HTTP handler on mobile)
  1. Validates JSONL (A2UI v0.8 format)
  2. A2UI Renderer (Lit-based) parses JSON
  3. Renders components as HTML
  4. Injects live-reload + action bridge script
  5. Serves index.html to WebView
         ↓
Mobile WebView
  - Renders <Canvas Surface>
  - User taps button → postMessage() via iOS webkit bridge
    or Android openclawCanvasA2UIAction.postMessage()
         ↓
Agent receives user action event
  (async callback, not sync response)
```

## A2UI Protocol (Current: v0.8)

A2UI is a **declarative JSON UI description format** — agent sends data, client renders.

### Core Actions (JSONL, one JSON object per line):

| Action | Purpose |
|--------|---------|
| `surfaceUpdate` | Add/update components on canvas. IDs are flat references (e.g., "root", "text_1") |
| `beginRendering` | Signal: render tree is ready; point root to surface |
| `dataModelUpdate` | Update state values (for dynamic bindings) |
| `createSurface` | v0.9 only — create a new named surface (not supported in OpenClaw yet) |
| `deleteSurface` | v0.9 only — remove a surface |

### Example (from a2ui-jsonl.ts):

```json
{"surfaceUpdate": {"surfaceId": "main", "components": [
  {"id": "root", "component": {"Column": {"children": {"explicitList": ["text"]}}}},
  {"id": "text", "component": {"Text": {"text": {"literalString": "Hello World"}, "usageHint": "body"}}}
]}}
{"beginRendering": {"surfaceId": "main", "root": "root"}}
```

### Components Available (from spec v0.8):

- **Layout:** Column, Row, Grid, Stack, Spacer, Padding
- **Input:** TextField, Select, Checkbox, Slider, DatePicker, TimePicker
- **Display:** Text, Image, Card, Badge, Loading
- **Action:** Button (sends action events back to agent)

## User Action Bridge (Cross-Platform)

**JavaScript injected into HTML by server:**

```ts
// src/canvas-host/a2ui.ts injectCanvasLiveReload()

// iOS
window.webkit.messageHandlers.openclawCanvasA2UIAction.postMessage({
  userAction: { name, surfaceId, sourceComponentId, context }
})

// Android
window.openclawCanvasA2UIAction.postMessage(JSON.stringify({
  userAction: { name, surfaceId, sourceComponentId, context }
}))

// Helper
window.openclawSendUserAction({ name, ... }) → posts to bridge
```

**Agent receives async event** (not a sync RPC response).

## Real-Time Updates (Live Reload)

**WebSocket + Chokidar file watcher**

- Server watches `rootDir/index.html` for changes
- On file change → debounced WebSocket broadcast "reload"
- Client receives "reload" → `location.reload()`
- Used for dev/testing, not production user flows

```ts
// server.ts
const wss = new WebSocketServer({ noServer: true });
chokidar.watch(rootDir, ...).on('change', () => scheduleReload());
// Browser receives "reload" msg → reloads
```

## File Structure (Canvas Host)

```
src/canvas-host/
├── a2ui.ts              # HTTP handler + asset server + live-reload injector
├── a2ui/                # A2UI renderer bundle (Lit-based, pre-built)
│   ├── index.html       # Canvas entry point
│   ├── a2ui.bundle.js   # Lit renderer + component catalog (~486KB minified)
│   └── .bundle.hash     # Cache buster
├── server.ts            # Full HTTP server + WebSocket upgrade handler
├── file-resolver.ts     # Safe file path resolution (no traversal attacks)
└── server.state-dir.test.ts # Tests

scripts/
├── canvas-a2ui-copy.ts  # Copy built A2UI assets to dist/
└── bundle-a2ui.mjs      # Build A2UI bundle (runs `a2ui build` → outputs index.html + a2ui.bundle.js)
```

## Integration Points in OpenClaw

1. **Canvas Host Server** (`cmd/canvas-host` in OpenClaw binary)
   - Listens on `__openclaw__/canvas/*` routes
   - Serves HTML + A2UI assets
   - Upgrades to WebSocket at `__openclaw__/ws` for live-reload

2. **Node CLI** (`src/cli/nodes-cli/register.canvas.ts`)
   - Commands: `canvas snapshot`, `canvas present`, `canvas navigate`, `canvas eval`, `canvas a2ui push`
   - `canvas a2ui push --text "hello"` → generates minimal A2UI JSONL → sends to node
   - `canvas a2ui push --jsonl file.jsonl` → validates + sends

3. **Chat Renderer** (`src/chat/canvas-render.ts`)
   - Chat message can contain `kind: "canvas"` object
   - Renders `<surface>` on target ("assistant_message" default)
   - Links to `/__openclaw__/canvas/documents/{encoded}/index.html`

## Is This Relevant for PM-OS Panel/Interactive Steps?

**Assessment: NOT directly copy-worthy, but concept is borrowable.**

### Why NOT Direct Copy:

1. **Platform Gap:** A2UI is iOS/Android native bridge protocol. PM-OS runs on Cloud Run (stateless HTTP), not on mobile nodes.
2. **Architecture Mismatch:** 
   - A2UI assumes persistent WebView per user session
   - PM-OS workers are ephemeral, per-task, fire-and-forget
   - PM-OS doesn't have a "canvas host" service running 24/7
3. **Complexity Overkill:** A2UI requires Lit renderer bundle (486KB), WebSocket live-reload, native bridge plumbing. PM-OS "panel" steps are simpler: form submission → JSON input/output.

### Concept Worth Borrowing:

1. **Declarative Component Model:** Instead of free-text HTML, define forms as JSON schema:
   ```json
   {
     "type": "form",
     "fields": [
       { "id": "name", "label": "Name", "type": "text" },
       { "id": "agree", "label": "Agree?", "type": "checkbox" }
     ]
   }
   ```
   Then client renders with its own component library (React, Vue, Flutter, etc.).

2. **User Action Events:** Capture form submission as structured event:
   ```json
   {
     "userAction": {
       "step_id": "s1",
       "action": "submit",
       "payload": { "name": "Alice", "agree": true }
     }
   }
   ```

3. **Safe-by-Default:** Declare components in catalog upfront, LLM can only reference them.

### Better Fit for PM-OS:

Instead of cloning A2UI infrastructure:
- **Panel step type** continues to be simple: `{ "type": "panel", "form": {...}, "on_submit": "continue_to_next_step" }`
- Form is rendered by **client** (browser, CLI, SDK) using its own components
- User submits → PM-OS receives structured JSON
- Agent continues execution with that data

**No WebView, no bundle, no live-reload needed.** PM-OS already has this pattern via `clarify` step type (interactive prompt).

## Summary Table

| Aspect | A2UI | PM-OS Panel |
|--------|------|-----------|
| **Target** | Mobile agents (iOS/Android) | Cloud orchestration (HTTP API) |
| **Transport** | WebView + native bridge | HTTP request/response or SSE |
| **Lifecycle** | Persistent session | Per-task, stateless |
| **Component Catalog** | Lit renderer bundle (486KB) | JSON schema (KBs) |
| **Real-time Updates** | WebSocket + file watcher | SSE or callback |
| **User Interaction** | Native bridge (webkit, Android) | HTTP POST or form submission |
| **Complexity** | High (framework, renderers, spec) | Low (already exists) |

## Recommendation

**Skip A2UI code copy.** Borrow the **philosophy** (declarative, schema-driven, safe-by-default), but keep PM-OS panel/interactive steps lightweight and HTTP-native. A2UI is production-grade for mobile, but over-engineered for stateless cloud tasks.
