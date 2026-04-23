# OpenClaw LLM Provider Extensions: Multi-Provider Abstraction Pattern

## Executive Summary

OpenClaw implements a **plugin-based provider system** with standardized interfaces for LLM execution, cost tracking, and streaming. The architecture decouples provider-specific details from the core execution engine via:

1. **Common LLM Interface** (`StreamFn` + `ProviderRuntimeModel`)
2. **Streaming Wrapper Factories** for provider-specific features (thinking, tool use, etc.)
3. **Provider Registry** system with hot-swappable auth, models, and capabilities
4. **Normalized Tool/Function Handling** across OpenAI, Claude, Gemini
5. **Cost Tracking** via provider metadata and token accounting

---

## 1. Core LLM Interface Contract

### Request/Response Abstraction

```typescript
// From plugin-sdk/provider-stream-shared.ts
type StreamFn = (
  payload: ChatCompletionRequest,
  ...extras
) => Promise<Message> & AsyncIterable<StreamDelta>

interface ProviderRuntimeModel {
  id: string
  name: string
  api: "openai-completions" | "google-generative-ai" | "anthropic" | string
  provider: string
  baseUrl?: string
  input: ("text" | "image" | "audio" | "video")[]
  contextWindow: number
  maxTokens: number
  cost: { input: $/1K, output: $/1K, cacheRead?: $/1K, cacheWrite?: $/1K }
  reasoning?: boolean
}
```

**Key insight:** Providers normalize to **common payload types** (e.g., `openai-completions` API), not provider-native formats. Normalization happens at adapter layer.

---

## 2. Provider Registration Pattern

### Example: Anthropic vs OpenAI vs OpenRouter

**Anthropic (native):**
```typescript
// extensions/anthropic/register.runtime.ts
api.registerProvider({
  id: "anthropic",
  label: "Claude",
  auth: [createAnthropicApiKeyMethod(), createCliAuthMethod()],
  catalog: { models: [...] },
  wrapStreamFn: (ctx) => wrapAnthropicProviderStream(ctx.streamFn, ctx.extraParams)
})
```

**OpenAI (OpenAI-compatible API):**
```typescript
// extensions/openai/openai-provider.ts
const model = {
  api: "openai-completions", // Standard interface
  provider: "openai",
  baseUrl: "https://api.openai.com/v1",
  input: ["text", "image"],
  cost: { input: 2.5, output: 15 }
}
```

**OpenRouter (proxy):**
```typescript
// extensions/openrouter/index.ts
const model = {
  api: "openai-completions", // Proxy normalizes to OpenAI format
  provider: "openrouter",
  baseUrl: "https://openrouter.ai/api/v1",
  input: getCapabilities(modelId) // Dynamic per-model
}
```

**Key insight:** OpenRouter + Ollama **normalize to OpenAI API** (`openai-completions`), delegating transport to common HTTP client.

---

## 3. Streaming & Feature Normalization

### ProviderStreamWrapperFactory Pattern

All providers use **decorator pattern** for feature injection:

```typescript
type ProviderStreamWrapperFactory = 
  (streamFn: StreamFn | undefined) => StreamFn | undefined

// Compose multiple wrappers
function composeProviderStreamWrappers(
  baseStreamFn,
  ...wrappers: ProviderStreamWrapperFactory[]
) {
  return wrappers.reduce((fn, wrapper) => wrapper?.(fn), baseStreamFn)
}
```

### Provider-Specific Stream Wrappers

| Provider | Feature | Wrapper |
|----------|---------|---------|
| **Anthropic** | Cache control, batch streaming | `wrapAnthropicProviderStream` |
| **OpenAI** | Reasoning (o1/o3), service tier, fast mode | `createOpenAIThinkingLevelWrapper`, `createOpenAIServiceTierWrapper` |
| **Google/Gemini** | Extended thinking, budget tuning | `createGoogleThinkingPayloadWrapper` |
| **Moonshot** | Thinking (type + keep), long context | `createMoonshotThinkingWrapper` |
| **OpenRouter** | System cache (for Anthropic/Moonshot), reasoning proxy | `createOpenRouterSystemCacheWrapper` |
| **LiteLLM** | Proxy format, no custom features | Identity (none) |
| **Ollama** | Context window injection (`num_ctx`) | `wrapOllamaCompatNumCtx` |

**Normalization of extended thinking:**
- OpenAI: `reasoning_effort` param (preview/high)
- Google: `thinking_config.budget_tokens`
- Moonshot: `thinking` type enum
- All use same `thinkingLevel` context variable in wrapper

---

## 4. Tool/Function Use Normalization

### Cross-Provider Tool Format

```typescript
// Common normalization in stream handlers
export function createToolStreamWrapper(streamFn: StreamFn, enabled: boolean) {
  return async function* wrappedStream(request) {
    for await (const chunk of streamFn(request)) {
      // Normalize tool use across providers:
      // OpenAI: { type: 'function', function: { name, arguments } }
      // Anthropic: { type: 'tool_use', name, input }
      // Google: { type: 'function_call', name, args }
      yield normalizeToolBlock(chunk)
    }
  }
}

// Decode HTML entities in tool arguments (cross-platform quirk)
function decodeToolCallArgumentsHtmlEntitiesInMessage(message) {
  visitObjectContentBlocks(message, (block) => {
    if (block.type === 'toolCall') {
      block.arguments = decodeHtmlEntitiesInObject(block.arguments)
    }
  })
}
```

---

## 5. Cost Tracking Pattern

### Metadata-Based Cost Calculation

