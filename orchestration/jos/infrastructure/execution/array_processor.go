package execution

import (
	"fmt"
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// ArrayProcessor handles array-type fields
type ArrayProcessor struct {
	llmProvider    domain.LLMProvider
	promptBuilder  domain.PromptBuilder
	fieldProcessor *FieldProcessor
}

func NewArrayProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *ArrayProcessor {
	return &ArrayProcessor{
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
	}
}

func NewArrayProcessorWithFieldProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder, fieldProcessor *FieldProcessor) *ArrayProcessor {
	return &ArrayProcessor{
		llmProvider:    llmProvider,
		promptBuilder:  promptBuilder,
		fieldProcessor: fieldProcessor,
	}
}

func (p *ArrayProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Array
}

func (p *ArrayProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Determine array size using LLM
	arraySize, listString, err := p.determineArraySize(task, context)
	if err != nil {
		// Log error but continue with default size
		fmt.Printf("Warning: failed to determine array size, using default: %v\n", err)
	}

	// Get item definition
	itemDef := task.Definition().Items
	if itemDef == nil {
		return nil, fmt.Errorf("array items definition is nil")
	}

	// Generate array items
	items := make([]interface{}, 0, arraySize)
	totalCost := 0.0

	// Create an enhanced context with the list information if available
	enhancedContext := context
	if listString != "" {
		enhancedContext = p.createEnhancedContext(context, listString)
	}

	for i := 0; i < arraySize; i++ {
		itemTask := domain.NewFieldTask(fmt.Sprintf("%s[%d]", task.Key(), i), itemDef, task)

		// Handle object items specially - they need recursive field processing
		var result *domain.TaskResult
		var err error

		if itemDef.Type == jsonSchema.Object && p.fieldProcessor != nil {
			// Object items use FieldProcessor for recursive processing
			results := p.fieldProcessor.processObjectField(itemTask, enhancedContext)
			if len(results) > 0 {
				result = results[0] // Object fields return single result
			} else {
				return nil, fmt.Errorf("array item %d (object) failed: no result", i)
			}
		} else {
			// Non-object items use type processors
			processor := p.createProcessorForType(itemDef.Type)
			result, err = processor.Process(itemTask, enhancedContext)
			if err != nil {
				return nil, fmt.Errorf("array item %d failed: %w", i, err)
			}
		}

		items = append(items, result.Value())
		totalCost += result.Metadata().Cost
	}

	metadata := domain.NewResultMetadata()
	metadata.Cost = totalCost

	result := domain.NewTaskResult(task.ID(), task.Key(), items, metadata)
	return result.WithPath(task.Path()), nil
}

// createEnhancedContext adds the list information to the context
func (p *ArrayProcessor) createEnhancedContext(context *domain.ExecutionContext, listString string) *domain.ExecutionContext {
	// Add list information to the prompt context
	promptCtx := context.PromptContext()
	listPrompt := fmt.Sprintf("\n\nGeneral information:\n%s\n\nPlease continue processing items from this list.\n", listString)
	promptCtx.AddPrompt(listPrompt)
	return context
}

func (p *ArrayProcessor) determineArraySize(task *domain.FieldTask, context *domain.ExecutionContext) (int, string, error) {
	// Create a structured object to extract array size and list items
	// This follows the V1 pattern from agentListCreator.go
	listDef := p.createListExtractionDefinition(task.Key(), task.Definition())

	// Create a temporary task for the list extraction
	listTask := domain.NewFieldTask("listInfo", listDef, task)

	// Execute the LLM request to get list information
	result, err := p.executeLLMRequest(listTask, context)
	if err != nil {
		// Fallback to default size
		return 3, "", err
	}

	// Extract number of items and list string from result
	numItems, listString := p.extractListInfo(result)

	return numItems, listString, nil
}

