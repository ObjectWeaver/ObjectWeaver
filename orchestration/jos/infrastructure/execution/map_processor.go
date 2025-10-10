package execution

import (
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// MapProcessor handles map-type fields
type MapProcessor struct {
	llmProvider   domain.LLMProvider
	promptBuilder domain.PromptBuilder
}

func NewMapProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *MapProcessor {
	return &MapProcessor{
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
	}
}

func (p *MapProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Map
}

func (p *MapProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Simplified map processing
	result := make(map[string]interface{})
	metadata := domain.NewResultMetadata()

	taskResult := domain.NewTaskResult(task.ID(), task.Key(), result, metadata)
	return taskResult.WithPath(task.Path()), nil
}
