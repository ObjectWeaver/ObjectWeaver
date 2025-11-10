package epstimic

import (
	"os"
	"testing"
)

func TestGetEpstimicEngine_LLMAsJudge(t *testing.T) {
	// Save original env vars
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")
	originalLLMModel := os.Getenv("LLM_JUDGE_MODEL")

	// Restore after test
	defer func() {
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
		os.Setenv("LLM_JUDGE_MODEL", originalLLMModel)
	}()

	// Setup environment
	os.Setenv("LLM_AS_JUDGE", "true")
	os.Setenv("KMEAN_AS_JUDGE", "false")
	os.Setenv("LLM_JUDGE_MODEL", "gpt-4o")

	generator := &mockGenerator{}
	engine := GetEpstimicEngine(generator)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	// Check if it's an LLM as judge engine
	_, ok := engine.(*LLMasJudge)
	if !ok {
		t.Errorf("Expected *LLMasJudge, got %T", engine)
	}
}

func TestGetEpstimicEngine_KMeanAsJudge(t *testing.T) {
	// Save original env vars
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")
	originalKMeanModel := os.Getenv("KMEAN_EMBEDDING_MODEL")

	// Restore after test
	defer func() {
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
		os.Setenv("KMEAN_EMBEDDING_MODEL", originalKMeanModel)
	}()

	// Setup environment
	os.Setenv("LLM_AS_JUDGE", "false")
	os.Setenv("KMEAN_AS_JUDGE", "true")
	os.Setenv("KMEAN_EMBEDDING_MODEL", "text-embedding-3-large")

	generator := &mockGenerator{}
	engine := GetEpstimicEngine(generator)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	// Check if it's a KMean engine
	_, ok := engine.(*KMeanEngine)
	if !ok {
		t.Errorf("Expected *KMeanEngine, got %T", engine)
	}
}

func TestGetEpstimicEngine_KMeanTakesPrecedence(t *testing.T) {
	// Save original env vars
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")
	originalKMeanModel := os.Getenv("KMEAN_EMBEDDING_MODEL")

	// Restore after test
	defer func() {
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
		os.Setenv("KMEAN_EMBEDDING_MODEL", originalKMeanModel)
	}()

	// Setup environment - both true, K-Mean should take precedence
	os.Setenv("LLM_AS_JUDGE", "true")
	os.Setenv("KMEAN_AS_JUDGE", "true")
	os.Setenv("KMEAN_EMBEDDING_MODEL", "text-embedding-3-small")

	generator := &mockGenerator{}
	engine := GetEpstimicEngine(generator)

	// Should return KMean engine since it takes precedence
	_, ok := engine.(*KMeanEngine)
	if !ok {
		t.Errorf("Expected *KMeanEngine (should take precedence), got %T", engine)
	}
}

func TestGetEpstimicEngine_DefaultToLLMAsJudge(t *testing.T) {
	// Save original env vars
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")
	originalLLMModel := os.Getenv("LLM_JUDGE_MODEL")

	// Restore after test
	defer func() {
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
		os.Setenv("LLM_JUDGE_MODEL", originalLLMModel)
	}()

	// Setup environment - neither set
	os.Unsetenv("LLM_AS_JUDGE")
	os.Unsetenv("KMEAN_AS_JUDGE")
	os.Unsetenv("LLM_JUDGE_MODEL")

	generator := &mockGenerator{}
	engine := GetEpstimicEngine(generator)

	// Should default to LLM as judge
	_, ok := engine.(*LLMasJudge)
	if !ok {
		t.Errorf("Expected default *LLMasJudge, got %T", engine)
	}
}

func TestGetEpstimicEngine_DefaultLLMModel(t *testing.T) {
	// Save original env vars
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")
	originalLLMModel := os.Getenv("LLM_JUDGE_MODEL")

	// Restore after test
	defer func() {
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
		os.Setenv("LLM_JUDGE_MODEL", originalLLMModel)
	}()

	// Setup environment - LLM as judge without model specified
	os.Setenv("LLM_AS_JUDGE", "true")
	os.Setenv("KMEAN_AS_JUDGE", "false")
	os.Unsetenv("LLM_JUDGE_MODEL")

	generator := &mockGenerator{}
	engine := GetEpstimicEngine(generator)

	llmEngine, ok := engine.(*LLMasJudge)
	if !ok {
		t.Fatalf("Expected *LLMasJudge, got %T", engine)
	}

	// Should use default model
	if llmEngine.model != "gpt-4o-mini" {
		t.Errorf("Expected default model 'gpt-4o-mini', got %s", llmEngine.model)
	}
}

