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
package factory

import (
	"fmt"
	"objectweaver/orchestration/jos/application"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure/llm"
	"objectweaver/orchestration/jos/infrastructure/prompt"
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

// Create builds a fully configured generator. Returns the execute system within the generator can accessed and intialised. The primary initialisation point.
func (f *GeneratorFactory) Create() (domain.Generator, error) {
	// Create LLM provider and prompt builder
	llmProvider := f.createLLMProvider()
	promptBuilder := f.createPromptBuilder()

	// Create generator based on mode
	var generator domain.Generator

	switch f.config.Mode {
	case ModeStreamingProgressive:
		// This mode is temporarily unsupported until refactoring is complete
		return nil, fmt.Errorf("progressive streaming mode not yet supported with new architecture")

	case ModeStreaming, ModeStreamingComplete:
		// Streaming now uses the same FieldProcessor as default mode!
		generator = application.NewStreamingGenerator(llmProvider, promptBuilder)

	default:
		// All non-streaming modes use the new recursive architecture
		// This completely bypasses TaskExecutor, Analyzer, Strategy, and Assembler
		generator = application.NewDefaultGenerator(llmProvider, promptBuilder)
	}

	// Register plugins if needed
	if pluggable, ok := generator.(PluginRegistry); ok {
		f.registerPlugins(pluggable)
	}

	return generator, nil
}

// The following methods are deprecated and only kept for reference
// They were used by the old TaskExecutor architecture
// TODO: Remove once ProgressiveGenerator is refactored

/*
func (f *GeneratorFactory) createAnalyzer() domain.SchemaAnalyzer {
	return analysis.NewDefaultSchemaAnalyzer()
}

// createExecutor creates the class with the neccessary injections so that the the individual peice of content is generated. The TaskExecutor returned from this will be used along with the stratgies for the generation process.
func (f *GeneratorFactory) createExecutor() domain.TaskExecutor {
	llmProvider := f.createLLMProvider()
	promptBuilder := f.createPromptBuilder()

	// Create type processors - object, array, primitive types etc.
	processors := f.createTypeProcessors(llmProvider, promptBuilder)

	return execution.NewCompositeTaskExecutor(llmProvider, promptBuilder, processors)
}

// createTypeProcessors intialisers of the different types of processors along with the streaming counter part if that is the approach that the request takes. Through the JSON structure and through the usage of grpc server.
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

// createAssembler the factory for how the data is being sent to the final server. Either waiting until all the generation is created. Or streaming the content back out.
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

// createStrategy factory for how the generation process will occur. ie sequentially or in parrell for faster performance.
func (f *GeneratorFactory) createStrategy() domain.ExecutionStrategy {
	switch f.config.Mode {
	case ModeSync:
		return strategies.NewSequentialStrategy()
	case ModeParallel, ModeStreaming, ModeStreamingComplete, ModeStreamingProgressive:
		return strategies.NewParallelStrategy(f.config.MaxConcurrency)
	case ModeDependencyAware:
		return strategies.NewDependencyAwareStrategy(f.config.MaxConcurrency)
	default:
		return strategies.NewSequentialStrategy()
	}
}
*/

// createLLMProvider currently just returns the openAi provider so that the requests are sent out in the openAI format. Which is the main standard for API requests.
func (f *GeneratorFactory) createLLMProvider() domain.LLMProvider {
	return llm.NewOpenAIProvider()
}

func (f *GeneratorFactory) createPromptBuilder() domain.PromptBuilder {
	return prompt.NewDefaultPromptBuilder()
}

// registerPlugins plugins for pre and post processing so that additional functionality can be clicked in depending on the configuration of the service.
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
