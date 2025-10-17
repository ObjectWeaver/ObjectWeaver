package domain

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
type TaskExecutor interface {
	Execute(task *FieldTask, context *ExecutionContext) (*TaskResult, error)
	ExecuteBatch(tasks []*FieldTask, context *ExecutionContext) ([]*TaskResult, error)
}

// ExecutionContext provides context for task execution
type ExecutionContext struct {
	request          *GenerationRequest
	parentContext    *ExecutionContext
	generatedValues  map[string]interface{}
	metadata         map[string]interface{}
	promptContext    *PromptContext
	generationConfig *GenerationConfig
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

func (e *ExecutionContext) WithParent(parent *FieldTask) *ExecutionContext {
	return &ExecutionContext{
		request:          e.request,
		parentContext:    e,
		generatedValues:  e.copyGeneratedValues(),
		metadata:         e.copyMetadata(),
		promptContext:    e.promptContext,
		generationConfig: e.generationConfig,
	}
}

func (e *ExecutionContext) Request() *GenerationRequest             { return e.request }
func (e *ExecutionContext) GeneratedValues() map[string]interface{} { return e.generatedValues }
func (e *ExecutionContext) PromptContext() *PromptContext           { return e.promptContext }
func (e *ExecutionContext) GenerationConfig() *GenerationConfig     { return e.generationConfig }

func (e *ExecutionContext) SetGeneratedValue(key string, value interface{}) {
	e.generatedValues[key] = value
}

func (e *ExecutionContext) copyGeneratedValues() map[string]interface{} {
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