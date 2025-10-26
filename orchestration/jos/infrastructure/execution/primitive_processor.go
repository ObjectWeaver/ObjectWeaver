package execution

import (
	"fmt"
	"log"
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

func (p *PrimitiveProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.String ||
		schemaType == jsonSchema.Number ||
		schemaType == jsonSchema.Integer ||
		schemaType == jsonSchema.Boolean
	// Note: jsonSchema.Byte is handled by ByteProcessor, not PrimitiveProcessor
}

func (p *PrimitiveProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Build prompt
	prompt, err := p.promptBuilder.Build(task, context.PromptContext())
	if err != nil {
		return nil, fmt.Errorf("prompt building failed: %w", err)
	}

	// Generate with LLM
	config := context.GenerationConfig()
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

	// Log task processing details
	log.Printf("[TaskExecutor] Processing %s property '%s' with model %s",
		task.Definition().Type, task.Key(), config.Model)

	response, metadata, err := p.llmProvider.Generate(prompt, config)
	if err != nil {
		log.Printf("[TaskExecutor ERROR] Generation failed for property '%s': %v", task.Key(), err)
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	log.Printf("[TaskExecutor] Received response for property '%s', parsing as %s",
		task.Key(), task.Definition().Type)

	// Parse response based on type
	value := p.parseValue(response, task.Definition().Type)

	log.Printf("[TaskExecutor] Parsed value for property '%s': %+v", task.Key(), value)

	// Create result
	resultMetadata := domain.NewResultMetadata()
	resultMetadata.Cost = metadata.Cost
	resultMetadata.TokensUsed = metadata.TokensUsed
	resultMetadata.ModelUsed = metadata.Model

	result := domain.NewTaskResult(task.ID(), task.Key(), value, resultMetadata)
	return result.WithPath(task.Path()), nil
}

func (p *PrimitiveProcessor) parseValue(response string, fieldType jsonSchema.DataType) interface{} {
	// Clean response
	response = cleanResponse(response)

	switch fieldType {
	case jsonSchema.Boolean:
		return response == "true" || response == "True" || response == "TRUE"
	case jsonSchema.Number, jsonSchema.Integer:
		// Parse number - simplified
		num, err := p.numberExtractor.Extract(response)
		if err != nil {
			return 0
		}
		return num
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

func cleanResponse(response string) string {
	// Remove common artifacts
	response = trimQuotes(response)
	response = trimWhitespace(response)
	return response
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
