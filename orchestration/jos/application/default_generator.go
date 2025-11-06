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
package application

import (
	"fmt"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure/epstimic"
	"objectweaver/orchestration/jos/infrastructure/execution"
)

// DefaultGenerator - Main orchestrator for synchronous generation
// Now uses the simpler recursive GenerateObject approach
type DefaultGenerator struct {
	llmProvider    domain.LLMProvider
	promptBuilder  domain.PromptBuilder
	fieldProcessor *execution.FieldProcessor
	plugins        *PluginRegistry
}

func NewDefaultGenerator(
	llmProvider domain.LLMProvider,
	promptBuilder domain.PromptBuilder,
) *DefaultGenerator {
	// Create field processor for recursive generation
	fieldProcessor := execution.NewFieldProcessor(llmProvider, promptBuilder)

	generator := &DefaultGenerator{
		llmProvider:    llmProvider,
		promptBuilder:  promptBuilder,
		fieldProcessor: fieldProcessor,
		plugins:        NewPluginRegistry(),
	}

	// Set generator reference for recursive loops and decision points
	fieldProcessor.SetGenerator(generator)

	// Set up epstimic orchestrator if enabled
	epstimicOrch := epstimic.GetEpstimicOrchestrator(generator)
	if epstimicOrch != nil {
		fieldProcessor.SetEpstimicOrchestrator(epstimicOrch)
	}

	return generator
}

// Generate - Main generation workflow using recursive field processing
func (g *DefaultGenerator) Generate(request *domain.GenerationRequest) (*domain.GenerationResult, error) {
	// Phase 1: Pre-processing plugins
	processedRequest, err := g.plugins.ApplyPreProcessors(request)
	if err != nil {
		return nil, fmt.Errorf("pre-processing failed: %w", err)
	}

	// Phase 2: Check cache
	if cached, found := g.plugins.GetFromCache(generateCacheKey(processedRequest)); found {
		return cached, nil
	}

	// Phase 3: Generate using recursive field processor
	// Create execution context
	context := domain.NewExecutionContext(processedRequest)
	context.PromptContext().AddPrompt(processedRequest.Prompt())

	// Process all fields recursively
	resultsCh := g.fieldProcessor.ProcessFields(processedRequest.Schema(), nil, context)

	// Collect results into map
	data := make(map[string]interface{})

	for result := range resultsCh {
		if result != nil {
			data[result.Key()] = result.Value()
			//here another object needs to be created which is essentially a verbose version of the results data
			//the aim is to provide all the related metadata and choices and embeddings etc for the fields generating in a format which makes sense
			//and means that the end developer can consume it in their own way.
		}
	}

	// Create result metadata
	metadata := domain.NewResultMetadata()

	// Create generation result
	result := domain.NewGenerationResult(data, metadata)

	// Phase 4: Post-processing plugins
	processedResult, err := g.plugins.ApplyPostProcessors(result)
	if err != nil {
		return nil, fmt.Errorf("post-processing failed: %w", err)
	}

	// Phase 5: Validation
	if err := g.plugins.ApplyValidation(processedResult, processedRequest.Schema()); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Phase 6: Cache result
	g.plugins.CacheResult(generateCacheKey(processedRequest), processedResult)

	return processedResult, nil
}

// GenerateStream - Streaming variant (not supported by default generator)
func (g *DefaultGenerator) GenerateStream(request *domain.GenerationRequest) (<-chan *domain.StreamChunk, error) {
	return nil, fmt.Errorf("streaming not supported by DefaultGenerator")
}

// GenerateStreamProgressive - Progressive streaming (not supported by default generator)
func (g *DefaultGenerator) GenerateStreamProgressive(request *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	return nil, fmt.Errorf("progressive streaming not supported by DefaultGenerator")
}

// RegisterPlugin registers a plugin
func (g *DefaultGenerator) RegisterPlugin(plugin domain.Plugin) {
	g.plugins.Register(plugin)
}

func generateCacheKey(request *domain.GenerationRequest) string {
	// Simple cache key generation - could be made more sophisticated
	return fmt.Sprintf("%s_%s", request.Prompt(), request.Schema().Instruction)
}
