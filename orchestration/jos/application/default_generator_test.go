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
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package application

import (
	"errors"
	"objectweaver/orchestration/jos/domain"
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// Mock LLMProvider for testing
type mockLLMProvider struct {
	generateFunc          func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error)
	supportsStreamingFunc func() bool
	modelTypeFunc         func() string
}

func (m *mockLLMProvider) Generate(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
	if m.generateFunc != nil {
		return m.generateFunc(prompt, config)
	}
	return "mock result", &domain.ProviderMetadata{
		TokensUsed:   10,
		Cost:         0.001,
		Model:        "mock-model",
		FinishReason: "stop",
	}, nil
}

func (m *mockLLMProvider) SupportsStreaming() bool {
	if m.supportsStreamingFunc != nil {
		return m.supportsStreamingFunc()
	}
	return false
}

func (m *mockLLMProvider) ModelType() string {
	if m.modelTypeFunc != nil {
		return m.modelTypeFunc()
	}
	return "mock"
}

// Mock PromptBuilder for testing
type mockPromptBuilder struct {
	buildFunc            func(task *domain.FieldTask, context *domain.PromptContext) (string, error)
	buildWithHistoryFunc func(task *domain.FieldTask, context *domain.PromptContext, history *domain.GenerationHistory) (string, error)
}

func (m *mockPromptBuilder) Build(task *domain.FieldTask, context *domain.PromptContext) (string, error) {
	if m.buildFunc != nil {
		return m.buildFunc(task, context)
	}
	return "mock prompt", nil
}

func (m *mockPromptBuilder) BuildWithHistory(task *domain.FieldTask, context *domain.PromptContext, history *domain.GenerationHistory) (string, error) {
	if m.buildWithHistoryFunc != nil {
		return m.buildWithHistoryFunc(task, context, history)
	}
	return "mock prompt with history", nil
}

// Mock PreProcessorPlugin for testing
type mockPreProcessorPlugin struct {
	name           string
	version        string
	preProcessFunc func(request *domain.GenerationRequest) (*domain.GenerationRequest, error)
}

func (m *mockPreProcessorPlugin) Name() string    { return m.name }
func (m *mockPreProcessorPlugin) Version() string { return m.version }
func (m *mockPreProcessorPlugin) Initialize(config map[string]interface{}) error {
	return nil
}
func (m *mockPreProcessorPlugin) PreProcess(request *domain.GenerationRequest) (*domain.GenerationRequest, error) {
	if m.preProcessFunc != nil {
		return m.preProcessFunc(request)
	}
	return request, nil
}

// Mock PostProcessorPlugin for testing
type mockPostProcessorPlugin struct {
	name            string
	version         string
	postProcessFunc func(result *domain.GenerationResult) (*domain.GenerationResult, error)
}

func (m *mockPostProcessorPlugin) Name() string    { return m.name }
func (m *mockPostProcessorPlugin) Version() string { return m.version }
func (m *mockPostProcessorPlugin) Initialize(config map[string]interface{}) error {
	return nil
}
func (m *mockPostProcessorPlugin) PostProcess(result *domain.GenerationResult) (*domain.GenerationResult, error) {
	if m.postProcessFunc != nil {
		return m.postProcessFunc(result)
	}
	return result, nil
}

// Mock ValidationPlugin for testing
type mockValidationPlugin struct {
	name         string
	version      string
	validateFunc func(result *domain.GenerationResult, schema *jsonSchema.Definition) ([]domain.ValidationError, error)
}

func (m *mockValidationPlugin) Name() string    { return m.name }
func (m *mockValidationPlugin) Version() string { return m.version }
func (m *mockValidationPlugin) Initialize(config map[string]interface{}) error {
	return nil
}
func (m *mockValidationPlugin) Validate(result *domain.GenerationResult, schema *jsonSchema.Definition) ([]domain.ValidationError, error) {
	if m.validateFunc != nil {
		return m.validateFunc(result, schema)
	}
	return []domain.ValidationError{}, nil
}

// Mock CachePlugin for testing
type mockCachePlugin struct {
	name    string
	version string
	cache   map[string]*domain.GenerationResult
}

func newMockCachePlugin() *mockCachePlugin {
	return &mockCachePlugin{
		name:    "mock-cache",
		version: "1.0.0",
		cache:   make(map[string]*domain.GenerationResult),
	}
}

func (m *mockCachePlugin) Name() string    { return m.name }
func (m *mockCachePlugin) Version() string { return m.version }
func (m *mockCachePlugin) Initialize(config map[string]interface{}) error {
	return nil
}
func (m *mockCachePlugin) Get(key string) (*domain.GenerationResult, bool) {
	result, found := m.cache[key]
	return result, found
}
func (m *mockCachePlugin) Set(key string, result *domain.GenerationResult) error {
	m.cache[key] = result
	return nil
}

