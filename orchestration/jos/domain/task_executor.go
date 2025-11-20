package domain

import (
	"context"
	"sync"
)

// TaskExecutor - Executes field generation tasks by delegating to type-specific processors
//
// Implementation: infrastructure/execution/task_executor.go (CompositeTaskExecutor)
// Created by: factory/generator_factory.go:createExecutor()
// Used by: ExecutionStrategy implementations
//
// Responsibilities:
//   - Route tasks to appropriate TypeProcessor based on schema type
//   - Handle special cases (byte operations: TTS, Image, STT)
//   - Coordinate with LLMProvider and PromptBuilder
//   - Execute batches of tasks
//   - Process decision points and return additional results from branches
//
// Execute returns a slice of TaskResults to support decision points generating
// multiple sibling fields. The first result is the primary field, subsequent
// results are additional fields from decision point branches.
type TaskExecutor interface {
	Execute(ctx context.Context, task *FieldTask, execContext *ExecutionContext) ([]*TaskResult, error)
	ExecuteBatch(ctx context.Context, tasks []*FieldTask, execContext *ExecutionContext) ([]*TaskResult, error)
}

// WorkerPool is an interface for controlling concurrent goroutine execution
type WorkerPool interface {
	Submit(fn func())
	SubmitWithContext(ctx context.Context, fn func()) bool
	AvailableWorkers() int
	MaxWorkers() int
}

// ExecutionContext provides context for task execution
type ExecutionContext struct {
	request          *GenerationRequest
	parentContext    *ExecutionContext
	generatedValues  map[string]interface{}
	metadata         map[string]interface{}
	promptContext    *PromptContext
	generationConfig *GenerationConfig
	workerPool       WorkerPool
	mu               sync.RWMutex // Protects generatedValues map from concurrent access
}

func NewExecutionContext(request *GenerationRequest) *ExecutionContext {
	return &ExecutionContext{
		request:          request,
		generatedValues:  make(map[string]interface{}),
		metadata:         make(map[string]interface{}),
		promptContext:    NewPromptContext(),
		generationConfig: DefaultGenerationConfig(),
	}
}

// SetWorkerPool sets the worker pool for this execution context
func (e *ExecutionContext) SetWorkerPool(pool WorkerPool) {
	e.workerPool = pool
}

func (e *ExecutionContext) WithParent(parent *FieldTask) *ExecutionContext {
	return &ExecutionContext{
		request:          e.request,
		parentContext:    e,
		generatedValues:  e.copyGeneratedValues(),
		metadata:         e.copyMetadata(),
		promptContext:    e.promptContext,
		generationConfig: e.generationConfig,
		workerPool:       e.workerPool,
	}
}

// WithItemContext creates a new context with an isolated PromptContext for array items
func (e *ExecutionContext) WithItemContext(parent *FieldTask, itemSpecificContext string) *ExecutionContext {
	// Create a new isolated PromptContext
	newPromptContext := NewPromptContext()
	// Copy base prompts from parent
	for _, prompt := range e.promptContext.Prompts {
		newPromptContext.AddPrompt(prompt)
	}
	// Set item-specific current generation context
	newPromptContext.CurrentGen = itemSpecificContext

	return &ExecutionContext{
		request:          e.request,
		parentContext:    e,
		generatedValues:  e.copyGeneratedValues(),
		metadata:         e.copyMetadata(),
		promptContext:    newPromptContext,
		generationConfig: e.generationConfig,
		workerPool:       e.workerPool,
	}
}

func (e *ExecutionContext) Request() *GenerationRequest         { return e.request }
func (e *ExecutionContext) PromptContext() *PromptContext       { return e.promptContext }
func (e *ExecutionContext) GenerationConfig() *GenerationConfig { return e.generationConfig }
func (e *ExecutionContext) WorkerPool() WorkerPool              { return e.workerPool }

// GeneratedValues returns a copy of the generated values map for thread safety
func (e *ExecutionContext) GeneratedValues() map[string]interface{} {
	return e.copyGeneratedValues()
}

// GetGeneratedValue safely retrieves a single value from the generated values map
func (e *ExecutionContext) GetGeneratedValue(key string) (interface{}, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	value, exists := e.generatedValues[key]
	return value, exists
}

func (e *ExecutionContext) SetGeneratedValue(key string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.generatedValues[key] = value
}

func (e *ExecutionContext) copyGeneratedValues() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copied := make(map[string]interface{})
	for k, v := range e.generatedValues {
		copied[k] = v
	}
	return copied
}

func (e *ExecutionContext) copyMetadata() map[string]interface{} {
	copied := make(map[string]interface{})
	for k, v := range e.metadata {
		copied[k] = v
	}
	return copied
}
