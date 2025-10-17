# JOS Quick Reference Guide

## Finding Implementations

Use this quick reference to find where interfaces are implemented.

### Core Interfaces → Implementations

| Interface | File Location | Implementation Class |
|-----------|---------------|---------------------|
| `Generator` | `domain/interfaces.go:24` | `DefaultGenerator` (app/default_generator.go)<br>`StreamingGenerator` (app/streaming_generator.go)<br>`ProgressiveGenerator` (app/progressive_generator.go) |
| `SchemaAnalyzer` | `domain/interfaces.go:41` | `DefaultSchemaAnalyzer` (infra/analysis/schema_analyzer.go) |
| `TaskExecutor` | `domain/interfaces.go:76` | `CompositeTaskExecutor` (infra/execution/task_executor.go) |
| `TypeProcessor` | `domain/interfaces.go:354` | Multiple processors (infra/execution/*_processor.go) |
| `PromptBuilder` | `domain/interfaces.go:145` | `DefaultPromptBuilder` (infra/prompt/prompt_builder.go) |
| `LLMProvider` | `domain/interfaces.go:197` | `OpenAIProvider` (infra/llm/openai_provider.go) |
| `ResultAssembler` | `domain/interfaces.go:289` | Multiple assemblers (infra/assembly/*.go) |
| `ExecutionStrategy` | `domain/interfaces.go:320` | Multiple strategies (infra/strategies/*.go) |

### Package Structure

```
jos/
├── domain/                      # Interfaces & Domain Models
│   ├── interfaces.go           # All interface definitions
│   ├── models.go               # Domain entities (Request, Result, etc.)
│   ├── streaming.go            # Streaming models
│   └── plugin.go               # Plugin system
│
├── application/                 # Application Services (Generators)
│   ├── default_generator.go   # Sync generation
│   ├── streaming_generator.go # Field-level streaming
│   └── progressive_generator.go # Token-level streaming
│
├── infrastructure/              # Implementations
│   ├── analysis/               # Schema analysis
│   ├── execution/              # Task execution & type processors
│   ├── llm/                    # LLM provider implementations
│   ├── prompt/                 # Prompt building
│   ├── assembly/               # Result assembly
│   └── strategies/             # Execution strategies
│
└── factory/                     # Dependency Injection
    ├── generator_factory.go    # Creates & wires components
    └── config.go               # Configuration models
```

## Common Tasks

### 1. Adding a New Type Processor

**Files to modify:**
1. Create: `infrastructure/execution/my_processor.go`
2. Update: `factory/generator_factory.go:createTypeProcessors()` - Add to processor list

**Template:**
```go
type MyProcessor struct {
    llmProvider   domain.LLMProvider
    promptBuilder domain.PromptBuilder
}

func (p *MyProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
    return schemaType == jsonSchema.MyType
}

func (p *MyProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
    // 1. Build prompt
    // 2. Call LLM
    // 3. Parse response
    // 4. Return TaskResult
}
```

### 2. Adding a New Execution Strategy

**Files to modify:**
1. Create: `infrastructure/strategies/my_strategy.go`
2. Update: `factory/generator_factory.go:createStrategy()` - Add case for new mode
3. Update: `factory/config.go` - Add new mode constant

**Must implement:**
- `Schedule(tasks []*FieldTask) (*ExecutionPlan, error)`
- `Execute(plan *ExecutionPlan, executor TaskExecutor, context *ExecutionContext) ([]*TaskResult, error)`

### 3. Adding a New Generator Mode

**Files to modify:**
1. Create: `application/my_generator.go` (implement `domain.Generator`)
2. Update: `factory/generator_factory.go:Create()` - Add case for new mode
3. Update: `factory/config.go` - Add new mode constant

**Must implement:**
- `Generate(request *GenerationRequest) (*GenerationResult, error)`
- `GenerateStream(request *GenerationRequest) (<-chan *StreamChunk, error)` (optional)
- `GenerateStreamProgressive(request *GenerationRequest) (<-chan *AccumulatedStreamChunk, error)` (optional)

## Initialization Sequence

```
1. service/objectGen.go
   └─> factory.NewGeneratorFactory(config)

2. factory/generator_factory.go:Create()
   ├─> createAnalyzer()      → DefaultSchemaAnalyzer
   ├─> createExecutor()
   │   ├─> createLLMProvider()    → OpenAIProvider
   │   ├─> createPromptBuilder()  → DefaultPromptBuilder
   │   └─> createTypeProcessors() → []*TypeProcessor
   ├─> createAssembler()     → Assembler (based on mode)
   └─> createStrategy()      → Strategy (based on mode)

3. New{Default|Streaming|Progressive}Generator(components...)
   └─> Returns fully wired generator
```

## Generation Flow

```
generator.Generate(request)
  ↓
analyzer.Analyze(schema)
  ↓ returns: []*FieldDefinition
analyzer.DetermineProcessingOrder(fields)
  ↓ returns: []*FieldTask (with dependencies)
strategy.Schedule(tasks)
  ↓ returns: *ExecutionPlan (stages)
strategy.Execute(plan, executor, context)
  ↓ for each task:
  └─> executor.Execute(task, context)
      ↓
      typeProcessor.Process(task, context)
        ├─> promptBuilder.Build()
        ├─> llmProvider.Generate()
        └─> return TaskResult
  ↓ returns: []*TaskResult
assembler.Assemble(results)
  ↓ returns: *GenerationResult
```

## Key Design Patterns

### Dependency Injection
All components receive dependencies via constructor (never use global state).

### Strategy Pattern
Execution behavior varies by strategy:
- **Sequential**: Tasks execute one-by-one
- **Parallel**: Independent tasks execute concurrently
- **DependencyAware**: Respects ProcessingOrder, parallelizes when possible

### Composite Pattern
`CompositeTaskExecutor` delegates to type-specific processors.

### Factory Pattern
`GeneratorFactory` creates all components with proper wiring.

### Plugin Pattern (Optional)
Generators can have plugins for cross-cutting concerns (cache, validation, observability).

## Debugging Tips

### 1. Trace execution flow
Add logging at these key points:
- `generator.Generate()` entry
- `analyzer.Analyze()` - see what fields are extracted
- `strategy.Schedule()` - see execution plan stages
- `executor.Execute()` - see each task execution
- `processor.Process()` - see LLM calls

### 2. Find which processor handles a type
Check: `infrastructure/execution/task_executor.go:Execute()`
- Special case: Byte operations checked first (TTS, Image, STT)
- Then: Type-based routing via `processor.CanProcess()`

### 3. Understand dependencies
Check: `infrastructure/analysis/schema_analyzer.go:DetermineProcessingOrder()`
- Reads `Definition.ProcessingOrder` array
- Creates dependency graph for strategies

### 4. See what gets injected
Check: `factory/generator_factory.go:Create()`
- Shows exact component creation and wiring
- Easy to see what each component receives

## Testing Strategy

### Unit Tests
Each component has `*_test.go` file:
- Mock dependencies using interfaces
- Test component in isolation

### Integration Tests
Test full flow:
- Create factory → generator → execute
- Use test schemas
- Verify full pipeline

## Common Patterns

### Reading Context Values
```go
// In a processor or strategy
request := context.Request()
generatedValues := context.GeneratedValues()
promptCtx := context.PromptContext()
config := context.GenerationConfig()
```

### Creating Child Context
```go
// For nested objects
childContext := context.WithParent(currentTask)
```

### Adding Metadata
```go
request.WithMetadata("key", value)
```

## Configuration Options

### GeneratorConfig (factory/config.go)
- `Mode`: Sync, Parallel, Streaming, etc.
- `MaxConcurrency`: Max parallel tasks
- `LLMProvider`: Which provider to use
- `Granularity`: Token vs Field level streaming
- `EnableCache`: Enable caching
- `EnableValidation`: Enable validation
- `Plugins`: Custom plugins

## File Naming Conventions

- `*_generator.go`: Generator implementations
- `*_processor.go`: Type-specific processors
- `*_strategy.go`: Execution strategies
- `*_assembler.go`: Result assemblers
- `*_test.go`: Unit tests
- `interfaces.go`: Interface definitions
- `models.go`: Data structures

## Import Paths

All imports use:
```go
import "objectweaver/orchestration/jos/{package}"
```

Packages:
- `domain` - Interfaces & models
- `application` - Generators
- `factory` - Factory & config
- `infrastructure/analysis`
- `infrastructure/execution`
- `infrastructure/llm`
- `infrastructure/prompt`
- `infrastructure/assembly`
- `infrastructure/strategies`