// TestNewDefaultGenerator tests the constructor
func TestNewDefaultGenerator(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	if generator == nil {
		t.Fatal("expected generator to be non-nil")
	}

	if generator.llmProvider == nil {
		t.Error("expected llmProvider to be set")
	}

	if generator.promptBuilder == nil {
		t.Error("expected promptBuilder to be set")
	}

	if generator.fieldProcessor == nil {
		t.Error("expected fieldProcessor to be set")
	}

	if generator.plugins == nil {
		t.Error("expected plugins registry to be set")
	}
}

// TestGenerate_BasicFlow tests the basic generation workflow
func TestGenerate_BasicFlow(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Generate a test object",
		Properties: map[string]jsonSchema.Definition{
			"name": {
				Type: jsonSchema.String,
			},
		},
	}

	request := domain.NewGenerationRequest("Generate a test", schema)

	result, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if !result.IsSuccess() {
		t.Error("expected result to be successful")
	}
}

// TestGenerate_PreProcessorPlugin tests pre-processor plugin integration
func TestGenerate_PreProcessorPlugin(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	preProcessorCalled := false
	preProcessor := &mockPreProcessorPlugin{
		name:    "test-preprocessor",
		version: "1.0.0",
		preProcessFunc: func(request *domain.GenerationRequest) (*domain.GenerationRequest, error) {
			preProcessorCalled = true
			return request.WithMetadata("preprocessed", true), nil
		},
	}

	generator.RegisterPlugin(preProcessor)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	_, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !preProcessorCalled {
		t.Error("expected pre-processor to be called")
	}
}

// TestGenerate_PreProcessorError tests error handling in pre-processor
func TestGenerate_PreProcessorError(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	preProcessorError := errors.New("pre-processing failed")
	preProcessor := &mockPreProcessorPlugin{
		name:    "failing-preprocessor",
		version: "1.0.0",
		preProcessFunc: func(request *domain.GenerationRequest) (*domain.GenerationRequest, error) {
			return nil, preProcessorError
		},
	}

	generator.RegisterPlugin(preProcessor)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties:  map[string]jsonSchema.Definition{},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	_, err := generator.Generate(request)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, preProcessorError) {
		t.Errorf("expected error to contain pre-processor error, got: %v", err)
	}
}

// TestGenerate_PostProcessorPlugin tests post-processor plugin integration
func TestGenerate_PostProcessorPlugin(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	postProcessorCalled := false
	postProcessor := &mockPostProcessorPlugin{
		name:    "test-postprocessor",
		version: "1.0.0",
		postProcessFunc: func(result *domain.GenerationResult) (*domain.GenerationResult, error) {
			postProcessorCalled = true
			return result, nil
		},
	}

	generator.RegisterPlugin(postProcessor)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	_, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !postProcessorCalled {
		t.Error("expected post-processor to be called")
	}
}

// TestGenerate_PostProcessorError tests error handling in post-processor
func TestGenerate_PostProcessorError(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	postProcessorError := errors.New("post-processing failed")
	postProcessor := &mockPostProcessorPlugin{
		name:    "failing-postprocessor",
		version: "1.0.0",
		postProcessFunc: func(result *domain.GenerationResult) (*domain.GenerationResult, error) {
			return nil, postProcessorError
		},
	}

	generator.RegisterPlugin(postProcessor)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	_, err := generator.Generate(request)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, postProcessorError) {
		t.Errorf("expected error to contain post-processor error, got: %v", err)
	}
}

// TestGenerate_ValidationPlugin tests validation plugin integration
func TestGenerate_ValidationPlugin(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	validatorCalled := false
	validator := &mockValidationPlugin{
		name:    "test-validator",
		version: "1.0.0",
		validateFunc: func(result *domain.GenerationResult, schema *jsonSchema.Definition) ([]domain.ValidationError, error) {
			validatorCalled = true
			return []domain.ValidationError{}, nil
		},
	}

	generator.RegisterPlugin(validator)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	_, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !validatorCalled {
		t.Error("expected validator to be called")
	}
}

// TestGenerate_ValidationError tests validation error handling
func TestGenerate_ValidationError(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	validationError := errors.New("validation failed")
	validator := &mockValidationPlugin{
		name:    "failing-validator",
		version: "1.0.0",
		validateFunc: func(result *domain.GenerationResult, schema *jsonSchema.Definition) ([]domain.ValidationError, error) {
			return []domain.ValidationError{
				{Field: "name", Message: "invalid", Code: "INVALID"},
			}, validationError
		},
	}

	generator.RegisterPlugin(validator)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	_, err := generator.Generate(request)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestGenerate_CacheHit tests cache hit scenario
func TestGenerate_CacheHit(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			t.Error("LLM provider should not be called on cache hit")
			return nil, nil, errors.New("should not be called")
		},
	}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	cache := newMockCachePlugin()
	generator.RegisterPlugin(cache)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	// Pre-populate cache
	cachedResult := domain.NewGenerationResult(
		map[string]interface{}{"field": "cached value"},
		domain.NewResultMetadata(),
	)
	cacheKey := generateCacheKey(request)
	cache.Set(cacheKey, cachedResult)

	result, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if result != cachedResult {
		t.Error("expected cached result to be returned")
	}
}

