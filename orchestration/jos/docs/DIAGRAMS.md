# JOS Component Relationships Diagram

This document contains ASCII diagrams showing how components relate to each other.

## Complete System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           SERVICE LAYER                                  │
│                                                                          │
│  ┌──────────────────┐              ┌──────────────────┐                │
│  │  HTTP Service    │              │  gRPC Service    │                │
│  │ objectGen.go     │              │ generateV2.go    │                │
│  └────────┬─────────┘              └────────┬─────────┘                │
└───────────┼──────────────────────────────────┼──────────────────────────┘
            │                                  │
            └──────────────┬───────────────────┘
                           │
                           │ Creates Factory
                           ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         FACTORY LAYER                                    │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────┐        │
│  │            GeneratorFactory (generator_factory.go)          │        │
│  │                                                              │        │
│  │  Create() method orchestrates component creation:           │        │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐           │        │
│  │  │ Analyzer   │  │ Executor   │  │ Assembler  │           │        │
│  │  └────────────┘  └────────────┘  └────────────┘           │        │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐           │        │
│  │  │ Strategy   │  │    LLM     │  │  Prompt    │           │        │
│  │  └────────────┘  └────────────┘  └────────────┘           │        │
│  └────────────────────────────┬─────────────────────────────┘        │
└─────────────────────────────────┼────────────────────────────────────┘
                                  │
                                  │ Injects into
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      APPLICATION LAYER                                   │
│                                                                          │
│  ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐  │
│  │ DefaultGenerator │   │StreamingGenerator│   │ProgressiveGen   │  │
│  │                  │   │                  │   │                  │  │
│  │ Sync execution   │   │ Field streaming  │   │ Token streaming  │  │
│  └──────────────────┘   └──────────────────┘   └──────────────────┘  │
│           │                      │                       │             │
│           └──────────────────────┼───────────────────────┘             │
│                                  │                                     │
│              All implement Generator interface                         │
└──────────────────────────────────┼─────────────────────────────────────┘
                                   │
                                   │ Depends on
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        DOMAIN LAYER                                      │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────┐        │
│  │                     interfaces.go                           │        │
│  │                                                              │        │
│  │  • Generator           • TaskExecutor    • LLMProvider      │        │
│  │  • SchemaAnalyzer      • TypeProcessor   • PromptBuilder    │        │
│  │  • ExecutionStrategy   • ResultAssembler • + Extensions     │        │
│  └────────────────────────────────────────────────────────────┘        │
│  ┌────────────────────────────────────────────────────────────┐        │
│  │                       models.go                             │        │
│  │                                                              │        │
│  │  • GenerationRequest   • ExecutionContext                   │        │
│  │  • GenerationResult    • FieldTask                          │        │
│  │  • FieldDefinition     • TaskResult                         │        │
│  └────────────────────────────────────────────────────────────┘        │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │
                                   │ Implemented by
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    INFRASTRUCTURE LAYER                                  │
│                                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                 │
│  │  Analysis    │  │  Execution   │  │   Assembly   │                 │
│  │              │  │              │  │              │                 │
│  │ Schema       │  │ Task         │  │ Result       │                 │
│  │ Analyzer     │  │ Executor     │  │ Assemblers   │                 │
│  │              │  │              │  │ (4 types)    │                 │
│  └──────────────┘  └──────────────┘  └──────────────┘                 │
│                           │                                             │
│                           │ Contains                                    │
│                           ▼                                             │
│                    ┌──────────────┐                                     │
│                    │ Type         │                                     │
│                    │ Processors   │                                     │
│                    │ (7+ types)   │                                     │
│                    └──────────────┘                                     │
│                           │                                             │
│                           │ Uses                                        │
│                    ┌──────┴──────┐                                      │
│                    │             │                                      │
│             ┌──────▼──────┐  ┌──▼──────────┐                           │
│             │     LLM     │  │   Prompt    │                           │
│             │  Provider   │  │   Builder   │                           │
│             └─────────────┘  └─────────────┘                           │
│                                                                          │
│  ┌──────────────┐                                                       │
│  │ Strategies   │                                                       │
│  │              │                                                       │
│  │ • Sequential │                                                       │
│  │ • Parallel   │                                                       │
│  │ • Dependency │                                                       │
│  └──────────────┘                                                       │
└─────────────────────────────────────────────────────────────────────────┘
```

## Generation Request Flow

```
┌─────────────┐
│ HTTP/gRPC   │ Client sends request (prompt + schema)
│   Request   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ Service Layer (objectGen.go)                                │
├─────────────────────────────────────────────────────────────┤
│ 1. Parse request body                                       │
│ 2. Validate (check circular definitions)                    │
│ 3. Create factory with config                               │
│ 4. Get generator from factory                               │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ Generator.Generate(request)                                 │
├─────────────────────────────────────────────────────────────┤
│ Phase 1: Pre-processing (plugins)                           │
│ Phase 2: Cache check (plugins)                              │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ SchemaAnalyzer.Analyze(schema)                              │
├─────────────────────────────────────────────────────────────┤
│ • Extract fields from schema                                │
│ • Calculate depth and complexity                            │
│ • Returns SchemaAnalysis                                    │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ SchemaAnalyzer.DetermineProcessingOrder(fields)             │
├─────────────────────────────────────────────────────────────┤
│ • Create FieldTask for each field                           │
│ • Read ProcessingOrder from schema                          │
│ • Set up dependencies                                       │
│ • Returns []*FieldTask                                      │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ ExecutionStrategy.Schedule(tasks)                           │
├─────────────────────────────────────────────────────────────┤
│ Sequential: 1 stage, all tasks                              │
│ Parallel:   1 stage, parallel execution                     │
│ DependAware: Multiple stages by dependencies                │
│ • Returns ExecutionPlan with stages                         │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ ExecutionStrategy.Execute(plan, executor, context)          │
├─────────────────────────────────────────────────────────────┤
│ For each stage:                                             │
│   For each task in stage:                                   │
│     TaskExecutor.Execute(task, context) ────────────┐       │
└──────┬──────────────────────────────────────────────┼───────┘
       │                                               │
       │                                               ▼
       │         ┌─────────────────────────────────────────────┐
       │         │ TaskExecutor routes to TypeProcessor        │
       │         ├─────────────────────────────────────────────┤
       │         │ 1. Check for byte ops (TTS/Image/STT)      │
       │         │    → ByteProcessor                          │
       │         │ 2. Check schema type                        │
       │         │    → Appropriate TypeProcessor              │
       │         │ 3. Fallback to PrimitiveProcessor           │
       │         └─────────────────┬───────────────────────────┘
       │                           │
       │                           ▼
       │         ┌─────────────────────────────────────────────┐
       │         │ TypeProcessor.Process(task, context)        │
       │         ├─────────────────────────────────────────────┤
       │         │ 1. PromptBuilder.Build() → prompt           │
       │         │ 2. LLMProvider.Generate() → response        │
       │         │ 3. Parse response                           │
       │         │ 4. Return TaskResult                        │
       │         └─────────────────┬───────────────────────────┘
       │                           │
       │ ◄─────────────────────────┘
       │ Returns []*TaskResult (all tasks complete)
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ ResultAssembler.Assemble(results)                           │
├─────────────────────────────────────────────────────────────┤
│ • Collect all TaskResults                                   │
│ • Build final map[string]interface{}                        │
│ • Aggregate metadata (tokens, cost)                         │
│ • Returns GenerationResult                                  │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ Generator continues...                                       │
├─────────────────────────────────────────────────────────────┤
│ Phase 8: Post-processing (plugins)                          │
│ Phase 9: Validation (plugins)                               │
│ Phase 10: Cache result (plugins)                            │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────┐
│  Response   │ JSON data + metadata (cost, tokens)
└─────────────┘
```

## Component Dependency Graph

```
                    ┌──────────────┐
                    │ Generator    │ ◄──── Application boundary
                    │ (interface)  │
                    └──────┬───────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
      ┌─────▼─────┐  ┌────▼────┐  ┌─────▼─────┐
      │  Default  │  │Streaming│  │Progressive│
      │ Generator │  │Generator│  │ Generator │
      └─────┬─────┘  └────┬────┘  └─────┬─────┘
            │             │              │
            └─────────────┼──────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        │                 │                 │
