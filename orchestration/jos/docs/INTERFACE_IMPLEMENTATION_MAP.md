# Interface Implementation Map

This document provides a clear, visual mapping of every interface to its implementation(s).

## How to Use This Document

1. Find the interface you're working with
2. See exactly which file(s) implement it
3. Jump to the implementation file to see the code

---

## Core Application Interface

### `Generator`
**Location**: `domain/interfaces.go:24-37`

**Purpose**: Main API boundary - generates JSON objects from schemas

**Implementations**:

| Class | File | Mode | Description |
|-------|------|------|-------------|
| `DefaultGenerator` | `application/default_generator.go:9` | Synchronous | Standard batch generation |
| `StreamingGenerator` | `application/streaming_generator.go:9` | Field Streaming | Streams complete fields as ready |
| `ProgressiveGenerator` | `application/progressive_generator.go:9` | Token Streaming | Streams tokens as they arrive |

**Created By**: `factory/generator_factory.go:Create()` (line 27)

**Used By**:
- `service/objectGen.go:ObjectGen()` - HTTP service
- `grpcService/generateObjectV2.go` - gRPC service
- `grpcService/streamGenerateObjectV2.go` - gRPC streaming

---

## Schema Analysis

### `SchemaAnalyzer`
**Location**: `domain/interfaces.go:41-53`

**Purpose**: Breaks down JSON schemas into processable components

**Implementation**:

| Class | File | Description |
|-------|------|-------------|
| `DefaultSchemaAnalyzer` | `infrastructure/analysis/schema_analyzer.go:11` | Analyzes schemas, extracts fields, determines processing order |

**Created By**: `factory/generator_factory.go:createAnalyzer()` (line 54)

**Used By**:
- All `Generator` implementations
- `ObjectProcessor` for nested objects (line 15)

**Key Methods**:
- `Analyze()` - Line 18: Analyzes schema structure
- `ExtractFields()` - Line 36: Extracts field definitions
- `DetermineProcessingOrder()` - Line 73: Creates tasks with dependencies

---

## Task Execution

### `TaskExecutor`
**Location**: `domain/interfaces.go:76-90`

**Purpose**: Executes field generation tasks

**Implementation**:

| Class | File | Description |
|-------|------|-------------|
| `CompositeTaskExecutor` | `infrastructure/execution/task_executor.go:8` | Routes tasks to appropriate TypeProcessors |

**Created By**: `factory/generator_factory.go:createExecutor()` (line 57)

**Used By**: All `ExecutionStrategy` implementations

**Key Methods**:
- `Execute()` - Line 32: Routes single task to processor
- `ExecuteBatch()` - Line 61: Executes multiple tasks

**Routing Logic** (line 32-56):
1. Check for byte operations (TTS, Image, STT) → `ByteProcessor`
2. Find processor by `CanProcess(type)` → Type-specific processor
3. Fallback to `PrimitiveProcessor`

---

## Type Processing

### `TypeProcessor`
**Location**: `domain/interfaces.go:354-374`

**Purpose**: Handles generation for specific JSON schema types

**Implementations**:

| Class | File | Handles | Line |
|-------|------|---------|------|
| `PrimitiveProcessor` | `infrastructure/execution/primitive_processor.go` | string | - |
| `ObjectProcessor` | `infrastructure/execution/object_processor.go` | object | 11 |
| `ArrayProcessor` | `infrastructure/execution/array_processor.go` | array | 11 |
| `BooleanProcessor` | `infrastructure/execution/primitive_processors.go` | boolean | 12 |
| `NumberProcessor` | `infrastructure/execution/primitive_processors.go` | number | 42 |
| `MapProcessor` | `infrastructure/execution/map_processor.go` | key-value maps | 11 |
| `ByteProcessor` | `infrastructure/execution/byte_processor.go` | TTS/Image/STT | 16 |

**Created By**: `factory/generator_factory.go:createTypeProcessors()` (line 65)

**Used By**: `CompositeTaskExecutor`

