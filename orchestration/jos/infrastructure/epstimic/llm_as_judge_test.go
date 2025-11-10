package epstimic

import (
	"errors"
	"objectweaver/orchestration/jos/domain"
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestNewLLMAsJudge(t *testing.T) {
	generator := &mockGenerator{}
	model := "gpt-4o-mini"

	engine := NewLLMAsJudge(model, generator)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	llmEngine, ok := engine.(*LLMasJudge)
	if !ok {
		t.Fatal("Expected *LLMasJudge type")
	}

	if llmEngine.model != model {
		t.Errorf("Expected model %s, got %s", model, llmEngine.model)
	}

	if llmEngine.generator == nil {
		t.Error("Expected non-nil generator")
	}
}

func TestLLMasJudge_Validate_Success(t *testing.T) {
	// Setup mock generator that returns scores based on completion content
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			// Parse the prompt to determine which completion we're scoring
			prompt := req.Prompt()
			var completeness, correctness int

			// Determine score based on which completion is in the prompt
			if containsString(prompt, "completion1") {
				completeness, correctness = 80, 75
			} else if containsString(prompt, "completion2") {
				completeness, correctness = 90, 95
			} else if containsString(prompt, "completion3") {
				completeness, correctness = 70, 65
			} else {
				completeness, correctness = 50, 50
			}

			return domain.NewGenerationResult(
				map[string]interface{}{
					"completeness": completeness,
					"correctness":  correctness,
				},
				domain.NewResultMetadata(),
			), nil
		},
	}

	engine := NewLLMAsJudge("gpt-4o-mini", generator)

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

	// The second result should be best (score: (90+95)/2 = 92.5)
	if bestResult.Value != "completion2" {
		t.Errorf("Expected 'completion2' as best result, got: %v", bestResult.Value)
	}

	// Check metadata choices are populated
	if len(resultMetadata.Choices) != 3 {
		t.Errorf("Expected 3 choices in metadata, got: %d", len(resultMetadata.Choices))
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLLMasJudge_Validate_EmptyResults(t *testing.T) {
	generator := &mockGenerator{}
	engine := NewLLMAsJudge("gpt-4o-mini", generator)

	results := []TempResult{}

	bestResult, _, err := engine.Validate(results)

	if err == nil {
		t.Fatal("Expected error for empty results")
	}

	if err.Error() != "no valid results to validate" {
		t.Errorf("Expected 'no valid results to validate' error, got: %v", err)
	}

	if bestResult.Metadata != nil {
		t.Error("Expected nil metadata for error case")
	}
}

func TestLLMasJudge_Validate_AllResultsHaveErrors(t *testing.T) {
	generator := &mockGenerator{}
	engine := NewLLMAsJudge("gpt-4o-mini", generator)

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

func TestLLMasJudge_Validate_ScoringFailure(t *testing.T) {
	// Setup mock generator that fails to score
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return nil, errors.New("scoring failed")
		},
	}

	engine := NewLLMAsJudge("gpt-4o-mini", generator)

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
		t.Fatal("Expected error when scoring fails")
	}
}

func TestLLMasJudge_Validate_PartialScoringFailures(t *testing.T) {
	// Setup mock generator that fails for completion1 specifically
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			prompt := req.Prompt()
			// Fail for completion1
			if containsString(prompt, "completion1") {
				return nil, errors.New("scoring failed")
			}
			return domain.NewGenerationResult(
				map[string]interface{}{
					"completeness": 85,
					"correctness":  90,
				},
				domain.NewResultMetadata(),
			), nil
		},
	}

	engine := NewLLMAsJudge("gpt-4o-mini", generator)

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	metadata := &domain.ProviderMetadata{
		Prompt: "test prompt",
		Model:  "test-model",
	}

	results := []TempResult{
		{Task: task, Value: "completion1", Metadata: metadata, Error: nil},
		{Task: task, Value: "completion2", Metadata: metadata, Error: nil},
	}

	bestResult, _, err := engine.Validate(results)

	// Should succeed with at least one valid score
	if err != nil {
		t.Fatalf("Expected no error with partial scoring failures, got: %v", err)
	}

	// Should return completion2 since completion1 scoring failed
	if bestResult.Value != "completion2" {
		t.Errorf("Expected 'completion2' as best result, got: %v", bestResult.Value)
	}
}