┌───────▼───────┐  ┌──────▼──────┐  ┌──────▼──────┐
│ Schema        │  │ Execution   │  │   Result    │
│ Analyzer      │  │  Strategy   │  │  Assembler  │
└───────────────┘  └──────┬──────┘  └─────────────┘
                          │
                   ┌──────▼──────┐
                   │    Task     │
                   │  Executor   │
                   └──────┬──────┘
                          │
          ┌───────────────┼───────────────┐
          │               │               │
   ┌──────▼──────┐ ┌─────▼─────┐ ┌──────▼──────┐
   │    Type     │ │    LLM    │ │   Prompt    │
   │ Processors  │ │  Provider │ │   Builder   │
   │  (7+ types) │ └───────────┘ └─────────────┘
   └─────────────┘
```

## Type Processor Selection Flow

```
TaskExecutor.Execute(task)
        │
        ▼
┌───────────────────────────────────────┐
│ Check for byte operations?            │
│ (TextToSpeech, Image, SpeechToText)   │
└────┬──────────────────────┬───────────┘
     │ YES                  │ NO
     ▼                      ▼
┌────────────┐    ┌─────────────────────────┐
│   Byte     │    │ Check schema.Type       │
│ Processor  │    └────┬────────────────────┘
└────────────┘         │
                       ├─ "object"  → ObjectProcessor
                       ├─ "array"   → ArrayProcessor
                       ├─ "string"  → PrimitiveProcessor
                       ├─ "boolean" → BooleanProcessor
                       ├─ "number"  → NumberProcessor
                       ├─ "integer" → NumberProcessor
                       └─ default   → PrimitiveProcessor
                               │
                               ▼
                      ┌─────────────────┐
                      │  Processor      │
                      │  .Process()     │
                      └────┬────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
    ┌─────▼─────┐   ┌──────▼──────┐  ┌─────▼─────┐
    │  Build    │   │   Call LLM  │  │   Parse   │
    │  Prompt   │   │   Generate  │  │  Response │
    └───────────┘   └─────────────┘  └───────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  TaskResult   │
                    └───────────────┘
