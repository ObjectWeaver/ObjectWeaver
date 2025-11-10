package epstimic

import (
	"errors"
	"sync"
	"testing"
	"time"

	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// Mock EpstimicEngine for testing
type mockEpstimicEngine struct {
	validateFunc func(results []TempResult) (TempResult, domain.ProviderMetadata, error)
}

func (m *mockEpstimicEngine) Validate(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
	if m.validateFunc != nil {
		return m.validateFunc(results)
	}
	// Default implementation: return first result
	if len(results) == 0 {
		return TempResult{}, domain.ProviderMetadata{}, errors.New("no results to validate")
	}
	metadata := domain.ProviderMetadata{
		TokensUsed: 100,
		Cost:       0.01,
		Model:      "test-model",
		Choices:    []domain.Choice{},
	}
	return results[0], metadata, nil
}

func TestOrchestrator_EpstimicValidation_Success(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    3,
	}

	// Create test task
	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 3,
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	// Mock generate function
	generateCallCount := 0
	var mu sync.Mutex
	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		mu.Lock()
		generateCallCount++
		mu.Unlock()
		return "test-value", &domain.ProviderMetadata{
			TokensUsed: 50,
			Cost:       0.005,
			Model:      "test-model",
		}, nil
	}

	// Execute
	result, metadata, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if metadata == nil {
		t.Fatal("Expected metadata, got nil")
	}
	if result.Key() != "testField" {
		t.Errorf("Expected key 'testField', got: %s", result.Key())
	}
	if result.Value() != "test-value" {
		t.Errorf("Expected value 'test-value', got: %v", result.Value())
	}
	if generateCallCount != 3 {
		t.Errorf("Expected generate to be called 3 times, got: %d", generateCallCount)
	}
}

func TestOrchestrator_EpstimicValidation_DefaultWorkerCount(t *testing.T) {
	// Setup with default worker count (no Judges specified)
	engine := &mockEpstimicEngine{}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    5, // default
	}

	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 0, // No judges specified
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	generateCallCount := 0
	var mu sync.Mutex
	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		mu.Lock()
		generateCallCount++
		mu.Unlock()
		return "test-value", &domain.ProviderMetadata{}, nil
	}

	// Execute
	_, _, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if generateCallCount != 5 {
		t.Errorf("Expected generate to be called 5 times (default), got: %d", generateCallCount)
	}
}

func TestOrchestrator_EpstimicValidation_WithJudgesOverride(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    5,
	}

	// Task with specific number of judges
	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 7, // Override default
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	generateCallCount := 0
	var mu sync.Mutex
	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		mu.Lock()
		generateCallCount++
		mu.Unlock()
		return "test-value", &domain.ProviderMetadata{}, nil
	}

	// Execute
	_, _, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if generateCallCount != 7 {
		t.Errorf("Expected generate to be called 7 times (judges count), got: %d", generateCallCount)
	}
}

func TestOrchestrator_EpstimicValidation_PartialGenerateErrors(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{
		validateFunc: func(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
			// Should receive only successful results
			if len(results) != 2 {
				t.Errorf("Expected 2 successful results, got: %d", len(results))
			}
			metadata := domain.ProviderMetadata{
				TokensUsed: 100,
				Cost:       0.01,
				Model:      "test-model",
			}
			return results[0], metadata, nil
		},
	}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    3,
	}

	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 3,
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	// Mock generate function that fails on first call
	callCount := 0
	var mu sync.Mutex
	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		mu.Lock()
		callCount++
		currentCall := callCount
		mu.Unlock()

		if currentCall == 1 {
			return nil, nil, errors.New("generation failed")
		}
		return "test-value", &domain.ProviderMetadata{}, nil
	}

	// Execute
	result, _, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result despite partial errors")
	}
}

func TestOrchestrator_EpstimicValidation_AllGenerateErrors(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{
		validateFunc: func(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
			// Should receive empty results array
			if len(results) != 0 {
				t.Errorf("Expected 0 results, got: %d", len(results))
			}
			return TempResult{}, domain.ProviderMetadata{}, errors.New("no valid results")
		},
	}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    3,
	}

	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 3,
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	// Mock generate function that always fails
	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		return nil, nil, errors.New("generation failed")
	}

	// Execute
	result, metadata, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err == nil {
		t.Fatal("Expected error when all generations fail")
	}
	if result != nil {
		t.Error("Expected nil result on error")
	}
	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOrchestrator_EpstimicValidation_EngineValidationError(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{
		validateFunc: func(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
			return TempResult{}, domain.ProviderMetadata{}, errors.New("validation error")
		},
	}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    3,
	}

	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 3,
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		return "test-value", &domain.ProviderMetadata{}, nil
	}

	// Execute
	result, metadata, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err == nil {
		t.Fatal("Expected error from engine validation")
	}
	if err.Error() != "validation error" {
		t.Errorf("Expected 'validation error', got: %v", err)
	}
	if result != nil {
		t.Error("Expected nil result on error")
	}
	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOrchestrator_EpstimicValidation_MetadataMapping(t *testing.T) {
	// Setup
	expectedMetadata := domain.ProviderMetadata{
		TokensUsed: 250,
		Cost:       0.05,
		Model:      "gpt-4",
		Choices: []domain.Choice{
			{Prompt: "choice1"},
			{Prompt: "choice2"},
		},
	}

	engine := &mockEpstimicEngine{
		validateFunc: func(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
			return results[0], expectedMetadata, nil
		},
	}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    2,
	}

	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 2,
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		return "test-value", &domain.ProviderMetadata{}, nil
	}

	// Execute
	result, providerMeta, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check provider metadata
	if providerMeta.TokensUsed != expectedMetadata.TokensUsed {
		t.Errorf("Expected TokensUsed %d, got: %d", expectedMetadata.TokensUsed, providerMeta.TokensUsed)
	}
	if providerMeta.Cost != expectedMetadata.Cost {
		t.Errorf("Expected Cost %f, got: %f", expectedMetadata.Cost, providerMeta.Cost)
	}
	if providerMeta.Model != expectedMetadata.Model {
		t.Errorf("Expected Model %s, got: %s", expectedMetadata.Model, providerMeta.Model)
	}

	// Check result metadata
	resultMeta := result.Metadata()
	if resultMeta.TokensUsed != expectedMetadata.TokensUsed {
		t.Errorf("Expected result TokensUsed %d, got: %d", expectedMetadata.TokensUsed, resultMeta.TokensUsed)
	}
	if resultMeta.Cost != expectedMetadata.Cost {
		t.Errorf("Expected result Cost %f, got: %f", expectedMetadata.Cost, resultMeta.Cost)
	}
	if resultMeta.ModelUsed != expectedMetadata.Model {
		t.Errorf("Expected result ModelUsed %s, got: %s", expectedMetadata.Model, resultMeta.ModelUsed)
	}
	if len(resultMeta.Choices) != len(expectedMetadata.Choices) {
		t.Errorf("Expected %d choices, got: %d", len(expectedMetadata.Choices), len(resultMeta.Choices))
	}
}

