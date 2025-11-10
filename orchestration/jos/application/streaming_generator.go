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
package application

import (
	"fmt"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure/epstimic"
	"objectweaver/orchestration/jos/infrastructure/execution"
)

// StreamingGenerator - Generator with field-level streaming support
// Now uses the same FieldProcessor as DefaultGenerator but streams results
type StreamingGenerator struct {
	llmProvider    domain.LLMProvider
	promptBuilder  domain.PromptBuilder
	fieldProcessor *execution.FieldProcessor
	plugins        *PluginRegistry
}

func NewStreamingGenerator(
	llmProvider domain.LLMProvider,
	promptBuilder domain.PromptBuilder,
) *StreamingGenerator {
	// Create field processor for recursive generation
	fieldProcessor := execution.NewFieldProcessor(llmProvider, promptBuilder)

	generator := &StreamingGenerator{
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

// Generate - Synchronous generation (falls back to collecting stream)
func (g *StreamingGenerator) Generate(request *domain.GenerationRequest) (*domain.GenerationResult, error) {
	stream, err := g.GenerateStream(request)
	if err != nil {
		return nil, err
	}

	// Collect all chunks
	finalData := make(map[string]interface{})

	for chunk := range stream {
		if !chunk.IsFinal {
			finalData[chunk.Key] = chunk.Value
		}
	}

	metadata := domain.NewResultMetadata()

	return domain.NewGenerationResult(finalData, metadata), nil
}

// GenerateStream - Stream results as fields complete
func (g *StreamingGenerator) GenerateStream(request *domain.GenerationRequest) (<-chan *domain.StreamChunk, error) {
	out := make(chan *domain.StreamChunk, 100)

	go func() {
		defer close(out)

		// Pre-processing
		processedRequest, err := g.plugins.ApplyPreProcessors(request)
		if err != nil {
			return
		}

		// Create execution context
		context := domain.NewExecutionContext(processedRequest)
		context.PromptContext().AddPrompt(processedRequest.Prompt())

		// Process all fields recursively using FieldProcessor
		resultsCh := g.fieldProcessor.ProcessFields(processedRequest.Schema(), nil, context)

		// Stream results as they come in
		for result := range resultsCh {
			if result != nil {
				// Convert TaskResult to StreamChunk
				chunk := &domain.StreamChunk{
					Key:     result.Key(),
					Value:   result.Value(),
					Path:    result.Path(),
					IsFinal: false,
				}
				out <- chunk
			}
		}

		// Send final chunk
		out <- &domain.StreamChunk{
			IsFinal: true,
		}
	}()

	return out, nil
}

// GenerateStreamProgressive - Not supported
func (g *StreamingGenerator) GenerateStreamProgressive(request *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	return nil, fmt.Errorf("progressive streaming not supported by StreamingGenerator")
}

// RegisterPlugin registers a plugin
func (g *StreamingGenerator) RegisterPlugin(plugin domain.Plugin) {
	g.plugins.Register(plugin)
}
