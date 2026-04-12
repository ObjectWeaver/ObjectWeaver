package execution

import (
	"context"
	"objectweaver/orchestration/jos/domain"

	"objectweaver/jsonSchema"
)

// BooleanProcessor handles boolean with voting
type BooleanProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	systemPromptProvider SystemPromptProvider
	epstimicOrchestrator EpstimicOrchestrator
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

// SetEpstimicOrchestrator sets the epstimic orchestrator for validation
func (p *BooleanProcessor) SetEpstimicOrchestrator(orchestrator EpstimicOrchestrator) {
	p.epstimicOrchestrator = orchestrator
}

func (p *BooleanProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Boolean
}

func (p *BooleanProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Use primitive processor with the same prompt provider
	primitiveProc := NewPrimitiveProcessorWithPromptProvider(p.llmProvider, p.promptBuilder, p.systemPromptProvider)
	// Propagate epstimic orchestrator if set
	if p.epstimicOrchestrator != nil {
		primitiveProc.SetEpstimicOrchestrator(p.epstimicOrchestrator)
	}
	return primitiveProc.Process(ctx, task, execContext)
}

// NumberProcessor handles numeric types
type NumberProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	systemPromptProvider SystemPromptProvider
	epstimicOrchestrator EpstimicOrchestrator
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

// SetEpstimicOrchestrator sets the epstimic orchestrator for validation
func (p *NumberProcessor) SetEpstimicOrchestrator(orchestrator EpstimicOrchestrator) {
	p.epstimicOrchestrator = orchestrator
}

func (p *NumberProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Number || schemaType == jsonSchema.Integer
}

func (p *NumberProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Use primitive processor with the same prompt provider
	primitiveProc := NewPrimitiveProcessorWithPromptProvider(p.llmProvider, p.promptBuilder, p.systemPromptProvider)
	// Propagate epstimic orchestrator if set
	if p.epstimicOrchestrator != nil {
		primitiveProc.SetEpstimicOrchestrator(p.epstimicOrchestrator)
	}
	return primitiveProc.Process(ctx, task, execContext)
}
