package execution

import (
	"fmt"
	"log"
	"objectweaver/orchestration/jos/domain"
)

// CompositeTaskExecutor delegates to type-specific processors
type CompositeTaskExecutor struct {
	processors        []domain.TypeProcessor
	defaultProc       domain.TypeProcessor
	llmProvider       domain.LLMProvider
	promptBuilder     domain.PromptBuilder
	generator         domain.Generator
	decisionProcessor *DecisionProcessor
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

// SetGenerator sets the generator for recursive loop processing
// This is called after both executor and generator are created to resolve circular dependency
func (e *CompositeTaskExecutor) SetGenerator(generator domain.Generator) {
	e.generator = generator
	// Also create and set the decision processor
	e.decisionProcessor = NewDecisionProcessor(generator)
}

// Execute processes a single FieldTask and returns one or more results.
// Multiple results occur when decision points generate additional sibling fields.
func (e *CompositeTaskExecutor) Execute(task *domain.FieldTask, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	// Special case: Check for byte operations first (TTS, Image, STT)
	// This allows STT fields (type: string) to be handled by ByteProcessor
	def := task.Definition()
	if def.TextToSpeech != nil || def.Image != nil || def.SpeechToText != nil {
		// Find ByteProcessor specifically
		for _, processor := range e.processors {
			if byteProc, ok := processor.(*ByteProcessor); ok {
				result, err := byteProc.Process(task, context)
				if err != nil {
					return nil, err
				}
				return []*domain.TaskResult{result}, nil
			}
		}
	}

	var result *domain.TaskResult
	var err error

	// Find appropriate processor by type
	for _, processor := range e.processors {
		if task.Definition().RecursiveLoop != nil {
			// Check for recursive loop - delegate to RecursiveLoopProcessor
			loopProcessor := NewRecursiveLoopProcessor(processor, e.generator, e.decisionProcessor)
			result, err = loopProcessor.Process(task, context)
			if err != nil {
				return nil, err
			}
			break
		}

		if processor.CanProcess(task.Definition().Type) {
			result, err = processor.Process(task, context)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	// Fallback to default if no processor found
	if result == nil {
		result, err = e.defaultProc.Process(task, context)
		if err != nil {
			return nil, err
		}
	}

	// This allows SelectFields in decision branches to reference the generated value
	context.SetGeneratedValue(result.Key(), result.Value())
	log.Printf("[CompositeTaskExecutor] Added original result to context before decision point: %s = %v", result.Key(), result.Value())


	// Process decision point if present and decision processor is available
	if e.decisionProcessor != nil && task.Definition().DecisionPoint != nil {
		log.Printf("[CompositeTaskExecutor] Processing decision point for task %s", task.Key())

		results, err := e.decisionProcessor.ProcessDecisionPoint(task, result, context)
		if err != nil {
			return nil, fmt.Errorf("decision point processing failed: %w", err)
		}

		// Update context with all branch results so subsequent tasks can reference them
		if len(results) > 1 {
			log.Printf("[CompositeTaskExecutor] Got %d additional branch results", len(results)-1)
			for _, res := range results[1:] {
				context.SetGeneratedValue(res.Key(), res.Value())
				log.Printf("[CompositeTaskExecutor] Added branch field to context: %s = %v", res.Key(), res.Value())
			}
		}

		// Return all results (original + branches)
		return results, nil
	}

	// No decision point - return single result
	return []*domain.TaskResult{result}, nil
}

func (e *CompositeTaskExecutor) ExecuteBatch(tasks []*domain.FieldTask, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, 0, len(tasks))

	for _, task := range tasks {
		taskResults, err := e.Execute(task, context)
		if err != nil {
			return nil, fmt.Errorf("task %s failed: %w", task.Key(), err)
		}
		results = append(results, taskResults...)
	}

	return results, nil
}