```

## Factory Component Creation Order

```
GeneratorFactory.Create()
        │
        ├─► 1. createAnalyzer()
        │      └─► NewDefaultSchemaAnalyzer()
        │
        ├─► 2. createExecutor()
        │      ├─► createLLMProvider()
        │      │      └─► NewOpenAIProvider()
        │      │
        │      ├─► createPromptBuilder()
        │      │      └─► NewDefaultPromptBuilder()
        │      │
        │      └─► createTypeProcessors()
        │             ├─► NewPrimitiveProcessor(llm, prompt)
        │             ├─► NewObjectProcessor(llm, prompt, analyzer)
        │             ├─► NewArrayProcessor(llm, prompt)
        │             ├─► NewBooleanProcessor(llm, prompt)
        │             ├─► NewNumberProcessor(llm, prompt)
        │             ├─► NewByteProcessor(llm, prompt)
        │             └─► NewMapProcessor(llm, prompt)
        │
        ├─► 3. createAssembler()
        │      └─► Based on mode:
        │          ├─► ModeSync → NewDefaultAssembler()
        │          ├─► ModeStreaming → NewStreamingAssembler()
        │          ├─► ModeStreamingComplete → NewCompleteStreamingAssembler()
        │          └─► ModeStreamingProgressive → NewProgressiveObjectAssembler()
        │
        └─► 4. createStrategy()
               └─► Based on mode:
                   ├─► ModeSync → NewSequentialStrategy()
                   ├─► ModeParallel → NewParallelStrategy(maxConcurrency)
                   └─► ModeDependencyAware → NewDependencyAwareStrategy(maxConcurrency)
        │
        ▼
