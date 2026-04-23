# OpenClaw Chat & Channels Core Abstraction

**Research Date:** 2026-04-23  
**Scope:** Message routing, channel abstraction, thread model, multi-channel messaging

---

## Abstract Channel Interface Definition

OpenClaw uses **composition over inheritance** for channel implementations. The core abstraction is **`ChannelPlugin<TAccount>`** (in `src/channels/plugins/types.plugin.js`), which bundles:

### Base Plugin Structure

```typescript
// Minimal shape worth porting to Go
interface ChannelPlugin<TAccount> {
  id: ChatChannelId;           // "slack", "line", "telegram", etc.
  
  // Messaging pipeline
  messaging: ChannelMessagingAdapter;   // normalizeTarget, resolveInboundConversation
  
  // Outbound delivery
  outbound?: ChannelOutboundAdapter;    // sendPayload, sendReaction
  
  // Thread/conversation lifecycle
  conversationBindings?: {
    defaultTopLevelPlacement: "current" | "thread" | "reply";
  };
  
  // Inbound handling
  inbound?: ChannelInboundAdapter;      // parseInbound, validateInbound
  
  // Setup & configuration
  setup: ChannelSetupAdapter;           // interactiveSetup, validateConfig
  
  // Multi-channel routing
  bindings?: ChannelBindingsAdapter;    // resolveInboundConversation (peer → agent mapping)
  
  // Pairing (access control)
  pairing?: ChannelPairingAdapter;      // normalizeAllowEntry, notifyApproval
  
  // Security policies
  security?: ChannelSecurityAdapter;    // dmPolicy, groupPolicy, restrictSenders
  
  // Status & health
  status?: ChannelStatusAdapter;        // getAccountStatus, probe
}
```

### Message Model

```typescript
// Core inbound message - normalized across all channels
type InboundMessage = {
  channelId: ChatChannelId;
  accountId: string;
  
  // Routing context
  from: string;                         // sender ID (user, bot, etc.)
  target: string;                       // where msg came in (DM ID, channel ID, group ID)
  chatType: ChatType;                   // "direct" | "channel" | "group"
  
  // Content
  body: string;                         // normalized text (HTML stripped, markdown preserved)
  originalBody?: unknown;               // raw platform-specific payload
  
  // Thread awareness
  threadId?: string;                    // if reply to existing thread
  parentThreadId?: string;              // if nested reply
  
  // Metadata
  timestamp: number;                    // milliseconds
  isEdited?: boolean;
  isReaction?: boolean;
  reactionEmoji?: string;
  
  // Sender context
  senderLabel?: string;                 // display name
  isBot?: boolean;
  senderId?: string;
};

// Outbound message (replies sent back)
type OutboundMessage = {
  target: string;                       // recipient (user ID, channel ID)
  body: string;                         // formatted response
  
  // Thread routing
  threadId?: string;                    // if reply to thread
  mentions?: string[];                  // @user references
  
  // Channel-specific formatting
  format?: "text" | "markdown" | "html";
  richPayload?: unknown;                // platform-specific (LINE quick replies, Slack blocks, etc.)
  
  // Delivery options
  ephemeral?: boolean;                  // visible only to sender
  reactions?: string[];                 // emoji reactions to add
};
```

---

## Message Normalization & Routing

### Router: Incoming Message → Agent Dispatch

Location: `src/routing/resolve-route.ts` (815 LOC)

```typescript
// Core routing decision
type ResolveAgentRouteInput = {
  cfg: OpenClawConfig;
  channel: string;
  accountId?: string;
  peer?: RoutePeer;             // sender (user, bot)
  parentPeer?: RoutePeer;       // thread parent (inheritance)
  guildId?: string;             // Discord server context
  teamId?: string;              // Slack workspace context
  memberRoleIds?: string[];     // Role-based dispatch (Discord)
};

// Decision output
type ResolvedAgentRoute = {
  agentId: string;              // which agent handles this
  sessionKey: string;           // unique conversation session ID
  mainSessionKey: string;       // collapse multiple DMs to one main
  lastRoutePolicy: "main" | "session";  // where to store last route update
  matchedBy: "binding.peer" | "binding.peer.parent" | "binding.guild+roles" | "default";
};

// Route resolution (matching order, strict priority):
// 1. Exact peer binding (sender ID matches)
// 2. Parent peer binding (for threads, fallback if peer doesn't match)
// 3. Wildcard peer binding (peer kind matches, any ID)
// 4. Guild + Roles (Discord: server + user roles)
// 5. Guild only (Discord: any member of server)
// 6. Team (Slack: workspace)
// 7. Account-scoped fallback
// 8. Channel-wide fallback
// 9. Default agent
```

**Key innovation:** Thread parent inheritance. If a reply (child thread) doesn't have a direct binding, it inherits from the thread parent — preserving agent continuity in nested conversations.

---

## Thread & Conversation Model

Location: `src/channels/` + `src/infra/outbound/`

### Thread Ownership & Session Keys

