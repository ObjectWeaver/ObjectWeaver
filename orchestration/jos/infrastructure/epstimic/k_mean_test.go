package epstimic

import (
	"errors"
	"math"
	"objectweaver/orchestration/jos/domain"
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// Mock Generator for KMeanEngine testing
type mockGenerator struct {
	generateFunc func(req *domain.GenerationRequest) (*domain.GenerationResult, error)
}

func (m *mockGenerator) Generate(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(req)
	}
	// Default: return a mock embedding
	return domain.NewGenerationResult(
		map[string]interface{}{
			"embedding": []float32{0.1, 0.2, 0.3},
		},
		domain.NewResultMetadata(),
	), nil
}

func (m *mockGenerator) GenerateStream(request *domain.GenerationRequest) (<-chan *domain.StreamChunk, error) {
	ch := make(chan *domain.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockGenerator) GenerateStreamProgressive(request *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	ch := make(chan *domain.AccumulatedStreamChunk)
	close(ch)
	return ch, nil
}

func TestNewKMeanEngine(t *testing.T) {
	generator := &mockGenerator{}
	model := "text-embedding-3-small"

	engine := NewKMeanEngine(model, generator)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	kEngine, ok := engine.(*KMeanEngine)
	if !ok {
		t.Fatal("Expected *KMeanEngine type")
	}

	if kEngine.model != model {
		t.Errorf("Expected model %s, got %s", model, kEngine.model)
	}

	if kEngine.generator == nil {
		t.Error("Expected non-nil generator")
	}
}

func TestKMeanEngine_Validate_Success(t *testing.T) {
	// Setup mock generator that returns embeddings
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			// Return different embeddings for different completions
			completion := req.Prompt()
			var embedding []float32
			switch completion {
			case "completion1":
				embedding = []float32{1.0, 0.0, 0.0}
			case "completion2":
				embedding = []float32{0.0, 1.0, 0.0}
			case "completion3":
				embedding = []float32{0.5, 0.5, 0.0}
			default:
				embedding = []float32{0.0, 0.0, 0.0}
			}
			return domain.NewGenerationResult(
				map[string]interface{}{
					"embedding": embedding,
				},
				domain.NewResultMetadata(),
			), nil
		},
	}

	engine := NewKMeanEngine("test-model", generator)

	// Create test results
	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	metadata := &domain.ProviderMetadata{
		Prompt:     "test prompt",
		Model:      "test-model",
		TokensUsed: 50,
		Cost:       0.01,
	}

	results := []TempResult{
		{Task: task, Value: "completion1", Metadata: metadata, Error: nil},
		{Task: task, Value: "completion2", Metadata: metadata, Error: nil},
		{Task: task, Value: "completion3", Metadata: metadata, Error: nil},
	}

	// Execute
	bestResult, resultMetadata, err := engine.Validate(results)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if bestResult.Value == nil {
		t.Fatal("Expected non-nil value in best result")
	}

	// The average embedding is [0.5, 0.5, 0], so completion3 should be closest
	if bestResult.Value != "completion3" {
		t.Errorf("Expected 'completion3' as best result, got: %v", bestResult.Value)
	}

	// Check metadata choices are populated
	if len(resultMetadata.Choices) != 3 {
		t.Errorf("Expected 3 choices in metadata, got: %d", len(resultMetadata.Choices))
	}
}

func TestKMeanEngine_Validate_EmptyResults(t *testing.T) {
	generator := &mockGenerator{}
	engine := NewKMeanEngine("test-model", generator)

	results := []TempResult{}

	_, _, err := engine.Validate(results)

	if err == nil {
		t.Fatal("Expected error for empty results")
	}

	if err.Error() != "no valid results to calculate k-mean" {
		t.Errorf("Expected 'no valid results to calculate k-mean' error, got: %v", err)
	}
}

func TestKMeanEngine_Validate_AllResultsHaveErrors(t *testing.T) {
	generator := &mockGenerator{}
	engine := NewKMeanEngine("test-model", generator)

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	results := []TempResult{
		{Task: task, Value: nil, Metadata: nil, Error: errors.New("error 1")},
		{Task: task, Value: nil, Metadata: nil, Error: errors.New("error 2")},
	}

	_, _, err := engine.Validate(results)

	if err == nil {
		t.Fatal("Expected error when all results have errors")
	}
}