func TestGetEpstimicEngine_DefaultKMeanModel(t *testing.T) {
	// Save original env vars
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")
	originalKMeanModel := os.Getenv("KMEAN_EMBEDDING_MODEL")

	// Restore after test
	defer func() {
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
		os.Setenv("KMEAN_EMBEDDING_MODEL", originalKMeanModel)
	}()

	// Setup environment - K-Mean without model specified
	os.Setenv("LLM_AS_JUDGE", "false")
	os.Setenv("KMEAN_AS_JUDGE", "true")
	os.Unsetenv("KMEAN_EMBEDDING_MODEL")

	generator := &mockGenerator{}
	engine := GetEpstimicEngine(generator)

	kMeanEngine, ok := engine.(*KMeanEngine)
	if !ok {
		t.Fatalf("Expected *KMeanEngine, got %T", engine)
	}

	// Should use default embedding model
	if kMeanEngine.model != "text-embedding-3-small" {
		t.Errorf("Expected default model 'text-embedding-3-small', got %s", kMeanEngine.model)
	}
}

func TestGetEpstimicEngine_InvalidBooleanValues(t *testing.T) {
	// Save original env vars
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")

	// Restore after test
	defer func() {
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
	}()

	// Setup environment with invalid boolean values
	os.Setenv("LLM_AS_JUDGE", "maybe")
	os.Setenv("KMEAN_AS_JUDGE", "perhaps")

	generator := &mockGenerator{}
	engine := GetEpstimicEngine(generator)

	// Should default to LLM as judge on parsing error
	_, ok := engine.(*LLMasJudge)
	if !ok {
		t.Errorf("Expected default *LLMasJudge on invalid boolean, got %T", engine)
	}
}

func TestGetEpstimicOrchestrator_Success(t *testing.T) {
	// Save original env vars
	originalWorkerCount := os.Getenv("EPSTIMIC_WORKER_COUNT")

	// Restore after test
	defer func() {
		os.Setenv("EPSTIMIC_WORKER_COUNT", originalWorkerCount)
	}()

	// Setup environment
	os.Setenv("EPSTIMIC_WORKER_COUNT", "5")

	generator := &mockGenerator{}
	orchestrator := GetEpstimicOrchestrator(generator)

	if orchestrator == nil {
		t.Fatal("Expected non-nil orchestrator")
	}

	if orchestrator.workerCount != 5 {
		t.Errorf("Expected workerCount 5, got %d", orchestrator.workerCount)
	}

	if orchestrator.epstimicEngine == nil {
		t.Error("Expected non-nil epistemic engine")
	}
}

func TestGetEpstimicOrchestrator_DefaultWorkerCount(t *testing.T) {
	// Save original env vars
	originalWorkerCount := os.Getenv("EPSTIMIC_WORKER_COUNT")

	// Restore after test
	defer func() {
		os.Setenv("EPSTIMIC_WORKER_COUNT", originalWorkerCount)
	}()

	// Setup environment - no worker count specified
	os.Unsetenv("EPSTIMIC_WORKER_COUNT")

	generator := &mockGenerator{}
	orchestrator := GetEpstimicOrchestrator(generator)

	// Should use default worker count of 3
	if orchestrator.workerCount != 3 {
		t.Errorf("Expected default workerCount 3, got %d", orchestrator.workerCount)
	}
}

