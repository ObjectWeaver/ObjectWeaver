# JOS (JSON Object Schema) Architecture Flow

## Table of Contents
1. [Overview](#overview)
2. [Entry Points](#entry-points)
3. [Initialization Flow](#initialization-flow)
4. [Generation Flow](#generation-flow)
5. [Component Dependencies](#component-dependencies)
6. [File Structure](#file-structure)

---

## Overview

The JOS system is a modular JSON object generation system that uses Clean Architecture principles with dependency injection. It generates structured JSON data based on schemas by breaking down complex schemas into tasks, executing them with LLMs, and assembling the results.

**Key Design Patterns:**
- **Factory Pattern**: `GeneratorFactory` creates fully configured generators
- **Strategy Pattern**: Different execution strategies (Sequential, Parallel, DependencyAware)
- **Composite Pattern**: `CompositeTaskExecutor` delegates to type-specific processors
- **Chain of Responsibility**: Plugin registry for pre/post-processing

---

## Entry Points

### 1. HTTP Service Entry Point
**File**: `/service/objectGen.go`
**Function**: `ObjectGen()`

```
HTTP POST /objectgen
    ↓
service/objectGen.go:ObjectGen()
    ↓
Decodes client.RequestBody (prompt + schema)
    ↓
Creates factory & generator
    ↓
Calls generator.Generate(request)
    ↓
Returns JSON response
```

**Key Lines**:
- Line 73-77: Creates `GeneratorFactory` with config
- Line 80-86: Calls `factory.Create()` to get generator
- Line 89-96: Creates `GenerationRequest` and calls `generator.Generate()`

### 2. gRPC Service Entry Point
**Files**: `/grpcService/generateObjectV2.go`, `/grpcService/streamGenerateObjectV2.go`

These follow similar patterns but use gRPC request/response types.

---

## Initialization Flow

### Phase 1: Factory Creation
**File**: `factory/generator_factory.go`

```
NewGeneratorFactory(config) 
    ↓
Stores GeneratorConfig
    - Mode (Sync, Parallel, Streaming, etc.)
    - MaxConcurrency
    - LLMProvider
    - Plugins
```

**File Reference**: `factory/generator_factory.go:19-24`

### Phase 2: Component Creation
**File**: `factory/generator_factory.go`
**Method**: `Create()`

The factory creates components in this order:

```
1. factory.Create()  [Line 27]
       ↓
2. createAnalyzer()  [Line 29]
   → Returns: domain.SchemaAnalyzer
   → Implementation: analysis.DefaultSchemaAnalyzer
   → File: infrastructure/analysis/schema_analyzer.go:14
       ↓
3. createExecutor()  [Line 30]
   → Creates LLMProvider  [Line 61]
   → Creates PromptBuilder  [Line 62]
   → Creates TypeProcessors  [Line 65]
   → Returns: domain.TaskExecutor
   → Implementation: execution.CompositeTaskExecutor
   → File: infrastructure/execution/task_executor.go:12
       ↓
4. createAssembler()  [Line 31]
   → Returns: domain.ResultAssembler
   → Implementations based on mode:
      - assembly.DefaultAssembler (sync)
      - assembly.StreamingAssembler (streaming)
      - assembly.ProgressiveObjectAssembler (progressive)
   → Files: infrastructure/assembly/*.go
       ↓
5. createStrategy()  [Line 32]
   → Returns: domain.ExecutionStrategy
   → Implementations based on mode:
      - strategies.SequentialStrategy
      - strategies.ParallelStrategy
      - strategies.DependencyAwareStrategy
   → Files: infrastructure/strategies/*.go
       ↓
6. Create generator instance  [Line 37-43]
   → Based on config.Mode:
      - application.DefaultGenerator (sync)
      - application.StreamingGenerator (streaming)
      - application.ProgressiveGenerator (progressive)
   → Files: application/*_generator.go
       ↓
7. Register plugins  [Line 46-48]
   → If generator implements PluginRegistry
   → Registers cache, validation, observability plugins
```

### Phase 3: Dependency Injection

The generator receives all dependencies via constructor:

```go
// File: application/default_generator.go:19-30
NewDefaultGenerator(
    analyzer  domain.SchemaAnalyzer,      // How to break down schemas
    executor  domain.TaskExecutor,         // How to execute field tasks
    assembler domain.ResultAssembler,      // How to assemble results
    strategy  domain.ExecutionStrategy,    // When/how to execute tasks
)
```

---

## Generation Flow

### Complete Flow Diagram

```
generator.Generate(request)
    │
    ├─ Phase 1: Pre-processing
    │   └─ application/default_generator.go:38
    │       └─ plugins.ApplyPreProcessors()
    │
    ├─ Phase 2: Cache Check
    │   └─ application/default_generator.go:43
    │       └─ plugins.GetFromCache()
    │
    ├─ Phase 3: Schema Analysis
    │   └─ application/default_generator.go:48
    │       └─ analyzer.Analyze(schema)
    │           └─ infrastructure/analysis/schema_analyzer.go:18
    │               ├─ ExtractFields() - Parse schema properties
    │               │   └─ Returns: []*FieldDefinition
    │               └─ Calculate metrics (depth, nested objects)
    │
    ├─ Phase 4: Task Planning
    │   └─ application/default_generator.go:54
    │       └─ analyzer.DetermineProcessingOrder(fields)
    │           └─ infrastructure/analysis/schema_analyzer.go:73
    │               ├─ Creates FieldTask for each field
    │               ├─ Reads ProcessingOrder from schema
    │               ├─ Sets up dependencies between tasks
    │               └─ Returns: []*FieldTask
    │
    ├─ Phase 5: Execution Scheduling
    │   └─ application/default_generator.go:59
    │       └─ strategy.Schedule(tasks)
    │           └─ infrastructure/strategies/*.go
    │               ├─ Sequential: One stage, all tasks
    │               ├─ Parallel: One stage, parallel execution
    │               └─ DependencyAware: Multiple stages by dependencies
    │               └─ Returns: *ExecutionPlan with stages
    │
    ├─ Phase 6: Task Execution
    │   └─ application/default_generator.go:65
    │       └─ strategy.Execute(plan, executor, context)
    │           └─ For each stage:
    │               └─ executor.Execute(task, context)
    │                   └─ infrastructure/execution/task_executor.go:32
    │                       ├─ Check for byte operations (TTS, Image, STT)
    │                       │   └─ execution.ByteProcessor
    │                       │       └─ llmProvider (ByteOperationProvider)
    │                       │
    │                       ├─ Find processor by schema type
    │                       │   ├─ PrimitiveProcessor (string, boolean, number)
    │                       │   ├─ ObjectProcessor (nested objects)
    │                       │   ├─ ArrayProcessor (arrays)
    │                       │   ├─ MapProcessor (key-value objects)
    │                       │   └─ All in: infrastructure/execution/*_processor.go
    │                       │
    │                       └─ processor.Process(task, context)
    │                           ├─ promptBuilder.Build() - Create prompt
    │                           │   └─ infrastructure/prompt/prompt_builder.go
    │                           │
    │                           ├─ llmProvider.Generate() - Call LLM
    │                           │   └─ infrastructure/llm/openai_provider.go
    │                           │       └─ Wraps existing job submission
    │                           │
    │                           └─ Parse and return TaskResult
    │
    ├─ Phase 7: Result Assembly
    │   └─ application/default_generator.go:71
    │       └─ assembler.Assemble(results)
    │           └─ infrastructure/assembly/default.go
    │               ├─ Collects all TaskResults
    │               ├─ Builds final map[string]interface{}
    │               └─ Creates GenerationResult with metadata
    │
    ├─ Phase 8: Post-processing
    │   └─ application/default_generator.go:77
    │       └─ plugins.ApplyPostProcessors()
    │
    ├─ Phase 9: Validation
    │   └─ application/default_generator.go:83
    │       └─ plugins.ApplyValidation()
    │
    └─ Phase 10: Cache Result
        └─ application/default_generator.go:88
            └─ plugins.CacheResult()
```

### Detailed Phase Breakdown

#### Phase 3: Schema Analysis Detail
```
analyzer.Analyze(schema)
    ↓
Receives: *jsonSchema.Definition
    - Properties: map[string]Definition
    - Type: DataType (object, string, array, etc.)
    - Required: []string
    - ProcessingOrder: []string (dependencies)
    ↓
Calls ExtractFields():
    For each property in schema:
        Create FieldDefinition {
            Key: property name
            Definition: *jsonSchema.Definition
            Parent: nil (for top-level)
            Required: check if in Required array
        }
    ↓
Returns SchemaAnalysis {
    Fields: []*FieldDefinition
    TotalFieldCount: count
    MaxDepth: calculated depth
    HasNestedObjects: bool
}
```

**File**: `infrastructure/analysis/schema_analyzer.go:18-31`

#### Phase 4: Task Planning Detail
```
analyzer.DetermineProcessingOrder(fields)
    ↓
For each FieldDefinition:
    Create FieldTask {
        key: field.Key
        definition: field.Definition
        dependencies: [] (initially empty)
    }
    ↓
    If field.Definition.ProcessingOrder exists:
        For each dependency key:
            Add to task.dependencies
    ↓
Returns: []*FieldTask with dependencies set
```

**File**: `infrastructure/analysis/schema_analyzer.go:73-100`

#### Phase 6: Task Execution Detail
```
executor.Execute(task, context)
    ↓
1. Check for byte operations first:
   If task.Definition has:
      - TextToSpeech → ByteProcessor
      - Image → ByteProcessor
      - SpeechToText → ByteProcessor
    ↓
2. Otherwise, find processor by type:
   Match task.Definition.Type:
      - "string", "boolean", "number" → PrimitiveProcessor
      - "object" → ObjectProcessor
      - "array" → ArrayProcessor
    ↓
3. processor.Process(task, context):
    ↓
    a) Build prompt:
       promptBuilder.Build(task, context.PromptContext())
       - Uses field definition
       - Includes parent context
       - Adds existing generated values
    ↓
    b) Call LLM:
       llmProvider.Generate(prompt, config)
       - Submits to job queue
       - Waits for response
       - Returns generated text
    ↓
    c) Parse response:
       - Extract value from LLM output
       - Convert to appropriate type
       - Validate against schema
    ↓
    d) Create TaskResult:
       return TaskResult {
           Key: task.Key
           Value: parsed value
           Metadata: tokens, cost, etc.
       }
```

**Files**:
- Executor: `infrastructure/execution/task_executor.go:32-56`
- Processors: `infrastructure/execution/*_processor.go`
- Prompt Builder: `infrastructure/prompt/prompt_builder.go`
- LLM Provider: `infrastructure/llm/openai_provider.go`

---

## Component Dependencies

### Dependency Graph

```
service/objectGen.go
    ↓
factory/generator_factory.go
    ↓ creates
    ├─→ application/*_generator.go
    │   (DefaultGenerator, StreamingGenerator, ProgressiveGenerator)
    │       ↓ depends on
    │       ├─→ domain.SchemaAnalyzer
    │       │       ↓ implemented by
    │       │       └─→ infrastructure/analysis/schema_analyzer.go
    │       │
    │       ├─→ domain.TaskExecutor
    │       │       ↓ implemented by
    │       │       └─→ infrastructure/execution/task_executor.go
    │       │               ↓ depends on
    │       │               ├─→ domain.LLMProvider
    │       │               │       ↓ implemented by
    │       │               │       └─→ infrastructure/llm/openai_provider.go
    │       │               │
    │       │               ├─→ domain.PromptBuilder
    │       │               │       ↓ implemented by
    │       │               │       └─→ infrastructure/prompt/prompt_builder.go
    │       │               │
    │       │               └─→ []domain.TypeProcessor
    │       │                       ↓ implemented by
    │       │                       └─→ infrastructure/execution/*_processor.go
    │       │
    │       ├─→ domain.ResultAssembler
    │       │       ↓ implemented by
    │       │       └─→ infrastructure/assembly/*.go
    │       │
    │       └─→ domain.ExecutionStrategy
    │               ↓ implemented by
    │               └─→ infrastructure/strategies/*.go
    │
    └─→ domain/* (interfaces and models)
```

### Interface → Implementation Mapping

| Interface | Location | Implementations | Implementation Location |
|-----------|----------|-----------------|------------------------|
| `Generator` | `domain/interfaces.go:8` | `DefaultGenerator`<br>`StreamingGenerator`<br>`ProgressiveGenerator` | `application/default_generator.go:9`<br>`application/streaming_generator.go:9`<br>`application/progressive_generator.go:9` |
| `SchemaAnalyzer` | `domain/interfaces.go:16` | `DefaultSchemaAnalyzer` | `infrastructure/analysis/schema_analyzer.go:11` |
| `TaskExecutor` | `domain/interfaces.go:39` | `CompositeTaskExecutor` | `infrastructure/execution/task_executor.go:8` |
| `TypeProcessor` | `domain/interfaces.go:242` | `PrimitiveProcessor`<br>`ObjectProcessor`<br>`ArrayProcessor`<br>`BooleanProcessor`<br>`NumberProcessor`<br>`ByteProcessor`<br>`MapProcessor` | `infrastructure/execution/primitive_processor.go`<br>`infrastructure/execution/object_processor.go`<br>`infrastructure/execution/array_processor.go`<br>(various *_processor.go files) |
| `PromptBuilder` | `domain/interfaces.go:100` | `DefaultPromptBuilder` | `infrastructure/prompt/prompt_builder.go` |
| `LLMProvider` | `domain/interfaces.go:143` | `OpenAIProvider` | `infrastructure/llm/openai_provider.go` |
| `ResultAssembler` | `domain/interfaces.go:216` | `DefaultAssembler`<br>`StreamingAssembler`<br>`CompleteStreamingAssembler`<br>`ProgressiveObjectAssembler` | `infrastructure/assembly/default.go`<br>`infrastructure/assembly/streaming.go`<br>`infrastructure/assembly/complete.go`<br>`infrastructure/assembly/progressive.go` |
| `ExecutionStrategy` | `domain/interfaces.go:227` | `SequentialStrategy`<br>`ParallelStrategy`<br>`DependencyAwareStrategy` | `infrastructure/strategies/sequential.go`<br>`infrastructure/strategies/parrell.go`<br>`infrastructure/strategies/dependency_aware.go` |

---

## File Structure

### Current Architecture Overview

```
orchestration/jos/
├── domain/                          # Core domain layer (interfaces & models)
│   ├── interfaces.go               # ALL interfaces defined here
│   ├── models.go                   # Domain models (Request, Result, etc.)
│   ├── streaming.go                # Streaming-specific models
│   └── plugin.go                   # Plugin interfaces
│
├── application/                     # Application services (generators)
│   ├── default_generator.go       # Synchronous generator
│   ├── streaming_generator.go     # Streaming generator
│   ├── progressive_generator.go   # Progressive token-level generator
│   └── plugin_registry.go         # Plugin management
│
├── infrastructure/                  # Infrastructure implementations
│   ├── analysis/                   # Schema analysis
│   │   └── schema_analyzer.go     # Implements: SchemaAnalyzer
│   │
│   ├── execution/                  # Task execution
│   │   ├── task_executor.go       # Implements: TaskExecutor
│   │   ├── primitive_processor.go # Implements: TypeProcessor
│   │   ├── object_processor.go    # Implements: TypeProcessor
│   │   ├── array_processor.go     # Implements: TypeProcessor
│   │   ├── byte_processor.go      # Implements: TypeProcessor (TTS/Image/STT)
│   │   └── streaming_processors.go # Implements: StreamingTypeProcessor
│   │
│   ├── llm/                        # LLM integration
│   │   └── openai_provider.go     # Implements: LLMProvider
│   │
│   ├── prompt/                     # Prompt building
│   │   └── prompt_builder.go      # Implements: PromptBuilder
│   │
│   ├── assembly/                   # Result assembly
│   │   ├── default.go             # Implements: ResultAssembler
│   │   ├── streaming.go           # Implements: StreamingAssembler
│   │   └── progressive.go         # Implements: ProgressiveAssembler
│   │
│   └── strategies/                 # Execution strategies
│       ├── sequential.go          # Implements: ExecutionStrategy
│       ├── parrell.go             # Implements: ExecutionStrategy
│       └── dependency_aware.go    # Implements: ExecutionStrategy
│
└── factory/                        # Factory for dependency injection
    ├── generator_factory.go       # Creates and wires all components
    └── config.go                  # Configuration models
```

### Key Design Decisions

1. **All interfaces in `domain/`**: Following Clean Architecture, all interfaces live in the domain layer. This creates a clear contract but makes it harder to find implementations.

2. **Factory-based creation**: The `GeneratorFactory` knows about all concrete implementations and wires them together.

3. **Strategy pattern for execution**: Different strategies control how tasks are scheduled and executed.

4. **Type-specific processors**: Each JSON schema type has a dedicated processor for specialized handling.

---

## Data Flow Example

### Example: Generate a person object

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "age": {"type": "number"},
    "address": {
      "type": "object",
      "properties": {
        "city": {"type": "string"},
        "country": {"type": "string"}
      },
      "processingOrder": ["city"]
    }
  }
}
```

**Flow**:
1. **Analysis**: Extracts 3 top-level fields: `name`, `age`, `address`
2. **Task Planning**: Creates 3 FieldTasks with dependencies
3. **Scheduling**: 
   - Stage 1: `name`, `age` (parallel)
   - Stage 2: `address` (after dependencies)
4. **Execution**:
   - `name` → PrimitiveProcessor → LLM generates name
   - `age` → NumberProcessor → LLM generates age
   - `address` → ObjectProcessor → Recursively processes nested fields
5. **Assembly**: Combines all results into final JSON object

---

## Summary

**Initialization**: `service/objectGen.go` → `factory/generator_factory.go` → Creates all components with DI

**Generation**: `generator.Generate()` → Analyze → Plan → Schedule → Execute → Assemble

**Key Files to Understand**:
1. `factory/generator_factory.go` - Component creation and wiring
2. `application/default_generator.go` - Main generation workflow
3. `infrastructure/execution/task_executor.go` - Task execution coordination
4. `infrastructure/analysis/schema_analyzer.go` - Schema breakdown logic
5. `domain/interfaces.go` - All interface contracts

The system uses **dependency injection** heavily - all components receive their dependencies through constructors, making the system testable and modular.
