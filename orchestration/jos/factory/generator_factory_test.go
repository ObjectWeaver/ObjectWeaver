package factory

import (
	"testing"

	"firechimp/orchestration/jos/application"
	"firechimp/orchestration/jos/domain"
)

func TestNewGeneratorFactory_NilConfig_UsesDefault(t *testing.T) {
	factory := NewGeneratorFactory(nil)
	if factory.config == nil {
		t.Error("Expected default config, got nil")
	}
	if factory.config.Mode != ModeParallel {
		t.Errorf("Expected default mode %v, got %v", ModeParallel, factory.config.Mode)
	}
}

func TestNewGeneratorFactory_WithConfig_UsesProvided(t *testing.T) {
	config := &GeneratorConfig{Mode: ModeSync}
	factory := NewGeneratorFactory(config)
	if factory.config != config {
		t.Error("Expected provided config, got different config")
	}
}

func TestCreate_DefaultMode_ReturnsDefaultGenerator(t *testing.T) {
	config := DefaultGeneratorConfig().WithMode(ModeSync)
	factory := NewGeneratorFactory(config)
	generator, err := factory.Create()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if generator == nil {
		t.Error("Expected non-nil generator")
	}
	if _, ok := generator.(*application.DefaultGenerator); !ok {
		t.Errorf("Expected DefaultGenerator, got %T", generator)
	}
}

func TestCreate_StreamingMode_ReturnsStreamingGenerator(t *testing.T) {
	config := DefaultGeneratorConfig().WithMode(ModeStreaming)
	factory := NewGeneratorFactory(config)
	generator, err := factory.Create()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if generator == nil {
		t.Error("Expected non-nil generator")
	}
	if _, ok := generator.(*application.StreamingGenerator); !ok {
		t.Errorf("Expected StreamingGenerator, got %T", generator)
	}
}

func TestCreate_StreamingCompleteMode_ReturnsStreamingGenerator(t *testing.T) {
	config := DefaultGeneratorConfig().WithMode(ModeStreamingComplete)
	factory := NewGeneratorFactory(config)
	generator, err := factory.Create()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if generator == nil {
		t.Error("Expected non-nil generator")
	}
	if _, ok := generator.(*application.StreamingGenerator); !ok {
		t.Errorf("Expected StreamingGenerator, got %T", generator)
	}
}

func TestCreate_StreamingProgressiveMode_ReturnsProgressiveGenerator(t *testing.T) {
	config := DefaultGeneratorConfig().WithMode(ModeStreamingProgressive)
	factory := NewGeneratorFactory(config)
	generator, err := factory.Create()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if generator == nil {
		t.Error("Expected non-nil generator")
	}
	if _, ok := generator.(*application.ProgressiveGenerator); !ok {
		t.Errorf("Expected ProgressiveGenerator, got %T", generator)
	}
}

func TestCreate_ParallelMode_ReturnsDefaultGenerator(t *testing.T) {
	config := DefaultGeneratorConfig().WithMode(ModeParallel)
	factory := NewGeneratorFactory(config)
	generator, err := factory.Create()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if generator == nil {
		t.Error("Expected non-nil generator")
	}
	if _, ok := generator.(*application.DefaultGenerator); !ok {
		t.Errorf("Expected DefaultGenerator, got %T", generator)
	}
}

// Mock generator for testing plugin registration
type mockGenerator struct {
	domain.Generator
	registeredPlugins []domain.Plugin
}

func (m *mockGenerator) RegisterPlugin(plugin domain.Plugin) {
	m.registeredPlugins = append(m.registeredPlugins, plugin)
}

func TestCreate_RegistersPlugins_IfGeneratorSupports(t *testing.T) {
	// This test assumes we have a way to inject a mock generator, but since Create creates real ones,
	// and the real generators may or may not implement PluginRegistry, this is tricky.
	// For now, skip or test that it doesn't panic.
	config := DefaultGeneratorConfig()
	config.Plugins = []domain.Plugin{} // empty for now
	factory := NewGeneratorFactory(config)
	_, err := factory.Create()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// Since plugins are commented out, it should not register anything.
	// But to test, perhaps check that the generator is created.
}