func TestScoredResults_Best(t *testing.T) {
	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	scores := ScoredResults{
		{
			Completeness: 70,
			Correctness:  75,
			Score:        72,
			Result: TempResult{
				Task:  task,
				Value: "result1",
			},
		},
		{
			Completeness: 90,
			Correctness:  95,
			Score:        92,
			Result: TempResult{
				Task:  task,
				Value: "result2",
			},
		},
		{
			Completeness: 80,
			Correctness:  85,
			Score:        82,
			Result: TempResult{
				Task:  task,
				Value: "result3",
			},
		},
	}

	best, score := scores.Best()

	if score != 92 {
		t.Errorf("Expected best score 92, got %d", score)
	}

	if best.Value != "result2" {
		t.Errorf("Expected 'result2' as best, got: %v", best.Value)
	}
}

func TestScoredResults_Best_Empty(t *testing.T) {
	scores := ScoredResults{}

	best, score := scores.Best()

	if score != 0 {
		t.Errorf("Expected score 0 for empty results, got %d", score)
	}

	if best.Value != nil {
		t.Error("Expected nil value for empty results")
	}
}

func TestScoredResults_Best_SingleResult(t *testing.T) {
	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	scores := ScoredResults{
		{
			Completeness: 80,
			Correctness:  85,
			Score:        82,
			Result: TempResult{
				Task:  task,
				Value: "only-result",
			},
		},
	}

	best, score := scores.Best()

	if score != 82 {
		t.Errorf("Expected score 82, got %d", score)
	}

	if best.Value != "only-result" {
		t.Errorf("Expected 'only-result', got: %v", best.Value)
	}
}

func TestLLMasJudge_createJudgePrompt(t *testing.T) {
	judge := &LLMasJudge{
		model: "gpt-4o-mini",
	}

	prompt := judge.createJudgePrompt("original prompt", "completion text")

	expectedPrompt := "Prompt:\noriginal prompt\n\nCompletion:\ncompletion text"

	if prompt != expectedPrompt {
		t.Errorf("Expected prompt:\n%s\n\nGot:\n%s", expectedPrompt, prompt)
	}
}

func TestLLMasJudge_getJudgeDefinition(t *testing.T) {
	judge := &LLMasJudge{
		model: "gpt-4o-mini",
	}

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	result := TempResult{
		Task: task,
		Metadata: &domain.ProviderMetadata{
			Model: "original-model",
		},
	}

	// Test with judge's model
	def := judge.getJudgeDefinition(result)

	if def == nil {
		t.Fatal("Expected non-nil definition")
	}

	if def.Type != jsonSchema.Object {
		t.Errorf("Expected type Object, got %v", def.Type)
	}

	if def.Model != "gpt-4o-mini" {
		t.Errorf("Expected model 'gpt-4o-mini', got %s", def.Model)
	}

	// Check properties
	if _, ok := def.Properties["completeness"]; !ok {
		t.Error("Expected 'completeness' property")
	}

	if _, ok := def.Properties["correctness"]; !ok {
		t.Error("Expected 'correctness' property")
	}

	// Check epistemic is not active (avoid recursion)
	if def.Epistemic.Active {
		t.Error("Expected epistemic validation to be inactive")
	}
}

func TestLLMasJudge_getJudgeDefinition_FallbackToResultModel(t *testing.T) {
	judge := &LLMasJudge{
		model: "", // No model set
	}

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	result := TempResult{
		Task: task,
		Metadata: &domain.ProviderMetadata{
			Model: "result-model",
		},
	}

	def := judge.getJudgeDefinition(result)

	// Should use result's model as fallback
	if def.Model != "result-model" {
		t.Errorf("Expected model 'result-model', got %s", def.Model)
	}
}

