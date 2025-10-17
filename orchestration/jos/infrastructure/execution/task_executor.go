package execution

import (
	"fmt"
	"objectweaver/orchestration/jos/domain"
)

// CompositeTaskExecutor delegates to type-specific processors
type CompositeTaskExecutor struct {
	processors    []domain.TypeProcessor
	defaultProc   domain.TypeProcessor
	llmProvider   domain.LLMProvider
	promptBuilder domain.PromptBuilder
}

func NewCompositeTaskExecutor(
	llmProvider domain.LLMProvider,
	promptBuilder domain.PromptBuilder,
	processors []domain.TypeProcessor,
) *CompositeTaskExecutor {
	executor := &CompositeTaskExecutor{
		processors:    processors,
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
	}

	// Set default processor
	executor.defaultProc = NewPrimitiveProcessor(llmProvider, promptBuilder)

	return executor
}

func (e *CompositeTaskExecutor) Execute(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Special case: Check for byte operations first (TTS, Image, STT)
	// This allows STT fields (type: string) to be handled by ByteProcessor
	def := task.Definition()
	if def.TextToSpeech != nil || def.Image != nil || def.SpeechToText != nil {
		// Find ByteProcessor specifically
		for _, processor := range e.processors {
			if byteProc, ok := processor.(*ByteProcessor); ok {
				return byteProc.Process(task, context)
			}
		}
	}

	// Find appropriate processor by type
	for _, processor := range e.processors {
		if processor.CanProcess(task.Definition().Type) {
			return processor.Process(task, context)
		}
	}

	// Fallback to default
	return e.defaultProc.Process(task, context)
}

func (e *CompositeTaskExecutor) ExecuteBatch(tasks []*domain.FieldTask, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, 0, len(tasks))

	for _, task := range tasks {
		result, err := e.Execute(task, context)
		if err != nil {
			return nil, fmt.Errorf("task %s failed: %w", task.Key(), err)
		}
		results = append(results, result)
	}

	return results, nil
}
