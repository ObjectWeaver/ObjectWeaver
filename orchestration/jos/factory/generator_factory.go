package factory

import (
	"fmt"
	"objectweaver/orchestration/jos/application"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure/llm"
	"objectweaver/orchestration/jos/infrastructure/prompt"
	"sync"
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