```typescript
interface ProviderRuntimeModel {
  cost: {
    input: number      // $/1K tokens
    output: number     // $/1K tokens
    cacheRead?: number // $/1K (prompt cache hit)
    cacheWrite?: number // $/1K (cache write)
  }
}

// Usage capture
function calculateCost(result: Message) {
  const { inputTokens, outputTokens, cacheTokens } = result.usage
  return (
    (inputTokens / 1000) * model.cost.input +
    (outputTokens / 1000) * model.cost.output +
    (cacheTokens / 1000) * (model.cost.cacheRead || 0)
  )
}
```

**Providers tracked:**
- **Anthropic:** `claude-opus-4-7` @ $3/$15, `claude-sonnet-4-6` @ $3/$15
- **OpenAI:** `gpt-5.4` @ $2.5/$15, `gpt-5.4-mini` @ $0.75/$4.5
- **Google Gemini:** `gemini-3-pro` @ $1.25/$5, `gemini-3-flash` @ $0.075/$0.3
- **OpenRouter:** Dynamic per model via catalog lookup
- **Ollama:** Free (local), $0 cost

---

## 6. Failover & Routing

### Multi-Provider Dispatch

```typescript
// From detect_changes_tool pattern
function shouldUseOpenAIResponsesTransport(params: {
  provider: string
  api?: string
  baseUrl?: string
}): boolean {
  const isOwnerProvider = normalizeProviderId(params.provider) === "openai"
  if (isOwnerProvider) {
    return !baseUrl || isOpenAIApiBaseUrl(baseUrl)
  }
  return isOpenAIApiBaseUrl(baseUrl)
}

// If provider A fails, try provider B (if configured)
// FallbackExecutor pattern in PM-OS currently does this
```

---

## 7. Authentication Pattern

### Provider-Agnostic Auth

```typescript
// All providers implement this
type ProviderAuthMethod = {
  methodId: string
  label: string
  envVar?: string
  promptMessage?: string
  wizard?: {
    choiceId: string
    choiceLabel: string
    groupId: string
    groupLabel: string
  }
}

// Examples:
// Anthropic: API key + Claude CLI token + setup token
// OpenAI: API key only
// Google: API key + Service account JSON
// OpenRouter: API key only
// Ollama: None (local) or HTTP auth
```

---

## 8. Catalog Management

### Dynamic Model Discovery

**Static (Anthropic, OpenAI):**
```typescript
catalog: {
  models: [
    { id: "claude-opus-4-7", name: "Claude Opus 4.7", context: 1048576 },
    { id: "gpt-5.4", name: "GPT-5.4", context: 1050000 }
  ]
}
```

**Dynamic (OpenRouter, Ollama):**
```typescript
catalog: {
  run: async (ctx) => {
    const response = await fetch(`${baseUrl}/models`)
    return response.json().map(m => ({
      id: m.id,
      name: m.name || m.id,
      contextWindow: m.context_length || DEFAULT_CONTEXT_TOKENS,
      cost: m.pricing || DEFAULT_COST
    }))
  }
}
```

---

## 9. Key Design Patterns

| Pattern | Purpose | Example |
|---------|---------|---------|
| **Plugin Registry** | Hot-swappable providers | `registerProvider(id, config)` |
| **Wrapper Chain** | Feature injection | `composeProviderStreamWrappers(...)` |
| **API Normalization** | Reduce provider variants | All normalize to `openai-completions` or `anthropic` |
| **Metadata-Driven** | Cost, capabilities, models | `ProviderRuntimeModel` | 
| **Context Propagation** | Thinking level, cache TTL, etc. | `ProviderWrapStreamFnContext` |

---

## 10. Porting to Go (PM-OS)

### Top 3 Providers to Support Post-Anthropic

1. **OpenAI (gpt-5.4)** — widest adoption, closest to Anthropic API
2. **Google Gemini (gemini-3-pro)** — thinking parity with Opus, fast flash alternative
3. **OpenRouter** — fallback proxy for outages, access to 100+ models

### Minimum Go Interfaces Needed

```go
// In pkg/engine/adapters/executor.go
type LLMProvider interface {
  Name() string                    // "anthropic", "openai", "gemini"
  IsAvailable(ctx) bool
  Execute(ctx, workItem) (*TaskResult, error)
  StreamExecute(ctx, workItem) (<-chan TaskEvent, error)
}

type ProviderModel struct {
  ID          string
  Name        string
  Provider    string
  API         string  // "anthropic", "openai-completions", "google-generative-ai"
  CostPerK    CostTier
  InputTypes  []string
  MaxTokens   int
}

type CostTier struct {
  InputPerK  float64
  OutputPerK float64
  CachePerK  float64
}
```

### HTTP Transport (No SDK)

OpenClaw's approach: **HTTP-only normalization**
- Anthropic → native format + OAuth
- OpenAI → POST `/v1/chat/completions` + Bearer token
- Gemini → POST with API key in query string
- OpenRouter → OpenAI-compatible HTTP wrapper

Advantage over SDK: **smaller binary, no dependency lock-in, streaming trivial via HTTP chunked**.

---

## Summary: LLM Interface Worth Porting

OpenClaw abstracts three layers:

1. **Authentication** — per-provider secrets + onboarding
2. **Request Normalization** — map WorkItem → provider API format
3. **Feature Wrapping** — inject thinking, cache, tool handling via decorators

This **scales to 100+ providers** because:
- Core abstraction is minimal (`StreamFn`)
- Features are composable (wrappers)
- Costs are metadata (no special handling)
- Models are **self-descriptive** (no hardcoding)

**For PM-OS v3:** Port Anthropic + OpenAI + Gemini as native executors, add OpenRouter as HTTP-only fallback, all sharing the same `Execute(ctx, workItem)` interface.