```typescript
// Hierarchical thread binding
type ThreadBinding = {
  sessionKey: string;           // identifies this conversation branch
  channelId: string;
  accountId: string;
  agentId: string;
  
  // Thread metadata
  threadId?: string;            // platform thread ID
  parentSessionKey?: string;    // for nested replies
  boundAt: number;              // timestamp created
  lastActivityAt: number;       // last message time
  
  // Lifecycle
  idleTimeoutMs?: number;       // auto-close after silence
  maxAgeMs?: number;            // force-close after TTL
};

// DM scoping strategies
type DmScope = "main" | "per-peer" | "per-channel-peer" | "per-account-channel-peer";
// - "main": all DMs with user collapse to one session (small models benefit)
// - "per-peer": separate session per sender
// - "per-channel-peer": separate per channel + sender
// - "per-account-channel-peer": fully isolated (legacy, most complex)

// Session key building (deterministic)
sessionKey = buildAgentSessionKey({
  agentId: "claude",
  channel: "slack",
  accountId: "workspace-123",
  peer: { kind: "direct", id: "user-456" },
  dmScope: "per-peer"
}) // → "claude|slack|workspace-123|direct:user-456"
```

### Auto-Reply & Outbound Dispatch

Location: `src/auto-reply/envelope.ts`, `src/auto-reply/reply/`

```typescript
// Message envelope wrapping (metadata context)
function formatEnvelopeTimestamp(
  ts: number | Date,
  options?: { timezone?: "utc" | "local" | "user" | IANA string }
): string
// → "Mon 2026-04-23 15:30:45 UTC"

function formatInboundEnvelope(params: {
  channel: string;              // "slack", "line"
  from: string;                 // "@alice"
  body: string;                 // "What is 2+2?"
  timestamp?: number;
  chatType?: "direct" | "channel" | "group";
  senderLabel?: string;         // "Alice Smith"
  previousTimestamp?: number;   // for elapsed time
  envelope?: EnvelopeFormatOptions;
}): string
// → "[slack Alice @2026-04-23 15:30:45 UTC] What is 2+2?"

// Reply dispatcher chain (location: `src/auto-reply/reply/provider-dispatcher.types.ts`)
interface ReplyDispatcher {
  sendReply(message: OutboundMessage): Promise<SendResult>;
  sendReaction(emoji: string, onMessage: string): Promise<SendResult>;
}

// Auto-chunking for length limits (Slack 4K text block limit, LINE 5K, etc.)
type ChunkMode = "preserve" | "strip" | "markdown-safe";
function chunkText(text: string, maxChars: number, mode: ChunkMode): string[]
```

---

## Abstract Interface Worth Porting to Go

```go
// Minimal Channel abstraction for PM-OS
type Channel interface {
  // Routing
  ID() string                              // "slack", "telegram"
  ResolveInboundConversation(
    ctx context.Context,
    msg InboundMessage,
  ) (ConversationContext, error)           // map sender → agent
  
  // Messaging
  SendMessage(ctx context.Context, msg OutboundMessage) (SendResult, error)
  SendReaction(ctx context.Context, emoji string, onMessageID string) error
  NormalizeTarget(raw string) (string, error)  // sanitize IDs
  
  // Thread lifecycle
  GetOrCreateThread(ctx context.Context, params ThreadParams) (Thread, error)
  CloseThread(ctx context.Context, threadID string) error
}

// Minimal Message model
type InboundMessage struct {
  ChannelID    string    // "slack", "line"
  AccountID    string    // workspace/org context
  From         string    // sender ID
  Target       string    // DM/channel/group ID
  ChatType     string    // "direct", "channel", "group"
  Body         string    // normalized text
  Timestamp    int64     // milliseconds
  ThreadID     string    // optional parent thread
}

type OutboundMessage struct {
  Target       string    // recipient
  Body         string    // formatted response
  ThreadID     string    // if reply to thread
  RichPayload  json.RawMessage  // channel-specific (blocks, buttons)
}

type ConversationContext struct {
  AgentID      string
  SessionKey   string
  MatchedBy    string  // "binding.peer", "default", etc.
}
```

---

## Key Takeaways for PM-OS Integration

1. **Routing is hierarchical, not flat**
   - Thread parents can "sponsor" child bindings (inheritance)
   - Discord roles enable guild-level routing
   - Fallback chain ensures no unmatched message

2. **Message normalization is channel-agnostic**
   - All inbound → `InboundMessage` (text + metadata)
   - Outbound payload can be channel-specific (rich UI)
   - Envelope wraps agent context (timestamps, sender labels)

3. **Thread model is multi-tenant aware**
   - Session keys are deterministic (for idempotency)
   - DM scoping strategies degrade gracefully
   - Idle timeout + max age enable auto-cleanup

4. **Plugin SDK is minimal (no framework coupling)**
   - No web framework; only HTTP types
   - Adapters are small, composable functions
   - Plugins own their setup UI + runtime behavior

---

**Size Estimate (Go equivalent):** ~400 LOC core + ~800 LOC adapters per channel (Slack, Telegram, etc.)
