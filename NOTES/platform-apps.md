# OpenClaw Platform Client Architecture

## Overview
OpenClaw uses **native cross-platform clients** (Swift for iOS/macOS, Kotlin for Android) rather than a single framework like Flutter/React Native. Shared logic lives in a Swift package (`OpenClawKit`) accessible to iOS and macOS. Total client codebase: ~1,600 LOC Swift (iOS core) + ~140 LOC Kotlin (Android).

## Tech Stack

| Platform | Language | Framework | Size |
|----------|----------|-----------|------|
| iOS | Swift | SwiftUI + Combine | 1.6K LOC |
| macOS | Swift | SwiftUI + Cocoa | 300 LOC |
| macOS (TTS) | Swift | Speech synthesis | Minimal |
| Android | Kotlin | Jetpack Compose | 140 LOC |
| **Shared** | Swift | SPM package (OpenClawKit) | 4.7K LOC |

## Communication Protocol

**Protocol:** Custom binary/JSON over WebSocket + HTTP  
**Version:** Gateway Protocol v3 (versioned, allows evolution)  
**Connection Model:** WebSocket for bidirectional push; HTTP fallback for polling

### Message Frames
```swift
// Request
RequestFrame(type: "request", id: UUID, method: String, params: AnyCodable?)

// Response  
ResponseFrame(type: "response", id: UUID, ok: Bool, payload: AnyCodable?)

// Handshake
ConnectParams: minProtocol, maxProtocol, client metadata, capabilities, commands, permissions
HelloOk: server info, features, snapshot, canvas host URL, auth, policy
```

**Key insight:** Protocol is **version-negotiated** (minProtocol/maxProtocol) — clients can gracefully downgrade or reject incompatible servers.

## Push Notifications

**iOS/macOS:** Apple Push Notification service (APNs)
- Device token registered at connection time
- Silent pushes for background wakeups (background refresh task)
- Rich notifications with actions (watch prompts, execution approvals)
- Deep linking via URL schemes

**Android:** Firebase Cloud Messaging (FCM) equivalent (not visible in shared code — likely in native services)

**Notification Types:**
1. **Watch Prompts:** multi-action notifications with custom action handlers
2. **Exec Approval Requests:** approval notifications with inline response buttons
3. **Silent/Background:** triggers background refresh (15–90 sec retry)

## Authentication & Session Model

**Session Key:** Persistent session identifier stored in `KeychainStore` (iOS) / `SecurePrefs` (Android)
- One session per connected gateway
- Enables seamless reconnect on app restart

**Auth Flow:**
1. Device discovers gateway (mDNS or manual URL)
2. Optional trust prompt (SHA256 fingerprint verification)
3. ConnectParams sent with device metadata, scopes, permissions
4. Server responds with HelloOk + auth policy
5. Session key stored locally for future connections

**No OAuth/JWT visible** — appears to be session-based or certificate-pinned per gateway.

## Offline/Sync Strategy

**Online-first with graceful degradation:**
- Live WebSocket while app is foreground
- Silent push wakes in background (every 15–90 sec if server has pending data)
- Pending actions queued locally until connection restored
- Watch/approval actions mirrored to notification center if offline
- No explicit sync queue visible; push-driven

**LocalStorage:**
- Gateway settings: `GatewaySettingsStore` (domain-specific configuration)
- Keychain: session keys, auth tokens
- UserDefaults: debug flags, UI state
- No seen database for message dedup; likely handled server-side

## Client-Side vs Server-Side Logic Split

**Client (60%):**
- UI rendering (SwiftUI/Compose)
- Voice wake detection + audio capture
- Local gesture recognition
- Notification handling and routing
- Gateway discovery + trust validation
- Session persistence
- Speech-to-text + TTS formatting

**Server (40%):**
- LLM invocation
- Canvas state management
- Execution approval logic
- Multi-client sync (watch, phone, Mac)
- Push notification dispatch
- Session validation

**~30% shared protocol/models** (OpenClawKit) — validation, codecs, type definitions.

## Architecture Insights for PM-OS SaaS

### Key Learnings
1. **Native > cross-platform:** Platform-specific code is <200 LOC per app. Shared lib handles 90% of domain logic.
2. **Protocol versioning:** Designed for evolution; clients and servers can be deployed independently.
3. **Push-driven architecture:** Clients are mostly dormant; server initiates state changes. Reduces battery/bandwidth.
4. **Local-first perception:** Offline actions queue; sync is best-effort, not blocking.
5. **Minimal auth:** No OAuth in client layer; trust is per-gateway + session-based. Suitable for local/private networks.

### PM-OS Recommendation
**Start with web-first (React + TypeScript), then native iOS (Swift) as a secondary client, NOT a parallel effort.**

- **Phase 1:** Web client (Next.js) + Supabase auth → covers 95% of orchestration use cases (runs, recipes, logs, webhooks)
- **Phase 2 (later):** iOS widget (SwiftUI) for run notifications + quick approvals (execution gates)
- **Phase 3 (optional):** Android if PM-OS becomes consumer app (likely unnecessary for B2B orchestration)

**Protocol spec:** Version your `/api/v2/run`, `/api/runs/{id}` endpoints the way OpenClaw versions gateway protocol. Allow clients to declare minVersion/maxVersion; reject incompatible clients with 426 Upgrade Required.

**Offline sync:** Not needed for PM-OS (runs are server-driven, async). Focus on webhook callbacks + real-time SSE for browser clients instead.

---