func TestConvertToChoice(t *testing.T) {
	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	scores := ScoredResults{
		{
			Completeness: 80,
			Correctness:  85,
			Score:        82,
			Result: TempResult{
				Task:  task,
				Value: "result1",
				Metadata: &domain.ProviderMetadata{
					Prompt: "test prompt",
					Model:  "model1",
				},
			},
		},
		{
			Completeness: 90,
			Correctness:  95,
			Score:        92,
			Result: TempResult{
				Task:  task,
				Value: "result2",
				Metadata: &domain.ProviderMetadata{
					Prompt: "test prompt",
					Model:  "model2",
				},
			},
		},
	}

	choices := convertToChoice(scores)

	if len(choices) != 2 {
		t.Fatalf("Expected 2 choices, got %d", len(choices))
	}

	// Check first choice
	if choices[0].Prompt != "test prompt" {
		t.Errorf("Expected prompt 'test prompt', got %s", choices[0].Prompt)
	}

	if choices[0].Completion != "result1" {
		t.Errorf("Expected completion 'result1', got %v", choices[0].Completion)
	}

	if choices[0].Model != "model1" {
		t.Errorf("Expected model 'model1', got %s", choices[0].Model)
	}

	if choices[0].Score != 82 {
		t.Errorf("Expected score 82, got %d", choices[0].Score)
	}

	// Confidence should be score / 100
	expectedConfidence := 82.0 / 100.0
	if choices[0].Confidence != expectedConfidence {
		t.Errorf("Expected confidence %f, got %f", expectedConfidence, choices[0].Confidence)
	}
}

func TestConvertToChoice_SkipsNilMetadata(t *testing.T) {
	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	scores := ScoredResults{
		{
			Completeness: 80,
			Correctness:  85,
			Score:        82,
			Result: TempResult{
				Task:     task,
				Value:    "result1",
				Metadata: nil, // Nil metadata
			},
		},
		{
			Completeness: 90,
			Correctness:  95,
			Score:        92,
			Result: TempResult{
				Task:  task,
				Value: "result2",
				Metadata: &domain.ProviderMetadata{
					Prompt: "test prompt",
					Model:  "model2",
				},
			},
		},
	}

	choices := convertToChoice(scores)

	// Should skip the result with nil metadata
	if len(choices) != 1 {
		t.Fatalf("Expected 1 choice (skipping nil metadata), got %d", len(choices))
	}

	if choices[0].Completion != "result2" {
		t.Errorf("Expected completion 'result2', got %v", choices[0].Completion)
	}
}

func TestConvertToChoice_SkipsNilTask(t *testing.T) {
	scores := ScoredResults{
		{
			Completeness: 80,
			Correctness:  85,
			Score:        82,
			Result: TempResult{
				Task:  nil, // Nil task
				Value: "result1",
				Metadata: &domain.ProviderMetadata{
					Prompt: "test prompt",
					Model:  "model1",
				},
			},
		},
	}

	choices := convertToChoice(scores)

	// Should skip the result with nil task
	if len(choices) != 0 {
		t.Fatalf("Expected 0 choices (skipping nil task), got %d", len(choices))
	}
}

func TestLLMasJudge_Validate_MetadataAggregation(t *testing.T) {
	// Setup mock generator
	generator := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(
				map[string]interface{}{
					"completeness": 85,
					"correctness":  90,
				},
				domain.NewResultMetadata(),
			), nil
		},
	}

	engine := NewLLMAsJudge("gpt-4o-mini", generator)

	task := domain.NewFieldTask("testField", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	results := []TempResult{
		{
			Task:  task,
			Value: "result1",
			Metadata: &domain.ProviderMetadata{
				Prompt:     "test prompt",
				Model:      "model1",
				TokensUsed: 50,
				Cost:       0.01,
			},
			Error: nil,
		},
		{
			Task:  task,
			Value: "result2",
			Metadata: &domain.ProviderMetadata{
				Prompt:     "test prompt",
				Model:      "model2",
				TokensUsed: 60,
				Cost:       0.02,
			},
			Error: nil,
		},
	}

	bestResult, metadata, err := engine.Validate(results)

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
}
