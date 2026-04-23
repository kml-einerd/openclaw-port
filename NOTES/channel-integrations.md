# OpenClaw Channel Integrations Architecture

## Executive Summary

OpenClaw implements a mature, composable channel architecture across 24+ messaging platforms. PM-OS should adopt patterns from their `ChannelPlugin` abstraction, webhook/polling dual-mode inbound, and adapter pattern for outbound delivery. Telegram, Discord, and Slack are highest-value targets; each requires 1-2 days to port to Go. Recommend native Go implementation for core adapters + optional TypeScript sidecar for heavy computation.

---

## 1. Abstract Channel Interface Design

### Core Abstraction: `ChannelPlugin`

OpenClaw defines channels via a compositional **adapter pattern**, not a single interface. Key components:

```typescript
// Pseudo-contract (from types.adapters.ts + types.core.ts)
type ChannelPlugin = {
  // Setup lifecycle
  setup?: ChannelSetupAdapter;           // Onboard + validate credentials
  config?: ChannelConfigAdapter;         // Resolve accounts, list IDs, check enabled
  
  // Inbound message routing
  gateway?: ChannelGatewayAdapter;       // HTTP webhook dispatcher
  messaging?: ChannelMessagingAdapter;   // Parse inbound message
  
  // Outbound delivery
  outbound?: ChannelOutboundAdapter;     // Send message to channel
  
  // Auth + security
  auth?: ChannelAuthAdapter;             // Token validation
  security?: ChannelSecurityAdapter;     // DM policies, allowlist
  
  // Optional capabilities
  approval?: ChannelApprovalAdapter;     // Approval button handler
  action?: ChannelMessageActionAdapter;  // Message actions (reply, edit, etc)
  threading?: ChannelThreadingAdapter;   // Thread/topic handling
  
  // Agent tool integration
  agentTools?: ChannelAgentToolFactory;  // Tools exposed to agent
  
  // Lifecycle hooks
  heartbeat?: ChannelHeartbeatAdapter;   // Health check, poll updates
  lifecycle?: ChannelLifecycleAdapter;   // Init/shutdown
};
```

**Worth copying verbatim for PM-OS**: Yes, but simplified to ~8 core adapters (drop approval, agent-tools for v1).

### Key Design Principles

1. **Adapter Pattern**: Each responsibility (outbound, inbound parsing, auth) is a separate adapter. Reduces coupling.
2. **Optional Capabilities**: Not all channels support threads, reactions, or message editing. Adapters are optional.
3. **Account-Scoped**: Each channel can have multiple accounts (bots) configured. Adapters operate on `accountId`.
4. **Config-Driven**: Runtime behavior determined by `OpenClawConfig` (YAML/JSON), not code.

---

## 2. Message Model

OpenClaw doesn't define a single "Message" struct. Instead, channels define their native models and OpenClaw normalizes via context objects:

### Inbound Context (Channel → Core)
```typescript
type ChannelMessageActionContext = {
  // Routing
  channelId: string;           // "telegram", "discord", etc
  accountId: string;           // Which bot account
  senderId: string;            // Unique user ID in channel
  messageId: string | number;  // Native message ID (Telegram: int, Discord: snowflake)
  threadId?: string | number;  // Optional: Thread/topic ID
  
  // Content
  text: string;                // Normalized text content
  mediaUrls?: string[];        // Images, files, etc
  
  // Metadata
  timestamp?: Date;
  isEdited?: boolean;
  isReply?: boolean;
  
  // Execution context
  cfg: OpenClawConfig;         // Full config for auth lookup
  runtime: RuntimeEnv;         // Credentials, logger
};
```

### Outbound Hints (Core → Channel)
```typescript
type ChannelOutboundPayloadHint = {
  format?: "text" | "markdown" | "html";  // Markup support
  maxLength?: number;                       // Platform limit
  supportsThreadReply?: boolean;            // Can reply to specific message
  supportsReactions?: boolean;              // Can add emoji reactions
  mediaCapabilities?: string[];             // "image", "video", "file", "voice"
};
```

