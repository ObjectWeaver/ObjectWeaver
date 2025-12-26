package execution

import (
	"context"
	"fmt"
	"objectweaver/logger"
	"objectweaver/orchestration/extractor"
	"objectweaver/orchestration/jos/domain"
	"os"
	"strings"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// PrimitiveProcessor handles primitive types (string, number, boolean, byte)
type PrimitiveProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	systemPromptProvider SystemPromptProvider
	maxRetries           int
	numberExtractor      extractor.PrimitiveExtractor[int]
	generator            domain.Generator
	epstimicOrchestrator EpstimicOrchestrator
}

func NewPrimitiveProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *PrimitiveProcessor {
	return &PrimitiveProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: NewDefaultSystemPromptProvider(),
		maxRetries:           3,
		numberExtractor:      extractor.NewIntegerExtractor(),
	}
}

func NewPrimitiveProcessorWithPromptProvider(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder, promptProvider SystemPromptProvider) *PrimitiveProcessor {
	return &PrimitiveProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: promptProvider,
		maxRetries:           3,
		numberExtractor:      extractor.NewIntegerExtractor(),
	}
}

// SetEpstimicOrchestrator sets the epstimic orchestrator for validation
func (p *PrimitiveProcessor) SetEpstimicOrchestrator(orchestrator EpstimicOrchestrator) {
	p.epstimicOrchestrator = orchestrator
}

func (p *PrimitiveProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.String ||
		schemaType == jsonSchema.Number ||
		schemaType == jsonSchema.Integer ||
		schemaType == jsonSchema.Boolean
	// Note: jsonSchema.Byte is handled by ByteProcessor, not PrimitiveProcessor
}

func (p *PrimitiveProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	// Check if Epstimic engine is being used
	//if TRUE - then go into the Epstimic flow
	if task.Definition().Epistemic.Active {
		logger.Printf("[PrimitiveProcessor] Epistemic validation is active for field '%s'", task.Key())
		// Wrap generateValue to match expected signature
		generateFn := func(t *domain.FieldTask, ec *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
			return p.generateValue(ctx, t, ec)
		}
		result, _, err := p.epstimicOrchestrator.EpstimicValidation(task, execContext, generateFn)
		if err != nil {
			return nil, fmt.Errorf("failed to validate with Epstimic: %w", err)
		}
		return result.WithPath(task.Path()), nil
	}

	// else - go into the normal flow - below:
	value, metadata, err := p.generateValue(ctx, task, execContext)
	if err != nil {
		return nil, fmt.Errorf("failed to generate value: %w", err)
	}

	// Create result
	resultMetadata := domain.NewResultMetadata()
	resultMetadata.Cost = metadata.Cost
	resultMetadata.TokensUsed = metadata.TokensUsed
	resultMetadata.ModelUsed = metadata.Model

	result := domain.NewTaskResult(task.ID(), task.Key(), value, resultMetadata)
	return result.WithPath(task.Path()), nil

}

func (p *PrimitiveProcessor) buildRequestPieces(task *domain.FieldTask, context *domain.ExecutionContext) (string, *domain.GenerationConfig, error) {
	if task.Definition().Type == jsonSchema.Vector {
		return p.buildVectorRequest(task, context)
	}

	// Build prompt
	prompt, err := p.promptBuilder.Build(task, context.PromptContext())
	if err != nil {
		return "", nil, err
	}

	//TODO - for the vector type there will need to be a different handling here where the selected field becomes the entire prompt. With the aim that the vector embedding is created from that field only.

	// Enhance prompt with SelectFields if specified
	if len(task.Definition().SelectFields) > 0 {
		// Add selected field values directly to the prompt
		prompt += "\n\nContext from previous generation:\n"
		for _, fieldPath := range task.Definition().SelectFields {
			if value, exists := ResolveFieldPath(fieldPath, context.GeneratedValues()); exists {
				formattedValue := FormatFieldValue(value)
				prompt += fmt.Sprintf("\n%s:\n%s\n", fieldPath, formattedValue)
			}
		}
	}

	// Generate with LLM
	// Copy config to avoid race condition when multiple goroutines modify it
	sharedConfig := context.GenerationConfig()
	config := &domain.GenerationConfig{}
	if sharedConfig != nil {
		*config = *sharedConfig
	}
	config.Model = p.determineModel(task.Definition())
	config.Definition = task.Definition() // Pass the full definition for SendImage support

	// Set system prompt with priority: definition-level > provider-level
	if task.Definition().SystemPrompt != nil {
		// Use definition-level system prompt if provided
		config.SystemPrompt = *task.Definition().SystemPrompt
	} else if p.systemPromptProvider != nil {
		// Otherwise use provider's system prompt for this type
		if systemPrompt := p.systemPromptProvider.GetSystemPrompt(task.Definition().Type); systemPrompt != nil {
			config.SystemPrompt = *systemPrompt
		}
	}

	return prompt, config, nil
}