New{Default|Streaming|Progressive}Generator(
    analyzer,
    executor,
    assembler,
    strategy
)
```

## Streaming Flow Variants

### Field-Level Streaming (StreamingGenerator)

```
Request → Analyze → Plan → Execute (parallel) 
                                      │
                    ┌─────────────────┴─────────────────┐
                    │                                   │
              Task completes                      Task completes
                    │                                   │
                    ▼                                   ▼
              StreamChunk ──────► Channel ◄────── StreamChunk
              { key: "name",                { key: "age",
                value: "John" }               value: 25 }
                    │                                   │
                    └─────────────────┬─────────────────┘
                                      ▼
                            Client receives chunks
                            as fields complete
```

### Token-Level Streaming (ProgressiveGenerator)

```
Request → Analyze → Plan → Execute (parallel)
                                      │
                    ┌─────────────────┴─────────────────┐
                    │                                   │
              Task streams tokens              Task streams tokens
                    │                                   │
         Token → Token → Token           Token → Token → Token
           │       │       │               │       │       │
           ▼       ▼       ▼               ▼       ▼       ▼
         ┌──────────────────────────────────────────────────┐
         │        ProgressiveObjectAssembler                │
         │  (emits accumulated state every 100ms)           │
         └────────────────────┬─────────────────────────────┘
                              │
                              ▼
                    AccumulatedStreamChunk
                    { currentMap: { ... },
                      partialValue: "Jo...",
                      ... }
                              │
                              ▼
                    Client receives incremental
                    updates as tokens arrive
```

## Strategy Comparison

```
┌──────────────────────────────────────────────────────────────┐
│                     SEQUENTIAL STRATEGY                       │
├──────────────────────────────────────────────────────────────┤
│  Stage 1: Task A → Task B → Task C → Task D                 │
│  Time: ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ (slowest)          │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                     PARALLEL STRATEGY                         │
├──────────────────────────────────────────────────────────────┤
│  Stage 1: Task A ━━━━━━                                      │
│           Task B ━━━━━━━━                                    │
│           Task C ━━━━━━━                                     │
│           Task D ━━━━━━                                      │
│  Time: ━━━━━━━━━━ (faster, ignores dependencies)            │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                 DEPENDENCY-AWARE STRATEGY                     │
├──────────────────────────────────────────────────────────────┤
│  Stage 1: Task A ━━━━━━                                      │
│           Task B ━━━━━━━━ (no dependencies)                 │
│                                                               │
│  Stage 2: Task C ━━━━━━━ (depends on A)                     │
│           Task D ━━━━━━ (depends on A, B)                   │
│                                                               │
│  Time: ━━━━━━━━━━━━━━ (optimal, respects dependencies)      │
└──────────────────────────────────────────────────────────────┘
```

## Dependency Graph Example

Schema with `processingOrder`:
```json
{
  "name": { "type": "string" },
  "city": { "type": "string" },
  "country": { "type": "string" },
  "address": {
    "type": "string",
    "processingOrder": ["city", "country"]
  }
}
```

Dependency Graph:
```
    ┌──────┐       ┌──────────┐
    │ name │       │  city    │
    └──────┘       └────┬─────┘
                        │
                        └──────┐
                               │
                               ▼
    ┌─────────┐        ┌───────────┐
    │ country │───────►│  address  │
    └─────────┘        └───────────┘

DependencyAwareStrategy execution:

Stage 1 (parallel):
  - name
  - city
  - country

Stage 2 (after stage 1):
  - address (has access to city + country values)
```

---

These diagrams show the complete architecture and data flow of the JOS system. Use them alongside the documentation markdown files for a complete understanding.
