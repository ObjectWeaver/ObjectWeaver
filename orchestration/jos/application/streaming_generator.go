package application

import (
	"fmt"
	"objectweaver/orchestration/jos/domain"
)

// StreamingGenerator - Generator with field-level streaming support
type StreamingGenerator struct {
	analyzer  domain.SchemaAnalyzer
	executor  domain.TaskExecutor
	assembler domain.StreamingAssembler
	strategy  domain.ExecutionStrategy
	plugins   *PluginRegistry
}

func NewStreamingGenerator(
	analyzer domain.SchemaAnalyzer,
	executor domain.TaskExecutor,
	assembler domain.StreamingAssembler,
	strategy domain.ExecutionStrategy,
) *StreamingGenerator {
	return &StreamingGenerator{
		analyzer:  analyzer,
		executor:  executor,
		assembler: assembler,
		strategy:  strategy,
		plugins:   NewPluginRegistry(),
	}
}

// Generate - Synchronous generation (falls back to collecting stream)
func (g *StreamingGenerator) Generate(request *domain.GenerationRequest) (*domain.GenerationResult, error) {
	stream, err := g.GenerateStream(request)
	if err != nil {
		return nil, err
	}

	// Collect all chunks
	finalData := make(map[string]interface{})
	metadata := domain.NewResultMetadata()

	for chunk := range stream {
		if !chunk.IsFinal {
			finalData[chunk.Key] = chunk.Value
		}
	}

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

		// Execute tasks
		context := domain.NewExecutionContext(processedRequest)
		// Add user's prompt to context for proper field generation
		context.PromptContext().AddPrompt(processedRequest.Prompt())
		results, err := g.strategy.Execute(plan, g.executor, context)
		if err != nil {
			return
		}

		// Stream results through assembler
		resultChan := make(chan *domain.TaskResult, len(results))
		go func() {
			for _, result := range results {
				resultChan <- result
			}
			close(resultChan)
		}()

		chunks, err := g.assembler.AssembleStreaming(resultChan)
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

// GenerateStreamProgressive - Not supported
func (g *StreamingGenerator) GenerateStreamProgressive(request *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	return nil, fmt.Errorf("progressive streaming not supported by StreamingGenerator")
}

// RegisterPlugin registers a plugin
func (g *StreamingGenerator) RegisterPlugin(plugin domain.Plugin) {
	g.plugins.Register(plugin)
}