**Key insight**: OpenClaw normalizes at the **context/hint level**, not a shared message struct. Each channel keeps its native model (Grammy's Update, Discord.js Message, etc).

---

## 3. Inbound: Webhooks vs Polling

OpenClaw supports **both modes per channel**:

### Telegram Example (from `webhook.ts`)
- **Webhook mode** (preferred): HTTP server listens for `POST /telegram-webhook`, verifies HMAC secret, parses JSON update
  - Uses Grammy's `webhookCallback()` wrapper
  - Rate limiter: fixed-window (configurable)
  - Timeout: 5 seconds (returns 200 to Telegram before full processing)
  
- **Polling mode** (fallback): Continuous `getUpdates()` loop
  - Respects Telegram's 30-second timeout
  - Implements offset tracking to avoid re-processing

**Configuration-driven**:
```typescript
// In account config
useWebhook: boolean;
webhookUrl: string;           // e.g., https://myhost.com/telegram-webhook
webhookSecret: string;        // HMAC key
webhookPath: string;          // HTTP path
webhookPort: number;          // Optional: override default
```

### Discord Example
- **Webhook mode**: Interaction endpoints with signature verification
- **Polling mode**: None (requires WebSocket or polling via API)

### Slack Example
- **Webhook mode only**: Slash commands, Events API via POST

**Pattern for PM-OS**: Build inbound as HTTP router with pluggable parsers. Webhook is primary; polling optional for simplicity v1.

---

## 4. Outbound: Rate Limits, Retries, Delivery Confirmation

### Retry Strategy (Telegram Example)

```typescript
// From send.test.ts
{
  retry: {
    attempts: 2 | 3,           // Total attempts (1-based: 1 = no retry)
    minDelayMs: number;        // Min backoff
    maxDelayMs: number;        // Max backoff
    jitter: 0 | 1              // Add randomness
  }
}

// Honors Telegram's retry_after header
if (error.parameters?.retry_after) {
  waitMs = error.parameters.retry_after * 1000;  // Platform-specified delay
}
```

### Outbound Adapter Pattern

```typescript
type ChannelOutboundAdapter = {
  send: (params: {
    ctx: ChannelOutboundContext;
    message: string;
    mediaUrls?: string[];
    threadId?: string | number;
    requestId?: string;  // For idempotency
  }) => Promise<{
    messageId: string | number;
    timestamp: Date;
    platform?: string;  // For multi-protocol channels
  }>;
};
```

**Key observations**:
- **No native delivery confirmation** — adapters return `messageId` immediately
- **Idempotency via requestId**: Required for retry safety
- **Platform-specific rate handling**: Each adapter respects its platform's limits
- **Errors include retry_after hints**: Adapter uses to determine backoff

### Media Handling

OpenClaw normalizes media as **URLs**, not files:
```typescript
mediaUrls: string[];  // HTTP(S) URLs

// Channel adapter fetches + converts format
// Telegram: Download, convert to native format
// Discord: POST to CDN if needed
// Slack: Post as attachment
```

---

## 5. Threading/Replies

### Common Models Across Channels

| Channel | Model | PM-OS Mapping |
|---------|-------|---------------|
| Telegram | `message_thread_id` (int) | threadId: int |
| Discord | `thread_id` (snowflake), `channel_id` | threadId: string |
| Slack | `thread_ts` (timestamp string) | threadId: string |
| Matrix | Room ID + event ID | threadId: string (composit) |
| WhatsApp | No native threading | threadId: null |

OpenClaw stores `ChannelThreadingContext`:
```typescript
type ChannelThreadingContext = {
  threadId?: string | number;
  replyToMessageId?: string | number;
  threadTitle?: string;          // For channels that support naming
  isBroadcast?: boolean;         // Telegram channels don't support threads
};
```

**Adapter responsibility**: Normalize native thread models to this structure.

---

## 6. User Identity Mapping

OpenClaw maps channel users to internal users via:

```typescript
type ChannelAccountSnapshot = {
  channel: string;
  accountId: string;  // Which bot/account
  userId?: string;    // Channel-native user ID (Telegram: int, Discord: snowflake)
  username?: string;  // Display name
};

// In ChannelConfigAdapter
inspectAccount?: (cfg, accountId?) => unknown;  // Returns channel-specific user model
```

**Pattern**: Each channel adapter maintains a mapping file or in-memory cache.

---

## 7. Per-Tenant Credentials

OpenClaw uses **`ChannelSetupAdapter`** for credential onboarding:

```typescript
type ChannelSetupAdapter = {
  // Resolve which account to use (or create new)
  resolveAccountId?: (params: {
    cfg: OpenClawConfig;
    accountId?: string;
    input?: ChannelSetupInput;  // User-provided: token, auth code, etc
  }) => string;
  
  // Validate + apply config
  applyAccountConfig: (params: {
    cfg: OpenClawConfig;
    accountId: string;
    input: ChannelSetupInput;
  }) => OpenClawConfig;
  
  // Verify after writing config
  validateInput?: (params: {
    cfg: OpenClawConfig;
    accountId: string;
    input: ChannelSetupInput;
  }) => string | null;  // Error message if invalid
};

// Generic setup input bag
type ChannelSetupInput = {
  token?: string;
  botToken?: string;
  appToken?: string;
  webhookUrl?: string;
  // ... platform-specific fields
};
```

**Key insight**: Credentials stored in config YAML/JSON, not in code or env vars. Setup adapters validate before persisting.

---

## 8. Top 5 Channels for PM-OS

Ranked by: adoption in Brazil, Dis's ops, maturity in OpenClaw, implementation effort.

| # | Channel | Why | Maturity | Days to Port |
|---|---------|-----|----------|-------------|
| 1 | **Telegram** | 100% adoption in Brazil, bot-friendly, native typing, thread support, rich UI (keyboards) | 9/10 | 1-2 |
| 2 | **WhatsApp** | Growing B2B adoption, Dis's likely use case | 7/10 (requires official API) | 2-3 |
| 3 | **Discord** | Dev teams, automation-rich | 9/10 | 1-2 |
| 4 | **Slack** | Enterprise, approval workflows | 9/10 | 1-2 |
| 5 | **Matrix** | Open-source alternative, self-hostable | 6/10 (fewer enterprises) | 2-3 |

**Phasing recommendation**:
- **Week 1-2**: Telegram (highest ROI for Brazil ops)
- **Week 3**: WhatsApp or Discord (parallel work)
- **Week 4+**: Slack, Matrix as needed

---

## 9. Architectural Patterns

### Webhook Router Pattern (OpenClaw)

```typescript
// HTTP server with pluggable handlers per channel
POST /webhooks/:channelId → dispatcher
  → telegram-handler (verify secret, parse Update)
  → discord-handler (verify signature, parse Interaction)
  → slack-handler (verify X-Slack-Signature, parse Event)

// Each handler:
1. Verify request authenticity (HMAC/signature)
2. Parse native payload
3. Normalize to ChannelMessageActionContext
4. Dispatch to engine
5. Return 200 immediately (async processing)
```

### Inbound Queue Pattern

OpenClaw processes inbound async (webhook returns 200 immediately). For PM-OS with recipe engine:

```
Webhook receives message
  → Publish to Pub/Sub (Google Cloud Pub/Sub or in-process channel)
  → HTTP returns 200 immediately
  
Pub/Sub subscriber:
  → Lookup matching recipes by channel + trigger
  → RunRecipe(recipe, params={channel, userId, threadId, text, ...})
  → Store run metadata back to Supabase
```

This decouples webhook latency from recipe execution.

---

## 10. Effort Estimates per Channel (Go Port)

Assuming Go HTTP server + Supabase PostgREST only (no gRPC, no internal SDKs):

| Task | Telegram | Discord | Slack | WhatsApp |
|------|----------|---------|-------|----------|
| HTTP webhook handler | 0.5d | 0.5d | 0.5d | 0.5d |
| Message parser (inbound) | 0.5d | 0.5d | 0.5d | 1d |
| Outbound adapter (send) | 0.5d | 0.5d | 0.5d | 1d |
| Auth + credentials | 0.25d | 0.25d | 0.25d | 0.5d |
| Threading/context mapping | 0.25d | 0.25d | 0.25d | 0.25d |
| Rate limit + retry logic | 0.25d | 0.25d | 0.25d | 0.25d |
| Tests | 0.5d | 0.5d | 0.5d | 0.5d |
| **Total** | **3d** | **3d** | **3d** | **4.5d** |

**Notes**:
- Assumes reusable HTTP utilities (auth verification, rate-limiting middleware)
- Telegram simplest: official Bot API, no OAuth complexity
- Discord/Slack: signature verification + OAuth token flow adds 0.5d each
- WhatsApp: Official Cloud API required (no self-hosted option); OAuth + webhook signing

---

## 11. Build Channels Natively in Go or Fork OpenClaw as TS Sidecar?

### Option A: Native Go (Recommended)

**Pros**:
- Single binary, single deploy artifact
- Direct Supabase integration (no inter-process RPC)
- Simple debugging (single debug session)
- Better startup latency (~100ms vs ~500ms with sidecar)

**Cons**:
- Reimplements webhook parsing, OAuth, media handling from OpenClaw
- Maintenance burden for 24+ channels

**Effort**: 3 days per channel (as above). Total for top 5: 15 days.

### Option B: Fork OpenClaw Extensions as TS Sidecar

**Pros**:
- Reuse OpenClaw's battle-tested adapters
- Minimal reimplementation
- Can deploy independently

**Cons**:
- Inter-process communication overhead (gRPC, sidecar health checks)
- Credential passing complexity (sidecar ← PM-OS secrets)
- Double container size (Go + Node)
- Shared state (allOutputs, run metrics) must go through HTTP/gRPC

**Effort**: 2 days per channel. Total for top 5: 10 days. BUT +5 days for gRPC wiring + credential sync.

### Recommendation

**Go natively, but modularly**:
1. Extract OpenClaw's `ChannelPlugin` adapter interface to PM-OS as a Go interface
2. Implement top 3 channels (Telegram, Discord, Slack) in Go
3. Use OpenClaw's tests + payload examples as reference (don't copy-paste code; understand the contract)
4. If adoption requires 20+ channels, consider TypeScript sidecar for long tail