func TestKMeanEngine_Validate_PartialErrors(t *testing.T) {
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(
				map[string]interface{}{
					"embedding": []float32{0.1, 0.2, 0.3},
				},
				domain.NewResultMetadata(),
			), nil
		},
	}

	engine := NewKMeanEngine("test-model", generator)

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	metadata := &domain.ProviderMetadata{
		Prompt: "test prompt",
		Model:  "test-model",
	}

	results := []TempResult{
		{Task: task, Value: nil, Metadata: nil, Error: errors.New("error 1")},
		{Task: task, Value: "completion2", Metadata: metadata, Error: nil},
		{Task: task, Value: "completion3", Metadata: metadata, Error: nil},
	}

	bestResult, _, err := engine.Validate(results)

	// Should succeed with 2 valid results
	if err != nil {
		t.Fatalf("Expected no error with partial errors, got: %v", err)
	}

	if bestResult.Value == nil {
		t.Fatal("Expected non-nil value in best result")
	}
}

func TestKMeanEngine_Validate_EmbeddingGenerationError(t *testing.T) {
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return nil, errors.New("embedding generation failed")
		},
	}

	engine := NewKMeanEngine("test-model", generator)

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	metadata := &domain.ProviderMetadata{
		Prompt: "test prompt",
		Model:  "test-model",
	}

	results := []TempResult{
		{Task: task, Value: "completion1", Metadata: metadata, Error: nil},
	}

	_, _, err := engine.Validate(results)

	if err == nil {
		t.Fatal("Expected error when embedding generation fails")
	}
}

func TestKMeanEngine_calculateAverageEmbedding(t *testing.T) {
	engine := &KMeanEngine{}

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	kMeanResults := []KMeanResult{
		{
			Completion: "result1",
			Embedding:  []float32{1.0, 2.0, 3.0},
			Metadata:   domain.ProviderMetadata{},
			Task:       task,
		},
		{
			Completion: "result2",
			Embedding:  []float32{3.0, 4.0, 5.0},
			Metadata:   domain.ProviderMetadata{},
			Task:       task,
		},
	}

	avgEmbedding := engine.calculateAverageEmbedding(kMeanResults)

	// Expected average: [(1+3)/2, (2+4)/2, (3+5)/2] = [2.0, 3.0, 4.0]
	expected := []float32{2.0, 3.0, 4.0}

	if len(avgEmbedding) != len(expected) {
		t.Fatalf("Expected embedding length %d, got %d", len(expected), len(avgEmbedding))
	}

	for i := range expected {
		if math.Abs(float64(avgEmbedding[i]-expected[i])) > 0.0001 {
			t.Errorf("Expected avgEmbedding[%d] = %f, got %f", i, expected[i], avgEmbedding[i])
		}
	}
}

func TestKMeanEngine_calculateAverageEmbedding_EmptyResults(t *testing.T) {
	engine := &KMeanEngine{}

	kMeanResults := []KMeanResult{}

	avgEmbedding := engine.calculateAverageEmbedding(kMeanResults)

	if avgEmbedding != nil {
		t.Error("Expected nil embedding for empty results")
	}
}

func TestKMeanEngine_euclideanDistance(t *testing.T) {
	engine := &KMeanEngine{}

	tests := []struct {
		name       string
		embedding1 []float32
		embedding2 []float32
		expected   float64
	}{
		{
			name:       "identical vectors",
			embedding1: []float32{1.0, 2.0, 3.0},
			embedding2: []float32{1.0, 2.0, 3.0},
			expected:   0.0,
		},
		{
			name:       "simple distance",
			embedding1: []float32{0.0, 0.0, 0.0},
			embedding2: []float32{3.0, 4.0, 0.0},
			expected:   5.0, // 3-4-5 triangle
		},
		{
			name:       "different lengths",
			embedding1: []float32{1.0, 2.0},
			embedding2: []float32{1.0, 2.0, 3.0},
			expected:   math.MaxFloat64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := engine.euclideanDistance(tt.embedding1, tt.embedding2)

			if math.Abs(distance-tt.expected) > 0.0001 && distance != math.MaxFloat64 {
				t.Errorf("Expected distance %f, got %f", tt.expected, distance)
			}
		})
	}
}

func TestKMeanEngine_convertToChoices(t *testing.T) {
	engine := &KMeanEngine{}

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	avgEmbedding := []float32{0.5, 0.5, 0.0}

	kMeanResults := []KMeanResult{
		{
			Completion: "close",
			Embedding:  []float32{0.5, 0.5, 0.0}, // Distance: 0
			Metadata: domain.ProviderMetadata{
				Prompt: "test prompt",
				Model:  "test-model",
			},
			Task: task,
		},
		{
			Completion: "far",
			Embedding:  []float32{10.0, 10.0, 0.0}, // Distance: much larger
			Metadata: domain.ProviderMetadata{
				Prompt: "test prompt",
				Model:  "test-model",
			},
			Task: task,
		},
	}

	choices := engine.convertToChoices(kMeanResults, avgEmbedding)

	if len(choices) != 2 {
		t.Fatalf("Expected 2 choices, got %d", len(choices))
	}

	// First choice (closest) should have higher confidence
	if choices[0].Confidence < choices[1].Confidence {
		t.Error("Expected first choice to have higher confidence than second")
	}

	// Confidence should be between 0 and 1
	for i, choice := range choices {
		if choice.Confidence < 0.0 || choice.Confidence > 1.0 {
			t.Errorf("Choice %d confidence %f is out of range [0,1]", i, choice.Confidence)
		}

		// Score should be between 0 and 100
		if choice.Score < 0 || choice.Score > 100 {
			t.Errorf("Choice %d score %d is out of range [0,100]", i, choice.Score)
		}
	}
}

