// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
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

func (p *PrimitiveProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {

	// Check if Epstimic engine is being used
	//if TRUE - then go into the Epstimic flow
	if task.Definition().Epistemic.Active {
		log.Printf("[PrimitiveProcessor] Epistemic validation is active for field '%s'", task.Key())
		result, _, err := p.epstimicOrchestrator.EpstimicValidation(task, context, p.generateValue)
		if err != nil {
			return nil, fmt.Errorf("failed to validate with Epstimic: %w", err)
		}
		return result.WithPath(task.Path()), nil
	}

	// else - go into the normal flow - below:
	value, metadata, err := p.generateValue(task, context)
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
			log.Printf("[PrimitiveProcessor] Looking for field '%s' in context", fieldPath)
			if value, exists := context.GeneratedValues()[fieldPath]; exists {
				prompt += fmt.Sprintf("\n%s:\n%v\n", fieldPath, value)
				log.Printf("[PrimitiveProcessor] Added field '%s' to prompt (length: %d chars)", fieldPath, len(fmt.Sprintf("%v", value)))
			} else {
				log.Printf("[PrimitiveProcessor] Field '%s' not found in context", fieldPath)
			}
		}
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

	return prompt, config, nil
}

func (p *PrimitiveProcessor) buildVectorRequest(task *domain.FieldTask, context *domain.ExecutionContext) (string, *domain.GenerationConfig, error) {
	// This is a request structure specifically for vector types - the aim is not to use the prompt builder etc
	//the only prompt information being passed is in the prompt information passed from the user information

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

	//from the selected fields generation.
	if len(task.Definition().SelectFields) > 0 {
		prompt := ""
		for _, fieldPath := range task.Definition().SelectFields {
			if value, exists := context.GeneratedValues()[fieldPath]; exists {
				prompt += fmt.Sprintf("%v\n", value)
				log.Printf("[PrimitiveProcessor] Added field '%s' to prompt (length: %d chars)", fieldPath, len(fmt.Sprintf("%v", value)))
			} else {
				log.Printf("[PrimitiveProcessor] Field '%s' not found in context", fieldPath)
			}
		}

		return prompt, config, nil
	}

	//if there are no selected fields - then just return the first prompt from the context
	return context.PromptContext().FirstPrompt(), config, nil
}

func (p *PrimitiveProcessor) generateValue(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
	prompt, config, err := p.buildRequestPieces(task, context)
	if err != nil {
		log.Printf("[TaskExecutor ERROR] Failed to build request pieces for property '%s': %v", task.Key(), err)
		return nil, nil, fmt.Errorf("failed to build request pieces: %w", err)
	}

	response, metadata, err := p.llmProvider.Generate(prompt, config)
	if err != nil {
		log.Printf("[TaskExecutor ERROR] Generation failed for property '%s': %v", task.Key(), err)
		return nil, nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	log.Printf("[TaskExecutor] Received response for property '%s', parsing as %s",
		task.Key(), task.Definition().Type)

	// Parse response based on type
	value := p.parseValue(response, task.Definition().Type)
	metadata.Prompt = prompt

	log.Printf("[TaskExecutor] Parsed value for property '%s': %+v", task.Key(), value)
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
