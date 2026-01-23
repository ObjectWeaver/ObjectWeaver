package application

import (
	"context"
	"fmt"
	"objectweaver/logger"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure/assembly"
	"objectweaver/orchestration/jos/infrastructure/epstimic"
	"objectweaver/orchestration/jos/infrastructure/execution"
	"sync"
	"time"
)

// DefaultGenerator - Main orchestrator for synchronous generation
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
	// Use context from request if available, otherwise background
	reqCtx := request.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	// Create context with timeout to prevent indefinite hangs
	// Default to 120s, or use env var if needed (not implemented here for simplicity)
	ctx, cancel := context.WithTimeout(reqCtx, 300*time.Second)
	defer cancel()

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
	execContext := domain.NewExecutionContext(processedRequest)
	workerPool := execution.NewWorkerPool(-1)
	execContext.SetWorkerPool(workerPool)
	execContext.PromptContext().AddPrompt(processedRequest.Prompt())

	// Process all fields
	startTime := time.Now()
	logger.Printf("[DefaultGenerator] Starting field processing")

	resultsCh := g.fieldProcessor.ProcessFieldsStart(ctx, processedRequest.Schema(), nil, execContext)

	// Collect flat results with path information
	var allResults []*domain.TaskResult

	for results := range resultsCh {
		select {
		case <-ctx.Done():
			logger.Printf("[DefaultGenerator] Request TIMEOUT after %v: %v", time.Since(startTime), ctx.Err())
			return nil, fmt.Errorf("request timeout after %v: %w", time.Since(startTime), ctx.Err())
		default:
		}

		// Collect all results - they have path information
		allResults = append(allResults, results...)
	}

	logger.Printf("[DefaultGenerator] Field processing completed in %v, collected %d results", time.Since(startTime), len(allResults))

	// Use path-based assembler to reconstruct nested structure from flat results
	assembler := assembly.NewPathBasedAssembler()
	result, err := assembler.Assemble(allResults)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble results: %w", err)
	}

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
	// Simple cache key generation
	return fmt.Sprintf("%s_%s", request.Prompt(), request.Schema().Instruction)
}
