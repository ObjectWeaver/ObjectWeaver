package application

import (
	"testing"

	"firechimp/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

// Mock observability plugin for testing
type mockObservability struct{}

func (m *mockObservability) Name() string                                                    { return "mock-obs" }
func (m *mockObservability) Version() string                                                 { return "1.0" }
func (m *mockObservability) Initialize(config map[string]interface{}) error                  { return nil }
func (m *mockObservability) RecordMetric(name string, value float64, tags map[string]string) {}
func (m *mockObservability) StartSpan(name string) domain.Span                               { return nil }

func TestNewPluginRegistry(t *testing.T) {
	registry := NewPluginRegistry()

	if registry.preProcessors == nil {
		t.Error("preProcessors should be initialized")
	}
	if registry.postProcessors == nil {
		t.Error("postProcessors should be initialized")
	}
	if registry.validators == nil {
		t.Error("validators should be initialized")
	}
	if len(registry.preProcessors) != 0 {
		t.Error("preProcessors should start empty")
	}
	if len(registry.postProcessors) != 0 {
		t.Error("postProcessors should start empty")
	}
	if len(registry.validators) != 0 {
		t.Error("validators should start empty")
	}
	if registry.cache != nil {
		t.Error("cache should be nil initially")
	}
	if registry.observability != nil {
		t.Error("observability should be nil initially")
	}
}

func TestRegister(t *testing.T) {
	registry := NewPluginRegistry()

	preProc := &mockPreProcessor{}
	postProc := &mockPostProcessor{}
	validator := &mockValidationPlugin{}
	cache := &mockCachePlugin{}
	obs := &mockObservability{}

	registry.Register(preProc)
	registry.Register(postProc)
	registry.Register(validator)
	registry.Register(cache)
	registry.Register(obs)

	if len(registry.preProcessors) != 1 || registry.preProcessors[0] != preProc {
		t.Error("preProcessor not registered correctly")
	}
	if len(registry.postProcessors) != 1 || registry.postProcessors[0] != postProc {
		t.Error("postProcessor not registered correctly")
	}
	if len(registry.validators) != 1 || registry.validators[0] != validator {
		t.Error("validator not registered correctly")
	}
	if registry.cache != cache {
		t.Error("cache not registered correctly")
	}
	if registry.observability != obs {
		t.Error("observability not registered correctly")
	}
}

func TestApplyPreProcessors(t *testing.T) {
	registry := NewPluginRegistry()

	preProc1 := &mockPreProcessor{}
	preProc2 := &mockPreProcessor{}
	registry.Register(preProc1)
	registry.Register(preProc2)

	req := domain.NewGenerationRequest("test", &jsonSchema.Definition{})

	result, err := registry.ApplyPreProcessors(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != req {
		t.Error("request should be returned unchanged")
	}
}

func TestApplyPreProcessors_Error(t *testing.T) {
	registry := NewPluginRegistry()

	preProc1 := &mockPreProcessor{}
	preProc2 := &mockPreProcessor{shouldError: true}
	registry.Register(preProc1)
	registry.Register(preProc2)

	req := domain.NewGenerationRequest("test", &jsonSchema.Definition{})

	_, err := registry.ApplyPreProcessors(req)
	if err == nil {
		t.Error("expected error from preProcessor")
	}
}

func TestApplyPostProcessors(t *testing.T) {
	registry := NewPluginRegistry()

	postProc1 := &mockPostProcessor{}
	postProc2 := &mockPostProcessor{}
	registry.Register(postProc1)
	registry.Register(postProc2)

	res := domain.NewGenerationResult(map[string]interface{}{"key": "value"}, nil)

	result, err := registry.ApplyPostProcessors(res)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != res {
		t.Error("result should be returned unchanged")
	}
}

func TestApplyPostProcessors_Error(t *testing.T) {
	registry := NewPluginRegistry()

	postProc1 := &mockPostProcessor{}
	postProc2 := &mockPostProcessor{shouldError: true}
	registry.Register(postProc1)
	registry.Register(postProc2)

	res := domain.NewGenerationResult(map[string]interface{}{"key": "value"}, nil)

	_, err := registry.ApplyPostProcessors(res)
	if err == nil {
		t.Error("expected error from postProcessor")
	}
}

func TestApplyValidation(t *testing.T) {
	registry := NewPluginRegistry()

	validator1 := &mockValidationPlugin{}
	validator2 := &mockValidationPlugin{}
	registry.Register(validator1)
	registry.Register(validator2)

	res := domain.NewGenerationResult(map[string]interface{}{"key": "value"}, nil)
	schema := &jsonSchema.Definition{}

	err := registry.ApplyValidation(res, schema)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyValidation_WithErrors(t *testing.T) {
	registry := NewPluginRegistry()

	validator1 := &mockValidationPlugin{}
	validator2 := &mockValidationPlugin{shouldError: true}
	registry.Register(validator1)
	registry.Register(validator2)

	res := domain.NewGenerationResult(map[string]interface{}{"key": "value"}, nil)
	schema := &jsonSchema.Definition{}

	err := registry.ApplyValidation(res, schema)
	if err == nil {
		t.Error("expected error from validator")
	}
}

func TestGetFromCache_NoCache(t *testing.T) {
	registry := NewPluginRegistry()

	result, found := registry.GetFromCache("key")
	if result != nil || found {
		t.Error("should return nil and false when no cache plugin")
	}
}

func TestGetFromCache_WithCache(t *testing.T) {
	registry := NewPluginRegistry()
	cache := &mockCachePlugin{}
	registry.Register(cache)

	expected := domain.NewGenerationResult(map[string]interface{}{"cached": true}, nil)
	cache.Set("key", expected)

	result, found := registry.GetFromCache("key")
	if !found {
		t.Error("should find cached result")
	}
	if result != expected {
		t.Error("should return cached result")
	}
}

func TestCacheResult_NoCache(t *testing.T) {
	registry := NewPluginRegistry()

	result := domain.NewGenerationResult(map[string]interface{}{"key": "value"}, nil)
	// Should not panic
	registry.CacheResult("key", result)
}

func TestCacheResult_WithCache(t *testing.T) {
	registry := NewPluginRegistry()
	cache := &mockCachePlugin{}
	registry.Register(cache)

	result := domain.NewGenerationResult(map[string]interface{}{"key": "value"}, nil)
	registry.CacheResult("key", result)

	cached, found := cache.Get("key")
	if !found {
		t.Error("result should be cached")
	}
	if cached != result {
		t.Error("cached result should match")
	}
}