func TestGetEpstimicOrchestrator_InvalidWorkerCount(t *testing.T) {
	// Save original env vars
	originalWorkerCount := os.Getenv("EPSTIMIC_WORKER_COUNT")

	// Restore after test
	defer func() {
		os.Setenv("EPSTIMIC_WORKER_COUNT", originalWorkerCount)
	}()

	// Setup environment with invalid worker count
	os.Setenv("EPSTIMIC_WORKER_COUNT", "not-a-number")

	generator := &mockGenerator{}
	orchestrator := GetEpstimicOrchestrator(generator)

	// Should default to 3 on parsing error
	if orchestrator.workerCount != 3 {
		t.Errorf("Expected default workerCount 3 on invalid value, got %d", orchestrator.workerCount)
	}
}

func TestGetEpstimicOrchestrator_NegativeWorkerCount(t *testing.T) {
	// Save original env vars
	originalWorkerCount := os.Getenv("EPSTIMIC_WORKER_COUNT")

	// Restore after test
	defer func() {
		os.Setenv("EPSTIMIC_WORKER_COUNT", originalWorkerCount)
	}()

	// Setup environment with negative worker count
	os.Setenv("EPSTIMIC_WORKER_COUNT", "-5")

	generator := &mockGenerator{}
	orchestrator := GetEpstimicOrchestrator(generator)

	// Should default to 3 on negative value
	if orchestrator.workerCount != 3 {
		t.Errorf("Expected default workerCount 3 on negative value, got %d", orchestrator.workerCount)
	}
}

func TestGetEpstimicOrchestrator_ZeroWorkerCount(t *testing.T) {
	// Save original env vars
	originalWorkerCount := os.Getenv("EPSTIMIC_WORKER_COUNT")

	// Restore after test
	defer func() {
		os.Setenv("EPSTIMIC_WORKER_COUNT", originalWorkerCount)
	}()

	// Setup environment with zero worker count
	os.Setenv("EPSTIMIC_WORKER_COUNT", "0")

	generator := &mockGenerator{}
	orchestrator := GetEpstimicOrchestrator(generator)

	// Should default to 3 on zero value
	if orchestrator.workerCount != 3 {
		t.Errorf("Expected default workerCount 3 on zero value, got %d", orchestrator.workerCount)
	}
}

func TestGetEpstimicOrchestrator_LargeWorkerCount(t *testing.T) {
	// Save original env vars
	originalWorkerCount := os.Getenv("EPSTIMIC_WORKER_COUNT")

	// Restore after test
	defer func() {
		os.Setenv("EPSTIMIC_WORKER_COUNT", originalWorkerCount)
	}()

	// Setup environment with large worker count
	os.Setenv("EPSTIMIC_WORKER_COUNT", "100")

	generator := &mockGenerator{}
	orchestrator := GetEpstimicOrchestrator(generator)

	// Should accept large worker count
	if orchestrator.workerCount != 100 {
		t.Errorf("Expected workerCount 100, got %d", orchestrator.workerCount)
	}
}

func TestGetEpstimicOrchestrator_IntegrationWithEngine(t *testing.T) {
	// Save original env vars
	originalWorkerCount := os.Getenv("EPSTIMIC_WORKER_COUNT")
	originalLLMAsJudge := os.Getenv("LLM_AS_JUDGE")
	originalKMeanAsJudge := os.Getenv("KMEAN_AS_JUDGE")

	// Restore after test
	defer func() {
		os.Setenv("EPSTIMIC_WORKER_COUNT", originalWorkerCount)
		os.Setenv("LLM_AS_JUDGE", originalLLMAsJudge)
		os.Setenv("KMEAN_AS_JUDGE", originalKMeanAsJudge)
	}()

	// Setup environment for K-Mean engine
	os.Setenv("EPSTIMIC_WORKER_COUNT", "7")
	os.Setenv("LLM_AS_JUDGE", "false")
	os.Setenv("KMEAN_AS_JUDGE", "true")

	generator := &mockGenerator{}
	orchestrator := GetEpstimicOrchestrator(generator)

	if orchestrator.workerCount != 7 {
		t.Errorf("Expected workerCount 7, got %d", orchestrator.workerCount)
	}

	// Verify the correct engine type is used
	_, ok := orchestrator.epstimicEngine.(*KMeanEngine)
	if !ok {
		t.Errorf("Expected *KMeanEngine in orchestrator, got %T", orchestrator.epstimicEngine)
	}
}
