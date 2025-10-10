package factory

import (
	"objectGeneration/orchestration/jos/application"
	"objectGeneration/orchestration/jos/domain"
	"objectGeneration/orchestration/jos/infrastructure/analysis"
	"objectGeneration/orchestration/jos/infrastructure/assembly"
	"objectGeneration/orchestration/jos/infrastructure/execution"
	"objectGeneration/orchestration/jos/infrastructure/llm"
	"objectGeneration/orchestration/jos/infrastructure/prompt"
)

// GeneratorFactory creates configured generators
type GeneratorFactory struct {
	config *GeneratorConfig
}

func NewGeneratorFactory(config *GeneratorConfig) *GeneratorFactory {
	if config == nil {
		config = DefaultGeneratorConfig()
	}
	return &GeneratorFactory{config: config}
}

// Create builds a fully configured generator
func (f *GeneratorFactory) Create() (domain.Generator, error) {
	// Create core components
	analyzer := f.createAnalyzer()
	executor := f.createExecutor()
	assembler := f.createAssembler()
	strategy := f.createStrategy()

	// Create generator based on mode
	var generator domain.Generator

	switch f.config.Mode {
	case ModeStreamingProgressive:
		generator = application.NewProgressiveGenerator(analyzer, executor, assembler.(domain.ProgressiveAssembler), strategy)
	case ModeStreaming, ModeStreamingComplete:
		generator = application.NewStreamingGenerator(analyzer, executor, assembler.(domain.StreamingAssembler), strategy)
	default:
		generator = application.NewDefaultGenerator(analyzer, executor, assembler, strategy)
	}

	// Register plugins if needed
	if pluggable, ok := generator.(PluginRegistry); ok {
		f.registerPlugins(pluggable)
	}

	return generator, nil
}

func (f *GeneratorFactory) createAnalyzer() domain.SchemaAnalyzer {
	return analysis.NewDefaultSchemaAnalyzer()
}

func (f *GeneratorFactory) createExecutor() domain.TaskExecutor {
	llmProvider := f.createLLMProvider()
	promptBuilder := f.createPromptBuilder()

	// Create type processors
	processors := f.createTypeProcessors(llmProvider, promptBuilder)

	return execution.NewCompositeTaskExecutor(llmProvider, promptBuilder, processors)
}

func (f *GeneratorFactory) createTypeProcessors(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) []domain.TypeProcessor {
	processors := make([]domain.TypeProcessor, 0)

	// Create appropriate processors based on mode
	if f.config.Mode == ModeStreamingProgressive {
		if tokenProvider, ok := llmProvider.(domain.TokenStreamingProvider); ok {
			processors = append(processors, execution.NewStreamingPrimitiveProcessor(tokenProvider, promptBuilder, f.config.Granularity))
			processors = append(processors, execution.NewStreamingObjectProcessor(tokenProvider, promptBuilder, f.config.Granularity))
			processors = append(processors, execution.NewStreamingArrayProcessor(tokenProvider, promptBuilder, f.config.Granularity))
		}
	} else {
		// Standard processors - order doesn't matter much since CompositeTaskExecutor
		// checks for byte operation configs before type-based selection
		processors = append(processors, execution.NewPrimitiveProcessor(llmProvider, promptBuilder))
		processors = append(processors, execution.NewObjectProcessor(llmProvider, promptBuilder))
		processors = append(processors, execution.NewArrayProcessor(llmProvider, promptBuilder))
		processors = append(processors, execution.NewBooleanProcessor(llmProvider, promptBuilder))
		processors = append(processors, execution.NewNumberProcessor(llmProvider, promptBuilder))
		processors = append(processors, execution.NewByteProcessor(llmProvider, promptBuilder))
		processors = append(processors, execution.NewMapProcessor(llmProvider, promptBuilder))
	}

	return processors
}

func (f *GeneratorFactory) createAssembler() domain.ResultAssembler {
	switch f.config.Mode {
	case ModeStreamingProgressive:
		return assembly.NewProgressiveObjectAssembler(100) // 100ms emit interval
	case ModeStreaming:
		return assembly.NewStreamingAssembler()
	case ModeStreamingComplete:
		return assembly.NewCompleteStreamingAssembler()
	default:
		return assembly.NewDefaultAssembler()
	}
}

func (f *GeneratorFactory) createStrategy() domain.ExecutionStrategy {
	switch f.config.Mode {
	case ModeSync:
		return execution.NewSequentialStrategy()
	case ModeParallel, ModeStreaming, ModeStreamingComplete, ModeStreamingProgressive:
		return execution.NewParallelStrategy(f.config.MaxConcurrency)
	case ModeDependencyAware:
		return execution.NewDependencyAwareStrategy(f.config.MaxConcurrency)
	default:
		return execution.NewSequentialStrategy()
	}
}

func (f *GeneratorFactory) createLLMProvider() domain.LLMProvider {
	switch f.config.LLMProvider {
	case "openai":
		// Return adapter that wraps existing job submitter
		return llm.NewOpenAIProvider()
	default:
		return llm.NewOpenAIProvider()
	}
}

func (f *GeneratorFactory) createPromptBuilder() domain.PromptBuilder {
	return prompt.NewDefaultPromptBuilder()
}

func (f *GeneratorFactory) registerPlugins(registry PluginRegistry) {
	if f.config.EnableCache {
		// Register cache plugin when available
		// registry.RegisterPlugin(plugins.NewCachePlugin())
	}

	if f.config.EnableValidation {
		// Register validation plugin when available
		// registry.RegisterPlugin(plugins.NewValidationPlugin())
	}

	if f.config.EnableObservability {
		// Register observability plugins when available
		// registry.RegisterPlugin(plugins.NewPrometheusPlugin())
	}

	for _, plugin := range f.config.Plugins {
		registry.RegisterPlugin(plugin)
	}
}

// PluginRegistry interface for generators that support plugins
type PluginRegistry interface {
	RegisterPlugin(plugin domain.Plugin)
}
