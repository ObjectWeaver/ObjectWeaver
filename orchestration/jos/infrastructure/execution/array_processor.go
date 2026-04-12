package execution

import (
	"context"
	"fmt"
	"objectweaver/logger"
	"objectweaver/orchestration/jos/domain"
	"strings"

	"objectweaver/jsonSchema"

	"golang.org/x/sync/semaphore"
)

// global semaphore to limit concurrent array item processing
// this prevents goroutine explosion and http client contention
var arrayItemSemaphore = semaphore.NewWeighted(50000)

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

func (p *ArrayProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	arraySize, listString, err := p.determineArraySize(task, execContext)
	if err != nil {
		logger.Printf("Warning: failed to determine array size, using default: %v\n", err)
	}

	itemDef := task.Definition().Items
	if itemDef == nil {
		return nil, fmt.Errorf("array items definition is nil")
	}

	totalCost := 0.0

	enhancedContext := execContext
	if listString != "" {
		enhancedContext = p.createEnhancedContext(execContext, listString)
	}

	type itemResult struct {
		index int
		value interface{}
		cost  float64
		err   error
	}

	resultCh := make(chan *itemResult, arraySize)

	for index := 0; index < arraySize; index++ {
		processItemFunc := func(idx int) func() {
			return func() {
				itemTask := domain.NewFieldTask(fmt.Sprintf("%s[%d]", task.Key(), idx), itemDef, task)

				// create isolated context for this array item with item-specific prompt context
				var itemContext *domain.ExecutionContext
				if listString != "" {
					specificItem := p.extractItemFromList(listString, idx)
					if specificItem != "" {
						itemSpecificContext := fmt.Sprintf("Generate details for: %s", specificItem)
						itemContext = enhancedContext.WithItemContext(itemTask, itemSpecificContext)
					} else {
						itemContext = enhancedContext.WithParent(itemTask)
					}
				} else {
					// No list string, use standard isolation
					itemContext = enhancedContext.WithParent(itemTask)
				}

				var result *domain.TaskResult
				var err error

				if itemDef.Type == jsonSchema.Object && p.fieldProcessor != nil {
					// for object items in arrays, we need to collect results to build the array item
					// processfieldsstart returns a channel and processes in background
					resultsCh := p.fieldProcessor.ProcessFieldsStart(ctx, itemDef, itemTask, itemContext)

					// collect nested object results and aggregate metadata
					nestedResults := make(map[string]interface{})
					aggregatedMetadata := domain.NewResultMetadata()
					for nestedResultSet := range resultsCh {
						for _, r := range nestedResultSet {
							if r != nil {
								nestedResults[r.Key()] = r.Value()
								// Aggregate metadata from nested fields
								if r.Metadata() != nil {
									aggregatedMetadata.AddCost(r.Metadata().Cost)
									aggregatedMetadata.AddTokens(r.Metadata().TokensUsed)
								}
							}
						}
					}

					// create a result for this object item with aggregated metadata
					result = domain.NewTaskResult(itemTask.ID(), itemTask.Key(), nestedResults, aggregatedMetadata)
					result = result.WithPath(itemTask.Path())
				} else {
					processor := p.createProcessorForType(itemDef.Type)
					result, err = processor.Process(ctx, itemTask, itemContext)
					if err != nil {
						select {
						case resultCh <- &itemResult{
							index: idx,
							err:   fmt.Errorf("array item %d failed: %w", idx, err),
						}:
						case <-ctx.Done():
						}
						return
					}
				}

				select {
				case resultCh <- &itemResult{
					index: idx,
					value: result.Value(),
					cost:  result.Metadata().Cost,
					err:   nil,
				}:
				case <-ctx.Done():
				}
			}
		}

		// acquire semaphore to limit concurrency
		if err := arrayItemSemaphore.Acquire(ctx, 1); err != nil {
			select {
			case resultCh <- &itemResult{
				index: index,
				err:   fmt.Errorf("failed to acquire semaphore for item %d: %w", index, err),
			}:
			case <-ctx.Done():
			}
			return nil, ctx.Err()
		}

		go func(idx int) {
			defer arrayItemSemaphore.Release(1)
			processItemFunc(idx)()
		}(index)
	}

	collectedResults := make([]interface{}, arraySize)
	for i := 0; i < arraySize; i++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while collecting results: %w", ctx.Err())
		case item := <-resultCh:
			if item.err != nil {
				return nil, item.err
			}
			collectedResults[item.index] = item.value
			totalCost += item.cost
		}
	}
	close(resultCh)

	metadata := domain.NewResultMetadata()
	metadata.Cost = totalCost

	result := domain.NewTaskResult(task.ID(), task.Key(), collectedResults, metadata)
	return result.WithPath(task.Path()), nil
}

