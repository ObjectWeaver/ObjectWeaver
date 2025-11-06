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
// <https://objectweaver.dev/licensing/server-side-public-license>.
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
	if os.Getenv("USE_EPSTIMIC_ENGINE") == "true" && task.Definition().Epistemic.Active && p.epstimicOrchestrator != nil {
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
	// Build prompt
	prompt, err := p.promptBuilder.Build(task, context.PromptContext())
	if err != nil {
		return "", nil, err
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
