package execution

import (
	"context"
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// MapProcessor handles map-type fields
type MapProcessor struct {
	llmProvider    domain.LLMProvider
	promptBuilder  domain.PromptBuilder
	fieldProcessor *FieldProcessor
}

func NewMapProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *MapProcessor {
	return &MapProcessor{
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
	}
}

func NewMapProcessorWithFieldProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder, fieldProcessor *FieldProcessor) *MapProcessor {
	return &MapProcessor{
		llmProvider:    llmProvider,
		promptBuilder:  promptBuilder,
		fieldProcessor: fieldProcessor,
	}
}

func (p *MapProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Map
}

func (p *MapProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Simplified map processing
	result := make(map[string]interface{})
	metadata := domain.NewResultMetadata()

	taskResult := domain.NewTaskResult(task.ID(), task.Key(), result, metadata)
	return taskResult.WithPath(task.Path()), nil
}