**Common Pattern**:
```go
type XProcessor struct {
    llmProvider   domain.LLMProvider
    promptBuilder domain.PromptBuilder
}

func (p *XProcessor) CanProcess(t jsonSchema.DataType) bool {
    return t == jsonSchema.MyType
}

func (p *XProcessor) Process(task, context) (*TaskResult, error) {
    prompt := p.promptBuilder.Build(task, context)
    response := p.llmProvider.Generate(prompt, config)
    // Parse and return result
}
```

---

### `StreamingTypeProcessor`
**Location**: `domain/interfaces.go:377-380`

**Purpose**: TypeProcessor with token-level streaming support

**Implementations**:

| Class | File | Handles | Line |
|-------|------|---------|------|
| `StreamingPrimitiveProcessor` | `infrastructure/execution/streaming_processors.go` | string | 18 |
| `StreamingObjectProcessor` | `infrastructure/execution/streaming_processors.go` | object | 126 |
| `StreamingArrayProcessor` | `infrastructure/execution/streaming_processors.go` | array | 236 |

**Created By**: `factory/generator_factory.go:createTypeProcessors()` (line 75-78, when mode is `ModeStreamingProgressive`)

**Used By**: `ProgressiveGenerator`

---

## Prompt Building

### `PromptBuilder`
**Location**: `domain/interfaces.go:145-158`

**Purpose**: Builds contextual prompts for LLM generation

**Implementation**:

| Class | File | Description |
|-------|------|-------------|
| `DefaultPromptBuilder` | `infrastructure/prompt/prompt_builder.go` | Builds prompts with context awareness |

**Created By**: `factory/generator_factory.go:createPromptBuilder()` (line 130)

**Used By**: All `TypeProcessor` implementations

**Key Methods**:
- `Build()` - Creates prompt from task and context
- `BuildWithHistory()` - Includes generation history for retries

---

## LLM Interaction

### `LLMProvider`
**Location**: `domain/interfaces.go:197-213`

**Purpose**: Abstract interface for LLM interactions

**Implementation**:

| Class | File | Description |
|-------|------|-------------|
| `OpenAIProvider` | `infrastructure/llm/openai_provider.go` | Wraps existing job submission system |

**Created By**: `factory/generator_factory.go:createLLMProvider()` (line 120)

**Used By**: All `TypeProcessor` implementations

**Key Methods**:
- `Generate()` - Synchronous generation
- `SupportsStreaming()` - Capability check
- `ModelType()` - Returns model identifier

---

### `TokenStreamingProvider`
**Location**: `domain/interfaces.go:215-220`

**Purpose**: LLM provider with token-level streaming

**Implementations**: Same as `LLMProvider` (OpenAIProvider implements both)

**Key Methods**:
- All `LLMProvider` methods
- `GenerateStream()` - String-level streaming
- `GenerateTokenStream()` - Token-level streaming
- `SupportsTokenStreaming()` - Capability check

---

### `ByteOperationProvider`
**Location**: `domain/interfaces.go:222-227`

**Purpose**: Provider supporting TTS, Image generation, STT

**Implementations**: Same as `LLMProvider` (OpenAIProvider implements multiple interfaces)

**Key Methods**:
- `GenerateAudio()` - Text-to-Speech
- `GenerateImage()` - Image generation
- `TranscribeAudio()` - Speech-to-Text
- `SupportsByteOperations()` - Capability check

---

## Result Assembly

### `ResultAssembler`
**Location**: `domain/interfaces.go:289-307`

**Purpose**: Assembles task results into final output

**Implementations**:

| Class | File | Mode | Description |
|-------|------|------|-------------|
| `DefaultAssembler` | `infrastructure/assembly/default.go` | Sync | Collects all results, builds final map |
| `StreamingAssembler` | `infrastructure/assembly/streaming.go` | Streaming | Emits StreamChunks as fields complete |
| `CompleteStreamingAssembler` | `infrastructure/assembly/complete.go` | Streaming | Only emits complete object |
| `ProgressiveObjectAssembler` | `infrastructure/assembly/progressive.go` | Progressive | Emits accumulated state on interval |

**Created By**: `factory/generator_factory.go:createAssembler()` (line 98)

**Used By**: All `Generator` implementations

---

### `StreamingAssembler`
**Location**: `domain/interfaces.go:309-312`

**Purpose**: Assembler with field-level streaming support

