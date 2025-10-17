package application

import (
	"fmt"
	"objectweaver/orchestration/jos/domain"
	"sync"
)

// DefaultGenerator - Main orchestrator for synchronous generation
type DefaultGenerator struct {
	analyzer  domain.SchemaAnalyzer
	executor  domain.TaskExecutor
	assembler domain.ResultAssembler
	strategy  domain.ExecutionStrategy
	plugins   *PluginRegistry
	mu        sync.RWMutex
}

func NewDefaultGenerator(
	analyzer domain.SchemaAnalyzer,
	executor domain.TaskExecutor,
	assembler domain.ResultAssembler,
	strategy domain.ExecutionStrategy,
) *DefaultGenerator {
	return &DefaultGenerator{
		analyzer:  analyzer,
		executor:  executor,
		assembler: assembler,
		strategy:  strategy,
		plugins:   NewPluginRegistry(),
	}
}

// Generate - Main generation workflow
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

	// Phase 3: Analyze schema
	analysis, err := g.analyzer.Analyze(processedRequest.Schema())
	if err != nil {
		return nil, fmt.Errorf("schema analysis failed: %w", err)
	}

	// Phase 4: Create execution plan
	tasks, err := g.analyzer.DetermineProcessingOrder(analysis.Fields)
	if err != nil {
		return nil, fmt.Errorf("task planning failed: %w", err)
	}

	plan, err := g.strategy.Schedule(tasks)
	if err != nil {
		return nil, fmt.Errorf("scheduling failed: %w", err)
	}

	// Phase 5: Execute tasks
	context := domain.NewExecutionContext(processedRequest)
	// Add user's prompt to context for proper field generation
	context.PromptContext().AddPrompt(processedRequest.Prompt())
	results, err := g.strategy.Execute(plan, g.executor, context)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Phase 6: Assemble results
	finalResult, err := g.assembler.Assemble(results)
	if err != nil {
		return nil, fmt.Errorf("assembly failed: %w", err)
	}

	// Phase 7: Post-processing plugins
	processedResult, err := g.plugins.ApplyPostProcessors(finalResult)
	if err != nil {
		return nil, fmt.Errorf("post-processing failed: %w", err)
	}

	// Phase 8: Validation
	if err := g.plugins.ApplyValidation(processedResult, processedRequest.Schema()); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Phase 9: Cache result
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