func TestOrchestrator_EpstimicValidation_SeedGeneration(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    3,
	}

	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 3,
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	// Track seed values
	seeds := make([]*int, 0)
	var mu sync.Mutex

	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		mu.Lock()
		seeds = append(seeds, task.Definition().ModelConfig.Seed)
		mu.Unlock()
		return "test-value", &domain.ProviderMetadata{}, nil
	}

	// Execute
	_, _, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// First worker should have original seed, others should have new seeds
	// Note: We can't easily test the actual seed generation without more complex mocking
	// but we can verify the function completes successfully
	if len(seeds) != 3 {
		t.Errorf("Expected 3 seed captures, got: %d", len(seeds))
	}
}

func TestOrchestrator_EpstimicValidation_ConcurrentExecution(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    10,
	}

	definition := &jsonSchema.Definition{
		Type: "string",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 10,
		},
	}
	task := domain.NewFieldTask("testField", definition, nil)
	context := &domain.ExecutionContext{}

	// Track concurrent execution
	activeWorkers := 0
	maxConcurrent := 0
	var mu sync.Mutex

	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		mu.Lock()
		activeWorkers++
		if activeWorkers > maxConcurrent {
			maxConcurrent = activeWorkers
		}
		mu.Unlock()

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		activeWorkers--
		mu.Unlock()

		return "test-value", &domain.ProviderMetadata{}, nil
	}

	// Execute
	_, _, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that multiple workers ran concurrently
	if maxConcurrent < 2 {
		t.Errorf("Expected concurrent execution (max concurrent: %d)", maxConcurrent)
	}
}

func TestOrchestrator_EpstimicValidation_TaskResultFields(t *testing.T) {
	// Setup
	engine := &mockEpstimicEngine{
		validateFunc: func(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
			metadata := domain.ProviderMetadata{
				TokensUsed: 150,
				Cost:       0.03,
				Model:      "test-model-v2",
			}
			return results[0], metadata, nil
		},
	}
	orchestrator := &Orchestrator{
		epstimicEngine: engine,
		workerCount:    2,
	}

	definition := &jsonSchema.Definition{
		Type: "object",
		Epistemic: jsonSchema.EpistemicValidation{
			Judges: 2,
		},
	}
	task := domain.NewFieldTask("complexField", definition, nil)
	context := &domain.ExecutionContext{}

	expectedValue := map[string]interface{}{
		"name": "test",
		"age":  25,
	}

	generate := func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error) {
		return expectedValue, &domain.ProviderMetadata{}, nil
	}

	// Execute
	result, _, err := orchestrator.EpstimicValidation(task, context, generate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.TaskID() != task.ID() {
		t.Errorf("Expected TaskID %s, got: %s", task.ID(), result.TaskID())
	}
	if result.Key() != "complexField" {
		t.Errorf("Expected Key 'complexField', got: %s", result.Key())
	}
	if !result.IsSuccess() {
		t.Error("Expected successful result")
	}

	// Check value
	resultValue, ok := result.Value().(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{} value, got: %T", result.Value())
	}
	if resultValue["name"] != "test" {
		t.Errorf("Expected name 'test', got: %v", resultValue["name"])
	}
	if resultValue["age"] != 25 {
		t.Errorf("Expected age 25, got: %v", resultValue["age"])
	}
}

func TestTempResult_Structure(t *testing.T) {
	// Test TempResult structure used internally
	definition := &jsonSchema.Definition{Type: "string"}
	task := domain.NewFieldTask("test", definition, nil)
	metadata := &domain.ProviderMetadata{
		TokensUsed: 100,
		Cost:       0.01,
	}

	// Success case
	successResult := TempResult{
		Task:     task,
		Value:    "success-value",
		Metadata: metadata,
		Error:    nil,
	}

	if successResult.Error != nil {
		t.Error("Expected no error in success result")
	}
	if successResult.Value != "success-value" {
		t.Errorf("Expected value 'success-value', got: %v", successResult.Value)
	}

	// Error case
	errResult := TempResult{
		Task:     task,
		Value:    nil,
		Metadata: nil,
		Error:    errors.New("test error"),
	}

	if errResult.Error == nil {
		t.Error("Expected error in error result")
	}
	if errResult.Value != nil {
		t.Error("Expected nil value in error result")
	}
	if errResult.Metadata != nil {
		t.Error("Expected nil metadata in error result")
	}
}