**Implementations**: `StreamingAssembler`, `CompleteStreamingAssembler`

**Key Methods**:
- `Assemble()` - Inherited from ResultAssembler
- `AssembleStreaming()` - Streams field completions

---

### `ProgressiveAssembler`
**Location**: `domain/interfaces.go:314-318`

**Purpose**: Assembler with token-level streaming support

**Implementation**: `ProgressiveObjectAssembler`

**Key Methods**:
- `Assemble()` - Inherited from ResultAssembler
- `AssembleProgressive()` - Streams accumulated tokens

---

## Execution Strategies

### `ExecutionStrategy`
**Location**: `domain/interfaces.go:320-337`

**Purpose**: Controls task scheduling and execution order

**Implementations**:

| Class | File | Description |
|-------|------|-------------|
| `SequentialStrategy` | `infrastructure/strategies/sequential.go:7` | Executes tasks one-by-one in order |
| `ParallelStrategy` | `infrastructure/strategies/parrell.go:12` | Executes all tasks concurrently |
| `DependencyAwareStrategy` | `infrastructure/strategies/dependency_aware.go:12` | Topological sort, parallel where possible |

**Created By**: `factory/generator_factory.go:createStrategy()` (line 108)

**Used By**: All `Generator` implementations

**Key Methods**:
- `Schedule()` - Creates execution plan with stages
- `Execute()` - Executes plan using provided executor

**Strategy Selection** (config mode):
- `ModeSync` → Sequential
- `ModeParallel` → Parallel
- `ModeDependencyAware` → DependencyAware
- `ModeStreaming*` → Parallel

---

## Summary Table

| Interface | Implementation(s) | File(s) |
|-----------|-------------------|---------|
| `Generator` | 3 variants | `application/*_generator.go` |
| `SchemaAnalyzer` | 1 | `infrastructure/analysis/schema_analyzer.go` |
| `TaskExecutor` | 1 | `infrastructure/execution/task_executor.go` |
| `TypeProcessor` | 7+ variants | `infrastructure/execution/*_processor.go` |
| `StreamingTypeProcessor` | 3 variants | `infrastructure/execution/streaming_processors.go` |
| `PromptBuilder` | 1 | `infrastructure/prompt/prompt_builder.go` |
| `LLMProvider` | 1 (multi-interface) | `infrastructure/llm/openai_provider.go` |
| `TokenStreamingProvider` | 1 (same as above) | `infrastructure/llm/openai_provider.go` |
| `ByteOperationProvider` | 1 (same as above) | `infrastructure/llm/openai_provider.go` |
| `ResultAssembler` | 4 variants | `infrastructure/assembly/*.go` |
| `StreamingAssembler` | 2 variants | `infrastructure/assembly/{streaming,complete}.go` |
| `ProgressiveAssembler` | 1 | `infrastructure/assembly/progressive.go` |
| `ExecutionStrategy` | 3 variants | `infrastructure/strategies/*.go` |

---

## Navigation Tips

### From Interface → Implementation
1. Open `domain/interfaces.go`
2. Find your interface (use search)
3. Read the doc comment - it lists implementations
4. Click/navigate to the implementation file

### From Implementation → Interface
1. Open implementation file (e.g., `default_generator.go`)
2. Look at struct definition
3. See which interface methods it implements
4. All interfaces are in `domain/interfaces.go`

### From Usage → Both
1. Find where interface is used (e.g., in a generator)
2. See the interface type (e.g., `domain.SchemaAnalyzer`)
3. Go to `domain/interfaces.go` for contract
4. Go to implementation for behavior

---

## Quick Jumps

**All Interfaces**: `domain/interfaces.go`

**Generators**: `application/` directory
- Default: `application/default_generator.go:9`
- Streaming: `application/streaming_generator.go:9`
- Progressive: `application/progressive_generator.go:9`

**Factory**: `factory/generator_factory.go`
- Component creation: `Create()` method (line 27)

**Service Entry**: `service/objectGen.go:ObjectGen()`

**Type Processors**: `infrastructure/execution/` directory
- Look for `*_processor.go` files

**Strategies**: `infrastructure/strategies/` directory

**Assemblers**: `infrastructure/assembly/` directory
