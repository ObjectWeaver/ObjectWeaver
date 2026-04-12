package application

import (
	"context"
	"fmt"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/infrastructure/epstimic"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/infrastructure/execution"
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

		// Create context for cancellation support
		ctx := context.Background()
		// TODO: In future, accept context from caller for proper deadline/cancellation propagation

		// Pre-processing
		processedRequest, err := g.plugins.ApplyPreProcessors(request)
		if err != nil {
			return
		}

		// Create execution context with worker pool
		execContext := domain.NewExecutionContext(processedRequest)
		workerPool := execution.NewWorkerPool(-1)
		execContext.SetWorkerPool(workerPool)
		execContext.PromptContext().AddPrompt(processedRequest.Prompt())

		// Process all fields recursively using FieldProcessor
		resultsCh := g.fieldProcessor.ProcessFieldsStart(ctx, processedRequest.Schema(), nil, execContext)

		// Stream results as they come in
		for results := range resultsCh {
			for _, result := range results {
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