func TestKMeanEngine_convertToChoices_SingleResult(t *testing.T) {
	engine := &KMeanEngine{}

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	avgEmbedding := []float32{0.5, 0.5, 0.0}

	kMeanResults := []KMeanResult{
		{
			Completion: "single",
			Embedding:  []float32{0.5, 0.5, 0.0},
			Metadata: domain.ProviderMetadata{
				Prompt: "test prompt",
				Model:  "test-model",
			},
			Task: task,
		},
	}

	choices := engine.convertToChoices(kMeanResults, avgEmbedding)

	if len(choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(choices))
	}

	// Single result should have confidence of 1.0
	if choices[0].Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", choices[0].Confidence)
	}

	if choices[0].Score != 100 {
		t.Errorf("Expected score 100, got %d", choices[0].Score)
	}
}

func TestKMeanEngine_getEmbeddingForResult_InvalidValueType(t *testing.T) {
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(
				map[string]interface{}{
					"embedding": []float32{0.1, 0.2, 0.3},
				},
				domain.NewResultMetadata(),
			), nil
		},
	}

	engine := &KMeanEngine{
		model:     "test-model",
		generator: generator,
	}

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	metadata := &domain.ProviderMetadata{
		Prompt: "test prompt",
		Model:  "test-model",
	}

	// Test with non-string value (will cause panic in current implementation)
	result := TempResult{
		Task:     task,
		Value:    12345, // Not a string
		Metadata: metadata,
		Error:    nil,
	}

	// This will panic due to type assertion failure in current implementation
	// We test that it panics as expected
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-string value, but no panic occurred")
		}
	}()

	// This should panic
	engine.getEmbeddingForResult(result)
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncate needed",
			input:    "hello world",
			maxLen:   5,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestKMeanEngine_createEmeddingDefinition(t *testing.T) {
	engine := &KMeanEngine{
		model: "test-embedding-model",
	}

	def := engine.createEmeddingDefinition()

	if def == nil {
		t.Fatal("Expected non-nil definition")
	}

	if def.Type != jsonSchema.Object {
		t.Errorf("Expected type Object, got %v", def.Type)
	}

	embeddingDef, ok := def.Properties["embedding"]
	if !ok {
		t.Fatal("Expected 'embedding' property in definition")
	}

	if embeddingDef.Type != jsonSchema.Vector {
		t.Errorf("Expected embedding type Vector, got %v", embeddingDef.Type)
	}

	if embeddingDef.Model != "test-embedding-model" {
		t.Errorf("Expected model 'test-embedding-model', got %s", embeddingDef.Model)
	}
}

func TestKMeanEngine_calculateKMean_MetadataAggregation(t *testing.T) {
	engine := &KMeanEngine{}

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	kMeanResults := []KMeanResult{
		{
			Completion: "result1",
			Embedding:  []float32{1.0, 0.0, 0.0},
			Metadata: domain.ProviderMetadata{
				Prompt:     "test prompt",
				Model:      "model1",
				TokensUsed: 50,
				Cost:       0.01,
			},
			Task: task,
		},
		{
			Completion: "result2",
			Embedding:  []float32{0.0, 1.0, 0.0},
			Metadata: domain.ProviderMetadata{
				Prompt:     "test prompt",
				Model:      "model2",
				TokensUsed: 60,
				Cost:       0.02,
			},
			Task: task,
		},
	}

	bestResult, metadata, err := engine.calculateKMean(kMeanResults)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that best result has valid metadata
	if bestResult.Metadata == nil {
		t.Fatal("Expected non-nil metadata in best result")
	}

	// Check that choices are populated
	if len(metadata.Choices) != 2 {
		t.Errorf("Expected 2 choices in metadata, got %d", len(metadata.Choices))
	}

	// Verify choice contents
	for i, choice := range metadata.Choices {
		if choice.Completion == "" {
			t.Errorf("Choice %d has empty completion", i)
		}
		if choice.Model == "" {
			t.Errorf("Choice %d has empty model", i)
		}
	}
}
