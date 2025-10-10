package execution

import (
	"objectGeneration/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

// BooleanProcessor handles boolean with voting
type BooleanProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	systemPromptProvider SystemPromptProvider
}

func NewBooleanProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *BooleanProcessor {
	return &BooleanProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: NewDefaultSystemPromptProvider(),
	}
}

func NewBooleanProcessorWithPromptProvider(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder, promptProvider SystemPromptProvider) *BooleanProcessor {
	return &BooleanProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: promptProvider,
	}
}

func (p *BooleanProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Boolean
}

func (p *BooleanProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Use primitive processor with the same prompt provider
	primitiveProc := NewPrimitiveProcessorWithPromptProvider(p.llmProvider, p.promptBuilder, p.systemPromptProvider)
	return primitiveProc.Process(task, context)
}

// NumberProcessor handles numeric types
type NumberProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	systemPromptProvider SystemPromptProvider
}

func NewNumberProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *NumberProcessor {
	return &NumberProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: NewDefaultSystemPromptProvider(),
	}
}

func NewNumberProcessorWithPromptProvider(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder, promptProvider SystemPromptProvider) *NumberProcessor {
	return &NumberProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: promptProvider,
	}
}

func (p *NumberProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Number || schemaType == jsonSchema.Integer
}

func (p *NumberProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Use primitive processor with the same prompt provider
	primitiveProc := NewPrimitiveProcessorWithPromptProvider(p.llmProvider, p.promptBuilder, p.systemPromptProvider)
	return primitiveProc.Process(task, context)
}
