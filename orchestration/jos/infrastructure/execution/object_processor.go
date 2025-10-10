package execution

import (
	"fmt"
	"objectGeneration/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

// ObjectProcessor handles object-type fields
type ObjectProcessor struct {
	llmProvider   domain.LLMProvider
	promptBuilder domain.PromptBuilder
	analyzer      domain.SchemaAnalyzer
}

func NewObjectProcessor(
	llmProvider domain.LLMProvider,
	promptBuilder domain.PromptBuilder,
) *ObjectProcessor {
	return &ObjectProcessor{
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
		analyzer:      nil, // Will be set by executor
	}
}

func (p *ObjectProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Object
}

func (p *ObjectProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Extract nested fields from object definition
	nestedFields, err := p.extractFields(task.Definition())
	if err != nil {
		return nil, fmt.Errorf("failed to extract nested fields: %w", err)
	}

	// Create nested tasks
	nestedTasks := make([]*domain.FieldTask, 0, len(nestedFields))
	for key, childDef := range nestedFields {
		nestedTask := domain.NewFieldTask(key, childDef, task)
		nestedTasks = append(nestedTasks, nestedTask)
	}

	// Execute nested tasks recursively
	nestedResults := make(map[string]interface{})
	totalCost := 0.0

	for _, nestedTask := range nestedTasks {
		// Create processor for nested task type
		processor := p.createProcessorForType(nestedTask.Definition().Type)

		result, err := processor.Process(nestedTask, context)
		if err != nil {
			return nil, fmt.Errorf("nested task %s failed: %w", nestedTask.Key(), err)
		}

		nestedResults[nestedTask.Key()] = result.Value()
		totalCost += result.Metadata().Cost
	}

	// Create result with nested object
	metadata := domain.NewResultMetadata()
	metadata.Cost = totalCost

	result := domain.NewTaskResult(task.ID(), task.Key(), nestedResults, metadata)
	return result.WithPath(task.Path()), nil
}

func (p *ObjectProcessor) extractFields(def *jsonSchema.Definition) (map[string]*jsonSchema.Definition, error) {
	if def.Properties == nil {
		return make(map[string]*jsonSchema.Definition), nil
	}

	fields := make(map[string]*jsonSchema.Definition)
	for key, childDef := range def.Properties {
		childDefCopy := childDef
		fields[key] = &childDefCopy
	}

	return fields, nil
}

func (p *ObjectProcessor) createProcessorForType(schemaType jsonSchema.DataType) domain.TypeProcessor {
	switch schemaType {
	case jsonSchema.Object:
		return p // Recursive
	case jsonSchema.Array:
		return NewArrayProcessor(p.llmProvider, p.promptBuilder)
	case jsonSchema.Map:
		return NewMapProcessor(p.llmProvider, p.promptBuilder)
	default:
		return NewPrimitiveProcessor(p.llmProvider, p.promptBuilder)
	}
}