---

## 12. Specific Implementation Notes for PM-OS

### Webhook Registration Pattern

PM-OS bundle SaaS needs:
```go
// pkg/channels/inbound.go
type InboundRouter struct {
  handlers map[string]InboundHandler  // "telegram", "discord", etc
  store    *Store
  engine   *Engine
}

type InboundHandler interface {
  VerifySignature(body []byte, signature string) bool
  ParseMessage(body []byte) (*MessageContext, error)
}

// HTTP endpoint
POST /api/channels/webhook/:channelId/:accountId
  → router.Handle(channelId, body, signature)
  → handler.VerifySignature()
  → handler.ParseMessage()
  → route to matching bundles by trigger rule
  → engine.RunRecipe(bundleRecipe, params)
  → store.CreateRun()
  → return 202 immediately
```

### Rate Limiting Middleware

```go
// pkg/channels/rate_limiter.go
type ChannelRateLimiter interface {
  Wait(ctx context.Context, channelID string, accountID string) error
  // Returns platform-specific wait duration if rate-limited
}

// Per-account limits (Telegram: ~30 msg/sec, Discord: varies by tier)
limits := map[string]time.Duration{
  "telegram":  100 * time.Millisecond,
  "discord":   500 * time.Millisecond,
  "slack":     1000 * time.Millisecond,
}
```