func (p *ArrayProcessor) createEnhancedContext(context *domain.ExecutionContext, listString string) *domain.ExecutionContext {
	// create a new prompt context with the list information
	// this avoids mutating the original context
	enhancedPromptCtx := context.PromptContext()
	listPrompt := fmt.Sprintf("\n\nGeneral information:\n%s\n\nPlease continue processing items from this list.\n", listString)
	enhancedPromptCtx.AddPrompt(listPrompt)

	return context
}

func (p *ArrayProcessor) determineArraySize(task *domain.FieldTask, context *domain.ExecutionContext) (int, string, error) {
	listDef := p.createListExtractionDefinition(task.Key(), task.Definition())

	listTask := domain.NewFieldTask("listInfo", listDef, task)

	result, err := p.executeLLMRequest(listTask, context)
	if err != nil {
		return 3, "", err
	}

	numItems, listString := p.extractListInfo(result)

	return numItems, listString, nil
}

func (p *ArrayProcessor) createListExtractionDefinition(key string, arrayDef *jsonSchema.Definition) *jsonSchema.Definition {
	systemPrompt := fmt.Sprintf("You are a list extracting expert who returns a list of values which relate to the %s. You always return a list of values as numbered bullet-pointed list.", key)
	numberSystemPrompt := "You are an expert in extracting the number of items in the bullet point list. Return only a whole number."

	temp := float32(0.0)
	model := arrayDef.Model

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
		Properties: map[string]jsonSchema.Definition{
			"numItems": {
				Type:         jsonSchema.Number,
				Model:        model,
				SystemPrompt: &numberSystemPrompt,
				NarrowFocus: &jsonSchema.Focus{
					Prompt: "Extract the number of items in the bullet point list. Return only a whole number.",
					Fields: []string{"listString"},
				},
				ModelConfig: &jsonSchema.ModelConfig{Temperature: temp},
			},
			"listString": {
				Type:        jsonSchema.String,
				Model:       model,
				Instruction: fmt.Sprintf("Return a numbered bullet point list of unique items. Which directly relate to %s.", key),
				ModelConfig: &jsonSchema.ModelConfig{Temperature: temp},
			},
		},
	}
}

/*
this format of request is essentially begging the LLM to return the format in a set format. This has been done to increase the speed of processing requests as the array size and topic generation was a massive bottle neck.
given the simplicity of the task. Most LLMs after GPT4 should be able to handle this without dying.
*/
func (p *ArrayProcessor) executeLLMRequest(task *domain.FieldTask, context *domain.ExecutionContext) (map[string]interface{}, error) {

	listDef := task.Definition()
	combinedPrompt := `Extract a numbered bullet point list and count the items.

Instructions:
1. First, create a numbered list of unique items (format: "1. item\n2. item\n...")
2. Then, count the total number of items

Return your response in this exact format:
LIST:
[your numbered list here]
COUNT: [number]`

	listStringTask := domain.NewFieldTask("combined", listDef, task)

	prompt, err := p.promptBuilder.Build(listStringTask, context.PromptContext())
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// override prompt with combined instruction
	prompt = combinedPrompt + "\n\n" + prompt

	// Create a local copy of the config to avoid race conditions and ensure correct system prompt
	sharedConfig := context.GenerationConfig()
	generationConfig := &domain.GenerationConfig{}
	if sharedConfig != nil {
		*generationConfig = *sharedConfig
	}
	generationConfig.Definition = listDef

	// Set system prompt from definition if available
	if listDef.SystemPrompt != nil {
		generationConfig.SystemPrompt = *listDef.SystemPrompt
	}

	response, _, err := p.llmProvider.Generate(prompt, generationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate combined response: %w", err)
	}

	// parse combined response
	responseStr, ok := response.(string)
	if !ok {
		responseStr = fmt.Sprintf("%v", response)
	}

	// Handle literal \n strings if they appear
	responseStr = strings.ReplaceAll(responseStr, "\\n", "\n")

	var listString string
	var numItems int = -1 // Use -1 to indicate not found

	// split response into list and count sections
	lines := strings.Split(responseStr, "\n")
	inList := false
	var listLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToUpper(trimmed), "LIST:") {
			inList = true
			// Check if there's content after "LIST:" on the same line
			content := strings.TrimSpace(line[len("LIST:"):])
			if content != "" {
				listLines = append(listLines, content)
			}
			continue
		}

		if strings.HasPrefix(strings.ToUpper(trimmed), "COUNT:") {
			inList = false
			// Extract count
			countStr := strings.TrimSpace(line[len("COUNT:"):])
			if _, err := fmt.Sscanf(countStr, "%d", &numItems); err != nil {
				numItems = -1
			}
			continue
		}

		if inList && trimmed != "" {
			listLines = append(listLines, line)
		}
	}

	listString = strings.Join(listLines, "\n")

	// fallback: if parsing failed, count numbered items manually
	if numItems == -1 || (numItems > 0 && listString == "") {
		manualCount := 0
		for _, line := range strings.Split(responseStr, "\n") {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9' {
				manualCount++
			}
		}

		if manualCount > 0 {
			numItems = manualCount
		} else if numItems == -1 {
			// Only default to 3 if we couldn't find anything
			numItems = 3
		}

		if listString == "" {
			listString = responseStr
		}
	}

	if numItems < 0 {
		numItems = 0
	}

	if numItems > 100 {
		numItems = 100
	}

	result := make(map[string]interface{})
	result["numItems"] = numItems
	result["listString"] = listString

	return result, nil
}