func (p *PrimitiveProcessor) buildVectorRequest(task *domain.FieldTask, context *domain.ExecutionContext) (string, *domain.GenerationConfig, error) {
	// This is a request structure specifically for vector types - the aim is not to use the prompt builder etc
	//the only prompt information being passed is in the prompt information passed from the user information

	// Copy config to avoid race condition when multiple goroutines modify it
	sharedConfig := context.GenerationConfig()
	config := &domain.GenerationConfig{}
	if sharedConfig != nil {
		*config = *sharedConfig
	}
	config.Model = p.determineModel(task.Definition())
	config.Definition = task.Definition() // Pass the full definition for SendImage support

	// Set system prompt with priority: definition-level > provider-level
	if task.Definition().SystemPrompt != nil {
		// Use definition-level system prompt if provided
		config.SystemPrompt = *task.Definition().SystemPrompt
	} else if p.systemPromptProvider != nil {
		// Otherwise use provider's system prompt for this type
		if systemPrompt := p.systemPromptProvider.GetSystemPrompt(task.Definition().Type); systemPrompt != nil {
			config.SystemPrompt = *systemPrompt
		}
	}

	//from the selected fields generation.
	if len(task.Definition().SelectFields) > 0 {
		prompt := ""
		for _, fieldPath := range task.Definition().SelectFields {
			if value, exists := ResolveFieldPath(fieldPath, context.GeneratedValues()); exists {
				formattedValue := FormatFieldValue(value)
				prompt += fmt.Sprintf("%s\n", formattedValue)
			}
		}

		return prompt, config, nil
	}

	//if there are no selected fields - then just return the first prompt from the context
	return context.PromptContext().FirstPrompt(), config, nil
}

func (p *PrimitiveProcessor) generateValue(ctx context.Context, task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
	prompt, config, err := p.buildRequestPieces(task, context)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request pieces: %w", err)
	}

	// Add context for cancellation support
	config.Context = ctx

	response, metadata, err := p.llmProvider.Generate(prompt, config)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse response based on type
	value := p.parseValue(response, task.Definition().Type)
	metadata.Prompt = prompt

	return value, metadata, nil
}

func (p *PrimitiveProcessor) parseValue(response any, fieldType jsonSchema.DataType) interface{} {
	// Clean response
	switch fieldType {
	case jsonSchema.Boolean:
		response = cleanResponse(response.(string))
		return response == "true" || response == "True" || response == "TRUE"
	case jsonSchema.Number, jsonSchema.Integer:
		// Parse number - simplified
		response = cleanResponse(response.(string))
		num, err := p.numberExtractor.Extract(response.(string))
		if err != nil {
			return 0
		}
		return num
	case jsonSchema.String:
		response = cleanResponse(response.(string))
		return response
	case jsonSchema.Vector:
		// Convert []float32 to []interface{} for protobuf compatibility
		if vec, ok := response.([]float32); ok {
			result := make([]interface{}, len(vec))
			for i, v := range vec {
				result[i] = float64(v)
			}
			return result
		}
		return response
	default:
		return response
	}
}

func (p *PrimitiveProcessor) determineModel(def *jsonSchema.Definition) string {
	if def.Model != "" {
		return def.Model
	}
	// Use provider-aware default model
	return getDefaultModelForProvider()
}

// getDefaultModelForProvider returns the appropriate default model based on LLM_PROVIDER
func getDefaultModelForProvider() string {
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))

	switch provider {
	case "gemini":
		return "gemini-2.0-flash"
	case "openai":
		return "gpt-4o-mini"
	case "local":
		return "gpt-4o-mini"
	default:
		// Auto-detect based on available configuration
		if os.Getenv("LLM_API_URL") != "" {
			return "gpt-4o-mini"
		}
		if os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("LLM_API_KEY") != "" {
			return "gemini-2.0-flash"
		}
		// Final fallback
		return "gpt-4o-mini"
	}
}

func cleanResponse(response any) string {
	// Remove common artifacts
	responseStr := response.(string)

	responseStr = trimQuotes(responseStr)
	responseStr = trimWhitespace(responseStr)
	return responseStr
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func trimWhitespace(s string) string {
	// Simplified trim
	return s
}