### Secret Management

```go
// Channels stored in Supabase
type ChannelAccount struct {
  ID          string
  TenantID    string
  ChannelID   string  // "telegram", "discord"
  AccountID   string  // "bot-main", "bot-support"
  
  // Encrypted in DB, decrypted at runtime
  BotToken    string  // Telegram bot_token, Discord token
  WebhookURL  string  // Optional: webhook registration endpoint
  WebhookSecret string // HMAC secret
  
  IsEnabled   bool
  CreatedAt   time.Time
}
```

---

## Conclusion

OpenClaw's channel architecture is **production-grade**, designed for multi-tenant SaaS with strict separation of concerns. PM-OS should adopt:

1. **Adapter pattern** for each channel responsibility
2. **Webhook-first inbound** with optional polling fallback
3. **Message normalization via context**, not a shared struct
4. **Per-tenant credentials in config**, validated on setup
5. **Async inbound processing** (webhook returns 200, Pub/Sub processes recipe)

**Next steps**:
- Implement native Go HTTP handlers for Telegram, Discord, Slack (Week 1-2)
- Wire webhook router to engine (Week 2)
- Add bundle invocation trigger rules (Week 3)
- Beta with Dis's ops team (Week 4)

---

## References

- OpenClaw source: `/tmp/openclaw-eval/openclaw/src/channels/plugins/`
- Telegram extension: `/tmp/openclaw-eval/openclaw/extensions/telegram/src/`
- Discord extension: `/tmp/openclaw-eval/openclaw/extensions/discord/src/`
- Slack extension: `/tmp/openclaw-eval/openclaw/extensions/slack/src/`