func (p *ArrayProcessor) extractListInfo(result map[string]interface{}) (int, string) {
	numItems := 0
	listString := ""

	if val, ok := result["numItems"]; ok {
		switch v := val.(type) {
		case int:
			numItems = v
		case float64:
			numItems = int(v)
		case string:
			fmt.Sscanf(v, "%d", &numItems)
		}
	}

	if val, ok := result["listString"]; ok {
		if str, ok := val.(string); ok {
			listString = str
		}
	}

	if numItems < 0 {
		numItems = 0
	}
	if numItems > 100 {
		numItems = 100
	}

	return numItems, listString
}

// extractitemfromlist extracts the specific item at the given index from a numbered list
// handles formats like: "1. item one\n2. item two" or with extra spaces "1.  item"
// also handles lists with preamble text
func (p *ArrayProcessor) extractItemFromList(listString string, index int) string {
	if listString == "" {
		return ""
	}

	// Split by newlines
	lines := []string{}
	currentLine := ""
	for _, char := range listString {
		if char == '\n' {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = ""
		} else {
			currentLine += string(char)
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	// Find the line that matches this index (1-based in the list)
	targetNumber := index + 1
	for _, line := range lines {
		// Trim leading/trailing whitespace
		trimmedLine := ""
		startIdx := 0
		endIdx := len(line)

		// Find first non-whitespace
		for startIdx < len(line) && (line[startIdx] == ' ' || line[startIdx] == '\t') {
			startIdx++
		}
		// Find last non-whitespace
		for endIdx > startIdx && (line[endIdx-1] == ' ' || line[endIdx-1] == '\t' || line[endIdx-1] == '\r') {
			endIdx--
		}

		if startIdx < endIdx {
			trimmedLine = line[startIdx:endIdx]
		} else {
			continue
		}

		// Check if line starts with the target number followed by dot
		if len(trimmedLine) < 2 {
			continue
		}

		// Parse the number at the start
		numStr := ""
		dotIdx := -1
		for i, ch := range trimmedLine {
			if ch >= '0' && ch <= '9' {
				numStr += string(ch)
			} else if ch == '.' && len(numStr) > 0 {
				dotIdx = i
				break
			} else {
				break
			}
		}

		if dotIdx == -1 || numStr == "" {
			continue
		}

		parsedNum := 0
		fmt.Sscanf(numStr, "%d", &parsedNum)

		if parsedNum == targetNumber {
			// Extract content after the dot
			content := trimmedLine[dotIdx+1:]
			// Trim leading whitespace from content
			contentStart := 0
			for contentStart < len(content) && (content[contentStart] == ' ' || content[contentStart] == '\t') {
				contentStart++
			}
			if contentStart < len(content) {
				return content[contentStart:]
			}
		}
	}

	return ""
}

func (p *ArrayProcessor) createProcessorForType(schemaType jsonSchema.DataType) domain.TypeProcessor {
	if p.fieldProcessor != nil {
		return p.fieldProcessor.getBaseProcessorForType(schemaType)
	}

	switch schemaType {
	case jsonSchema.Array:
		return p
	default:
		return NewPrimitiveProcessor(p.llmProvider, p.promptBuilder)
	}
}
