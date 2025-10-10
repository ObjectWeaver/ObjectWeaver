package application

import (
	"objectweaver/orchestration/jos/domain"
)

// ProgressiveGenerator - Generator with token-level streaming support
type ProgressiveGenerator struct {
	analyzer  domain.SchemaAnalyzer
	executor  domain.TaskExecutor
	assembler domain.ProgressiveAssembler
	strategy  domain.ExecutionStrategy
	plugins   *PluginRegistry
}

func NewProgressiveGenerator(
	analyzer domain.SchemaAnalyzer,
	executor domain.TaskExecutor,
	assembler domain.ProgressiveAssembler,
	strategy domain.ExecutionStrategy,
) *ProgressiveGenerator {
	return &ProgressiveGenerator{
		analyzer:  analyzer,
		executor:  executor,
		assembler: assembler,
		strategy:  strategy,
		plugins:   NewPluginRegistry(),
	}
}

// Generate - Synchronous generation (collects all progressive chunks)
func (g *ProgressiveGenerator) Generate(request *domain.GenerationRequest) (*domain.GenerationResult, error) {
	stream, err := g.GenerateStreamProgressive(request)
	if err != nil {
		return nil, err
	}

	// Collect final result
	var finalChunk *domain.AccumulatedStreamChunk
	for chunk := range stream {
		if chunk.IsFinal {
			finalChunk = chunk
		}
	}

	if finalChunk == nil {
		return domain.NewGenerationResultWithError(err), nil
	}

	return domain.NewGenerationResult(finalChunk.CurrentMap, domain.NewResultMetadata()), nil
}

// GenerateStream - Not directly supported, falls back to progressive
func (g *ProgressiveGenerator) GenerateStream(request *domain.GenerationRequest) (<-chan *domain.StreamChunk, error) {
	progressiveStream, err := g.GenerateStreamProgressive(request)
	if err != nil {
		return nil, err
	}

	// Convert progressive chunks to simple chunks
	out := make(chan *domain.StreamChunk, 100)
	go func() {
		defer close(out)
		for chunk := range progressiveStream {
			if chunk.NewToken != nil {
				out <- domain.NewStreamChunk(chunk.NewToken.Key, chunk.NewToken.Partial)
			}
		}
	}()

	return out, nil
}

// GenerateStreamProgressive - Progressive token-level streaming
func (g *ProgressiveGenerator) GenerateStreamProgressive(request *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	out := make(chan *domain.AccumulatedStreamChunk, 100)

	go func() {
		defer close(out)

		// Pre-processing
		processedRequest, err := g.plugins.ApplyPreProcessors(request)
		if err != nil {
			return
		}

		// Analyze schema
		analysis, err := g.analyzer.Analyze(processedRequest.Schema())
		if err != nil {
			return
		}

		// Create execution plan
		tasks, err := g.analyzer.DetermineProcessingOrder(analysis.Fields)
		if err != nil {
			return
		}

		plan, err := g.strategy.Schedule(tasks)
		if err != nil {
			return
		}

		// Execute tasks - need to get token stream
		// This requires special handling for progressive execution
		context := domain.NewExecutionContext(processedRequest)

		// Create token stream channel
		tokenStream := make(chan *domain.TokenStreamChunk, 100)

		go func() {
			defer close(tokenStream)

			// Execute tasks and collect token streams
			results, _ := g.strategy.Execute(plan, g.executor, context)

			// For progressive, we need to merge token streams
			// This is simplified - actual implementation would be more complex
			for _, result := range results {
				// Convert result to tokens (simplified)
				if result.IsSuccess() {
					tokenStream <- domain.NewTokenStreamChunk(result.Key(), result.Value().(string))
				}
			}
		}()

		// Assemble progressive results
		chunks, err := g.assembler.AssembleProgressive(tokenStream)
		if err != nil {
			return
		}

		// Forward chunks
		for chunk := range chunks {
			out <- chunk
		}
	}()

	return out, nil
}

// RegisterPlugin registers a plugin
func (g *ProgressiveGenerator) RegisterPlugin(plugin domain.Plugin) {
	g.plugins.Register(plugin)
}
