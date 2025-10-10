# LLM Management Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           User Application Layer                             │
│                                                                              │
│   job := &LLM.Job{...}                                                      │
│   LLM.WorkerChannel <- job                                                  │
│   response := <-job.Result                                                  │
└───────────────────────────────┬─────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Orchestration Layer (LLM/)                           │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────┐       │
│  │  OrchestrationService (Interface)                               │       │
│  │    • Enqueue(job)                                               │       │
│  │    • Stop()                                                     │       │
│  └────────────────────────┬────────────────────────────────────────┘       │
│                           │                                                 │
│                           ▼                                                 │
│  ┌─────────────────────────────────────────────────────────────────┐       │
│  │  Orchestrator (Concrete Implementation)                         │       │
│  │                                                                  │       │
│  │    Components:                                                  │       │
│  │    • JobQueue          → FIFO queue with priority               │       │
│  │    • Worker Pool       → Concurrent workers (configurable)      │       │
│  │    • Rate Limiter      → Token bucket (requests & tokens)       │       │
│  │    • Backoff Manager   → Exponential backoff strategies         │       │
│  │    • Retry Handler     → Automatic retry logic                 │       │
│  │    • Error Classifier  → Categorizes errors                    │       │
│  │    • ClientAdapter     → Interface to providers                │       │
│  └────────────────────────┬────────────────────────────────────────┘       │
└────────────────────────────┼────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Client Adapter Layer (clientManager/)                    │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────┐      │
│  │  ClientAdapter Interface                                         │      │
│  │    Process(inputs) → (*openai.ChatCompletionResponse, error)     │      │
│  └────────┬─────────────────────────┬──────────────────┬────────────┘      │
│           │                         │                  │                    │
│           ▼                         ▼                  ▼                    │
│  ┌────────────────┐     ┌────────────────┐   ┌────────────────┐           │
│  │ LocalClient    │     │ OpenAIClient   │   │ GeminiClient   │           │
│  │ Adapter        │     │ Adapter        │   │ Adapter        │           │
│  │                │     │                │   │                │           │
│  │ • HTTP Client  │     │ • Native SDK   │   │ • Format       │           │
│  │ • OpenAI       │     │ • Direct API   │   │   Converter    │           │
│  │   compatible   │     │   calls        │   │ • HTTP Client  │           │
│  └────────┬───────┘     └────────┬───────┘   └────────┬───────┘           │
└───────────┼──────────────────────┼────────────────────┼────────────────────┘
            │                      │                    │
            │                      │                    │