// createListExtractionDefinition creates a definition for extracting array size
// Matches the pattern from V1's agentListCreator.go
func (p *ArrayProcessor) createListExtractionDefinition(key string, arrayDef *jsonSchema.Definition) *jsonSchema.Definition {
	systemPrompt := fmt.Sprintf("You are a list extracting expert who returns a list of values which relate to the %s. You always return a list of values as numbered bullet-pointed list.", key)
	numberSystemPrompt := "You are an expert in extracting the number of items in the bullet point list. Return only a whole number."

	temp := 0.0
	model := arrayDef.Model
	// Model will use whatever is set in the array definition, or default from config

	instruction := arrayDef.Instruction
	if instruction == "" {
		instruction = fmt.Sprintf("With the information provided below, please extract the number of unique %s values. Then, return a numbered bullet-pointed list of the unique items that have been found.", key)
	}

	return &jsonSchema.Definition{
		Type:            jsonSchema.Object,
		Instruction:     instruction,
		Model:           model,
		SystemPrompt:    &systemPrompt,
		ProcessingOrder: []string{"listString", "numItems"},
		Temp:            temp,
		Properties: map[string]jsonSchema.Definition{
			"numItems": {
				Type:         jsonSchema.Number,
				Model:        model,
				SystemPrompt: &numberSystemPrompt,
				NarrowFocus: &jsonSchema.Focus{
					Prompt: "Extract the number of items in the bullet point list. Return only a whole number.",
					Fields: []string{"listString"},
				},
				Temp: temp,
			},
			"listString": {
				Type:        jsonSchema.String,
				Model:       model,
				Instruction: fmt.Sprintf("Return a numbered bullet point list of unique items. Which directly relate to %s.", key),
				Temp:        temp,
			},
		},
	}
}

// executeLLMRequest executes the LLM request to extract list information
func (p *ArrayProcessor) executeLLMRequest(task *domain.FieldTask, context *domain.ExecutionContext) (map[string]interface{}, error) {
	// First, process the listString field
	listStringDef := task.Definition().Properties["listString"]
	listStringTask := domain.NewFieldTask("listString", &listStringDef, task)

	// Build prompt for list string extraction
	prompt, err := p.promptBuilder.Build(listStringTask, context.PromptContext())
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Call LLM provider to get the list string
	generationConfig := context.GenerationConfig()
	listString, metadata, err := p.llmProvider.Generate(prompt, generationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate list string: %w", err)
	}

	// Now process the numItems field with narrow focus on listString
	numItemsDef := task.Definition().Properties["numItems"]
	numItemsTask := domain.NewFieldTask("numItems", &numItemsDef, task)

	// Create a new context with the listString result
	enhancedCtx := context.PromptContext()
	enhancedCtx.CurrentGen = fmt.Sprintf("listString: %s", listString)

	// Build prompt for number extraction
	numPrompt, err := p.promptBuilder.Build(numItemsTask, enhancedCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build number prompt: %w", err)
	}

	// Call LLM to extract the number
	numResponse, _, err := p.llmProvider.Generate(numPrompt, generationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate num items: %w", err)
	}

	// Parse the response to extract the number
	var numItems int
	fmt.Sscanf(numResponse, "%d", &numItems)
	if numItems < 1 {
		numItems = 3 // Default number of items to be in the list. This was chosen as LLMs like to do things in three's like people
	}

	// Return the results
	result := make(map[string]interface{})
	result["numItems"] = numItems
	result["listString"] = listString

	// Update cost tracking
	_ = metadata // TODO: Track this in context

	return result, nil
}

// extractListInfo extracts the number of items and list string from the result
func (p *ArrayProcessor) extractListInfo(result map[string]interface{}) (int, string) {
	numItems := 3 // Default
	listString := ""

	if val, ok := result["numItems"]; ok {
		switch v := val.(type) {
		case int:
			numItems = v
		case float64:
			numItems = int(v)
		case string:
			// Try to parse string as number
			fmt.Sscanf(v, "%d", &numItems)
		}
	}

	if val, ok := result["listString"]; ok {
		if str, ok := val.(string); ok {
			listString = str
		}
	}

	// Ensure reasonable bounds - to ensure that the LLMs don't go crazy. The weaker ones do this. If you want to have a long list of items This is probably better to break it down
	if numItems < 1 {
		numItems = 1
	}
	if numItems > 100 {
		numItems = 100
	}

	return numItems, listString
}

func (p *ArrayProcessor) createProcessorForType(schemaType jsonSchema.DataType) domain.TypeProcessor {
	// If we have a field processor, use it for recursive processing
	if p.fieldProcessor != nil {
		return p.fieldProcessor.getBaseProcessorForType(schemaType)
	}

	// Fallback: We should always have a field processor now, but just in case
	// Note: Objects should never reach this point as they're handled specially in Process()
	switch schemaType {
	case jsonSchema.Array:
		return p // Recursive
	default:
		return NewPrimitiveProcessor(p.llmProvider, p.promptBuilder)
	}
}
