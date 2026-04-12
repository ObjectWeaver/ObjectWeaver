package factory

import (
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
	"testing"
)

func TestGeneratorConfig_WithConcurrency(t *testing.T) {

	config := DefaultGeneratorConfig()

	concurrencyConfig := config.WithConcurrency(5)

	if concurrencyConfig.MaxConcurrency != 5 {
		t.Errorf("Expected concurrency to be 5, got %d", concurrencyConfig.MaxConcurrency)
	}
}

func TestGeneratorConfig_WithCache(t *testing.T) {

	config := DefaultGeneratorConfig()

	newConfig := config.WithCache(true)

	if !newConfig.EnableCache {
		t.Errorf("Expected cache to be enabled, got disabled")
	}

	// Ensure original config is unchanged
	if config.EnableCache {
		t.Errorf("Expected original config cache to be disabled, got enabled")
	}
}

type mockPlugin struct{}

func (m *mockPlugin) Name() string {
	return "mockPlugin"
}

func (m *mockPlugin) Version() string {
	return "1.0.0"
}

func (m *mockPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func TestGeneratorConfig_WithPlugin(t *testing.T) {

	config := DefaultGeneratorConfig()

	mockPlugin := &mockPlugin{}

	newConfig := config.WithPlugin(mockPlugin)

	if len(newConfig.Plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(newConfig.Plugins))
	}

	if newConfig.Plugins[0].Name() != "mockPlugin" {
		t.Errorf("Expected plugin name to be 'mockPlugin', got '%s'", newConfig.Plugins[0].Name())
	}

	// Ensure original config is unchanged
	if len(config.Plugins) != 0 {
		t.Errorf("Expected original config to have 0 plugins, got %d", len(config.Plugins))
	}
}

func TestGeneratorConfig_WithGranularity(t *testing.T) {
	config := DefaultGeneratorConfig()

	newConfig := config.WithGranularity(domain.GranularityToken)

	if newConfig.Granularity != domain.GranularityToken {
		t.Errorf("Expected granularity to be GranularityToken, got %v", newConfig.Granularity)
	}

	// Ensure original config is unchanged
	if config.Granularity != domain.GranularityField {
		t.Errorf("Expected original config granularity to be GranularityField, got %v", config.Granularity)
	}
}