┌───────────┼──────────────────────┼────────────────────┼────────────────────┐
│           │         Request Building Layer             │                    │
│           │                                            │                    │
│  ┌────────▼─────────────────────────────────────────────────────┐          │
│  │  RequestBuilder Interface                                    │          │
│  │    BuildRequest(inputs) → (ChatCompletionRequest, error)     │          │
│  └────────┬─────────────────────────────────────────────────────┘          │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐           │
│  │  DefaultOpenAIReqBuilder                                    │           │
│  │                                                              │           │
│  │    • Builds messages array                                 │           │
│  │    • Handles images (multi-content)                        │           │
│  │    • Applies temperature, top_p, seed                      │           │
│  │    • Reasoning model detection                             │           │
│  │    • Uses ModelConverter                                   │           │
│  └────────┬────────────────────────────────────────────────────┘           │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐           │
│  │  ModelConverter Interface                                   │           │
│  │    Convert(ModelType) → string                              │           │
│  └────────┬────────────────────────────────────────────────────┘           │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐           │
│  │  ProviderModelConverter                                     │           │
│  │                                                              │           │
│  │    jsonSchema.Gpt4Mini      → "gpt-4o-mini"                │           │
│  │    jsonSchema.GeminiFlash   → "gemini-2.5-flash-lite"      │           │
│  │    jsonSchema.ClaudeSonnet  → "claude-3-sonnet-..."        │           │
│  │    ModelType("custom")      → "custom" (pass-through)       │           │
│  └─────────────────────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         External Provider APIs                              │
│                                                                              │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐          │
│  │  Local/Custom    │  │  OpenAI API      │  │  Gemini API      │          │
│  │  Endpoints       │  │                  │  │                  │          │
│  │                  │  │  api.openai.com  │  │  generative      │          │
│  │  • Ollama        │  │                  │  │  language        │          │
│  │  • vLLM          │  │  /v1/chat/       │  │  .googleapis.com │          │
│  │  • LM Studio     │  │  completions     │  │                  │          │
│  │  • LocalAI       │  │                  │  │  /v1beta/models/ │          │
│  │  • Custom        │  │  (Native SDK)    │  │  :generateContent│          │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘          │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           Configuration Flow                                 │
│                                                                              │
│  Environment Variables (.env)                                               │
│  ┌────────────────────────────────────────────────────────────┐             │
│  │  LLM_PROVIDER=openai                                       │             │
│  │  LLM_API_KEY=sk-...                                        │             │
│  │  LLM_MAX_TOKENS_PER_MINUTE=90000                           │             │
│  │  LLM_MAX_REQUESTS_PER_MINUTE=3500                          │             │
│  │  LLM_BACKOFF_STRATEGY=per-worker                           │             │
│  └────────────────────────┬───────────────────────────────────┘             │
│                           │                                                 │
│                           ▼                                                 │
│  ┌────────────────────────────────────────────────────────────┐             │
│  │  Factory Pattern (clientManager/factory.go)                │             │
│  │                                                             │             │
│  │  NewClientAdapterFromEnv()                                 │             │
│  │    ↓                                                        │             │
│  │  Reads environment variables                               │             │
│  │    ↓                                                        │             │
│  │  Creates appropriate ClientAdapter                         │             │
│  │    ↓                                                        │             │
│  │  Returns configured adapter to Orchestrator                │             │
│  └─────────────────────────────────────────────────────────────┘            │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         Data Flow Example                                    │
│                                                                              │
│  1. User submits job                                                        │
│     ↓                                                                        │
│  2. Orchestrator enqueues job                                               │
│     ↓                                                                        │
│  3. Worker picks up job                                                     │
│     ↓                                                                        │
│  4. Apply backoff if needed                                                 │
│     ↓                                                                        │
│  5. Wait for rate limiters (requests & tokens)                              │
│     ↓                                                                        │
│  6. ModelConverter: Gpt4Mini → "gpt-4o-mini"                                │
│     ↓                                                                        │
│  7. RequestBuilder: Build OpenAI-format request                             │
│     {                                                                        │
│       "model": "gpt-4o-mini",                                               │
│       "messages": [...],                                                    │
│       "temperature": 0.7                                                    │
│     }                                                                        │
│     ↓                                                                        │
│  8. ClientAdapter processes request                                         │
│     • Local: Send as-is via HTTP                                            │
│     • OpenAI: Use native SDK                                                │
│     • Gemini: Convert to Gemini format, send, convert back                  │
│     ↓                                                                        │
│  9. Receive OpenAI-format response                                          │
│     {                                                                        │
│       "choices": [{                                                         │
│         "message": {"content": "Paris is the capital of France."}           │
│       }],                                                                   │
│       "usage": {...}                                                        │
│     }                                                                        │
│     ↓                                                                        │
│  10. On success: Send to job.Result channel                                 │
│      On error: Classify, retry or fail                                      │
│     ↓                                                                        │
│  11. User receives response                                                 │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         Error Handling Flow                                  │
│                                                                              │
│  Error occurs during API call                                               │
│     ↓                                                                        │
│  ErrorClassifier categorizes error                                          │
│     ↓                                                                        │
│  ┌──────────────────┬─────────────────────┬─────────────────────┐          │
│  │   Rate Limit     │   Transient Error   │   Permanent Error   │          │
│  │   (429, etc.)    │   (500, 502, etc.)  │   (400, 401, etc.)  │          │
│  └────────┬─────────┴──────────┬──────────┴──────────┬──────────┘          │
│           │                    │                      │                     │
│           ▼                    ▼                      ▼                     │
│  ┌────────────────┐  ┌─────────────────┐  ┌──────────────────┐            │
│  │ Activate       │  │ Increment retry │  │ Fail job         │            │
│  │ Backoff        │  │ counter         │  │ immediately      │            │
│  │                │  │                 │  │                  │            │
│  │ Requeue job    │  │ If < maxRetries │  │ Send error to    │            │
│  │                │  │   → Backoff     │  │ job.Error        │            │
│  │ Worker pauses  │  │   → Requeue     │  │ channel          │            │
│  │ based on       │  │ Else            │  │                  │            │
│  │ strategy:      │  │   → Fail job    │  │                  │            │
│  │ • None         │  │                 │  │                  │            │
│  │ • Global       │  │                 │  │                  │            │
│  │ • Per-worker   │  │                 │  │                  │            │
│  └────────────────┘  └─────────────────┘  └──────────────────┘            │
└─────────────────────────────────────────────────────────────────────────────┘
```