// TestGenerate_CacheMiss tests cache miss scenario
func TestGenerate_CacheMiss(t *testing.T) {
	generateCalled := false
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			generateCalled = true
			return "generated value", &domain.ProviderMetadata{}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	cache := newMockCachePlugin()
	generator.RegisterPlugin(cache)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	result, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if !generateCalled {
		t.Error("expected LLM provider to be called on cache miss")
	}

	// Verify result was cached
	cacheKey := generateCacheKey(request)
	cachedResult, found := cache.Get(cacheKey)
	if !found {
		t.Error("expected result to be cached")
	}
	if cachedResult == nil {
		t.Error("expected cached result to be non-nil")
	}
}

// TestGenerateStream_NotSupported tests that streaming is not supported
func TestGenerateStream_NotSupported(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	result, err := generator.GenerateStream(request)

	if err == nil {
		t.Fatal("expected error for unsupported streaming")
	}

	if result != nil {
		t.Error("expected nil result for unsupported streaming")
	}
}

// TestGenerateStreamProgressive_NotSupported tests that progressive streaming is not supported
func TestGenerateStreamProgressive_NotSupported(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	result, err := generator.GenerateStreamProgressive(request)

	if err == nil {
		t.Fatal("expected error for unsupported progressive streaming")
	}

	if result != nil {
		t.Error("expected nil result for unsupported progressive streaming")
	}
}

// TestRegisterPlugin tests plugin registration
func TestRegisterPlugin(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	preProcessor := &mockPreProcessorPlugin{name: "test", version: "1.0.0"}
	postProcessor := &mockPostProcessorPlugin{name: "test", version: "1.0.0"}
	validator := &mockValidationPlugin{name: "test", version: "1.0.0"}
	cache := newMockCachePlugin()

	// Should not panic
	generator.RegisterPlugin(preProcessor)
	generator.RegisterPlugin(postProcessor)
	generator.RegisterPlugin(validator)
	generator.RegisterPlugin(cache)
}

// TestGenerateCacheKey tests cache key generation
func TestGenerateCacheKey(t *testing.T) {
	schema1 := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test instruction",
	}
	request1 := domain.NewGenerationRequest("Test prompt", schema1)

	schema2 := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test instruction",
	}
	request2 := domain.NewGenerationRequest("Test prompt", schema2)

	schema3 := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Different instruction",
	}
	request3 := domain.NewGenerationRequest("Test prompt", schema3)

	key1 := generateCacheKey(request1)
	key2 := generateCacheKey(request2)
	key3 := generateCacheKey(request3)

	if key1 != key2 {
		t.Error("expected identical requests to generate same cache key")
	}

	if key1 == key3 {
		t.Error("expected different requests to generate different cache keys")
	}
}

// TestGenerate_MultiplePlugins tests multiple plugins working together
func TestGenerate_MultiplePlugins(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	preProcessorCalled := false
	postProcessorCalled := false
	validatorCalled := false

	preProcessor := &mockPreProcessorPlugin{
		name:    "test-preprocessor",
		version: "1.0.0",
		preProcessFunc: func(request *domain.GenerationRequest) (*domain.GenerationRequest, error) {
			preProcessorCalled = true
			return request, nil
		},
	}

	postProcessor := &mockPostProcessorPlugin{
		name:    "test-postprocessor",
		version: "1.0.0",
		postProcessFunc: func(result *domain.GenerationResult) (*domain.GenerationResult, error) {
			postProcessorCalled = true
			return result, nil
		},
	}

	validator := &mockValidationPlugin{
		name:    "test-validator",
		version: "1.0.0",
		validateFunc: func(result *domain.GenerationResult, schema *jsonSchema.Definition) ([]domain.ValidationError, error) {
			validatorCalled = true
			return []domain.ValidationError{}, nil
		},
	}

	generator.RegisterPlugin(preProcessor)
	generator.RegisterPlugin(postProcessor)
	generator.RegisterPlugin(validator)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	_, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !preProcessorCalled {
		t.Error("expected pre-processor to be called")
	}

	if !postProcessorCalled {
		t.Error("expected post-processor to be called")
	}

	if !validatorCalled {
		t.Error("expected validator to be called")
	}
}

// TestGenerate_EmptySchema tests generation with empty schema properties
func TestGenerate_EmptySchema(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewDefaultGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties:  map[string]jsonSchema.Definition{},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	result, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if result.Data() == nil {
		t.Error("expected data to be non-nil")
	}
}
