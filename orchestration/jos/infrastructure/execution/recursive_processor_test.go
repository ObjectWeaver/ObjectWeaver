package execution

import (
	"context"
	"errors"
	"fmt"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
	"os"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

// mockTypeProcessor for testing base processor delegation
type mockTypeProcessor struct {
	canProcessFunc func(jsonSchema.DataType) bool
	processFunc    func(context.Context, *domain.FieldTask, *domain.ExecutionContext) (*domain.TaskResult, error)
}

func (m *mockTypeProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	if m.canProcessFunc != nil {
		return m.canProcessFunc(schemaType)
	}
	return true
}

func (m *mockTypeProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, task, execContext)
	}
	return domain.NewTaskResult(task.ID(), task.Key(), "mock result", domain.NewResultMetadata()), nil
}

func TestNewRecursiveLoopProcessor(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	decisionProcessor := NewDecisionProcessor(gen)

	processor := NewRecursiveLoopProcessor(baseProcessor, gen, decisionProcessor)

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}
	if processor.baseProcessor == nil {
		t.Error("Expected baseProcessor to be set")
	}
	if processor.generator == nil {
		t.Error("Expected generator to be set")
	}
	if processor.decisionProcessor == nil {
		t.Error("Expected decisionProcessor to be set")
	}
}

func TestRecursiveLoopProcessor_CanProcess(t *testing.T) {
	baseProcessor := &mockTypeProcessor{
		canProcessFunc: func(dt jsonSchema.DataType) bool {
			return dt == jsonSchema.String
		},
	}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	// Should delegate to base processor
	if !processor.CanProcess(jsonSchema.String) {
		t.Error("Expected to be able to process string type")
	}
	if processor.CanProcess(jsonSchema.Number) {
		t.Error("Expected not to be able to process number type")
	}
}

func TestRecursiveLoopProcessor_Process_SingleIteration(t *testing.T) {
	callCount := 0
	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			callCount++
			return domain.NewTaskResult(task.ID(), task.Key(), fmt.Sprintf("result-%d", callCount), domain.NewResultMetadata()), nil
		},
	}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx := context.Background()

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations: 1,
		Selection:     jsonSchema.SelectLatest,
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	result, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 iteration, got %d", callCount)
	}

	if result.Value() != "result-1" {
		t.Errorf("Expected 'result-1', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_Process_MultipleIterations(t *testing.T) {
	callCount := 0
	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			callCount++
			return domain.NewTaskResult(task.ID(), task.Key(), fmt.Sprintf("result-%d", callCount), domain.NewResultMetadata()), nil
		},
	}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx := context.Background()

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations: 3,
		Selection:     jsonSchema.SelectLatest,
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	result, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 iterations, got %d", callCount)
	}

	if result.Value() != "result-3" {
		t.Errorf("Expected 'result-3', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_Process_ContextCancelled(t *testing.T) {
	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			return domain.NewTaskResult(task.ID(), task.Key(), "result", domain.NewResultMetadata()), nil
		},
	}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations: 3,
		Selection:     jsonSchema.SelectLatest,
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	_, err := processor.Process(ctx, task, execContext)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestRecursiveLoopProcessor_Process_ContextCancelledDuringIteration(t *testing.T) {
	callCount := 0
	ctx, cancel := context.WithCancel(context.Background())

	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			callCount++
			if callCount == 2 {
				cancel() // Cancel after second iteration
			}
			return domain.NewTaskResult(task.ID(), task.Key(), fmt.Sprintf("result-%d", callCount), domain.NewResultMetadata()), nil
		},
	}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations: 5,
		Selection:     jsonSchema.SelectLatest,
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	result, err := processor.Process(ctx, task, execContext)

	// When cancelled with existing results, should return best result without error
	// (graceful degradation - return what we have)
	if err != nil {
		t.Errorf("Expected no error when returning partial results, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result to be returned with partial results")
	}
	// Should have completed at least 2 iterations before cancellation
	if callCount < 2 {
		t.Errorf("Expected at least 2 iterations, got %d", callCount)
	}
	// Result should be from one of the completed iterations
	if result.Value() != "result-1" && result.Value() != "result-2" {
		t.Errorf("Expected result from completed iterations, got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_Process_IterationError(t *testing.T) {
	callCount := 0
	expectedErr := errors.New("iteration failed")

	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			callCount++
			if callCount == 2 {
				return nil, expectedErr
			}
			return domain.NewTaskResult(task.ID(), task.Key(), fmt.Sprintf("result-%d", callCount), domain.NewResultMetadata()), nil
		},
	}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx := context.Background()

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations: 3,
		Selection:     jsonSchema.SelectLatest,
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	_, err := processor.Process(ctx, task, execContext)
	if err == nil {
		t.Error("Expected error from failed iteration")
	}
	if callCount != 2 {
		t.Errorf("Expected 2 iterations before error, got %d", callCount)
	}
}

func TestRecursiveLoopProcessor_Process_WithScoring(t *testing.T) {
	callCount := 0
	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			callCount++
			return domain.NewTaskResult(task.ID(), task.Key(), fmt.Sprintf("result-%d", callCount), domain.NewResultMetadata()), nil
		},
	}

	// Mock generator that returns scores
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			// Return different scores based on call
			score := float64(callCount * 10)
			return domain.NewGenerationResult(map[string]interface{}{
				"quality": score,
			}, nil), nil
		},
	}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx := context.Background()

	scoringCriteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality score",
			},
		},
		AggregationMethod: jsonSchema.AggregateWeightedAverage,
	}

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations: 3,
		Selection:     jsonSchema.SelectHighestScore,
	}

	fieldDef := &jsonSchema.Definition{
		Type:            jsonSchema.String,
		RecursiveLoop:   recursiveLoop,
		ScoringCriteria: scoringCriteria,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	result, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 iterations, got %d", callCount)
	}

	// Should select the highest scoring result (result-3 with score 30)
	if result.Value() != "result-3" {
		t.Errorf("Expected highest scoring 'result-3', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_Process_VerboseLogging(t *testing.T) {
	// Set verbose mode
	os.Setenv("VERBOSE", "true")
	defer os.Unsetenv("VERBOSE")

	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx := context.Background()

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations: 2,
		Selection:     jsonSchema.SelectLatest,
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	_, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test passes if no panic occurs - verbose logging should not break functionality
}

func TestRecursiveLoopProcessor_ShouldTerminate_NoBranches(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx := context.Background()

	// Empty termination point with no branches
	terminationPoint := &jsonSchema.DecisionPoint{
		Name:     "termination",
		Branches: []jsonSchema.ConditionalBranch{},
	}

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("testField", fieldDef, nil)
	result := domain.NewTaskResult(task.ID(), task.Key(), "test", domain.NewResultMetadata())

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	shouldStop, err := processor.shouldTerminate(ctx, terminationPoint, result, execContext, task)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if shouldStop {
		t.Error("Expected shouldStop to be false when no branches")
	}
}

func TestRecursiveLoopProcessor_ShouldTerminate_NoDecisionProcessor(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil) // nil decision processor

	ctx := context.Background()

	terminationPoint := &jsonSchema.DecisionPoint{
		Name: "termination",
		Branches: []jsonSchema.ConditionalBranch{
			{Name: "stop", Conditions: []jsonSchema.Condition{{Field: "quality", Operator: jsonSchema.OpGreaterThan, Value: 90}}},
		},
	}

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("testField", fieldDef, nil)
	result := domain.NewTaskResult(task.ID(), task.Key(), "test", domain.NewResultMetadata())

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	shouldStop, err := processor.shouldTerminate(ctx, terminationPoint, result, execContext, task)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if shouldStop {
		t.Error("Expected shouldStop to be false when no decision processor")
	}
}

func TestRecursiveLoopProcessor_ShouldTerminate_ContextCancelled(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	decisionProcessor := NewDecisionProcessor(gen)
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, decisionProcessor)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	terminationPoint := &jsonSchema.DecisionPoint{
		Name: "termination",
		Branches: []jsonSchema.ConditionalBranch{
			{Name: "stop"},
		},
	}

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("testField", fieldDef, nil)
	result := domain.NewTaskResult(task.ID(), task.Key(), "test", domain.NewResultMetadata())

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	_, err := processor.shouldTerminate(ctx, terminationPoint, result, execContext, task)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestRecursiveLoopProcessor_Process_WithTermination(t *testing.T) {
	callCount := 0
	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			callCount++
			return domain.NewTaskResult(task.ID(), task.Key(), fmt.Sprintf("result-%d", callCount), domain.NewResultMetadata()), nil
		},
	}

	// Mock generator that evaluates termination condition
	conditionCallCount := 0
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			conditionCallCount++
			// On third iteration, return quality > 90 to trigger termination
			quality := 80
			if conditionCallCount >= 3 {
				quality = 95
			}
			return domain.NewGenerationResult(map[string]interface{}{
				"quality": quality,
			}, nil), nil
		},
	}
	decisionProcessor := NewDecisionProcessor(gen)
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, decisionProcessor)

	ctx := context.Background()

	terminationPoint := &jsonSchema.DecisionPoint{
		Name: "termination",
		Branches: []jsonSchema.ConditionalBranch{
			{
				Name: "high_quality",
				Conditions: []jsonSchema.Condition{
					{Field: "quality", Operator: jsonSchema.OpGreaterThan, Value: 90},
				},
			},
		},
	}

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations:    5,
		Selection:        jsonSchema.SelectLatest,
		TerminationPoint: terminationPoint,
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	result, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should terminate at iteration 3 when quality > 90
	if callCount != 3 {
		t.Errorf("Expected 3 iterations before termination, got %d", callCount)
	}

	if result.Value() != "result-3" {
		t.Errorf("Expected 'result-3', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_EvaluateScores(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(map[string]interface{}{
				"quality":   85.0,
				"relevance": 90.0,
			}, nil), nil
		},
	}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality score",
			},
			"relevance": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Relevance score",
			},
		},
		AggregationMethod: jsonSchema.AggregateWeightedAverage,
	}

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())
	req := domain.NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})
	execContext := domain.NewExecutionContext(req)

	scores, err := processor.evaluateScores(result, criteria, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if scores["quality"] != 85.0 {
		t.Errorf("Expected quality score 85.0, got %v", scores["quality"])
	}
	if scores["relevance"] != 90.0 {
		t.Errorf("Expected relevance score 90.0, got %v", scores["relevance"])
	}

	// Check aggregate is calculated
	if _, exists := scores["_aggregate"]; !exists {
		t.Error("Expected _aggregate score to be calculated")
	}
}

func TestRecursiveLoopProcessor_EvaluateScores_CustomModel(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	var capturedSchema *jsonSchema.Definition
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			capturedSchema = req.Schema()
			return domain.NewGenerationResult(map[string]interface{}{
				"score": 85.0,
			}, nil), nil
		},
	}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"score": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Score",
			},
		},
		EvaluationModel: "gpt-4",
	}

	result := domain.NewTaskResult("test-id", "testField", "test", domain.NewResultMetadata())
	req := domain.NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})
	execContext := domain.NewExecutionContext(req)

	_, err := processor.evaluateScores(result, criteria, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if capturedSchema == nil {
		t.Fatal("Expected schema to be captured")
	}
	if capturedSchema.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %v", capturedSchema.Model)
	}
}

func TestRecursiveLoopProcessor_CalculateAggregate_Average(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	scores := map[string]float64{
		"quality":   80.0,
		"relevance": 90.0,
		"clarity":   70.0,
	}

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality":   {Type: jsonSchema.ScoreNumeric},
			"relevance": {Type: jsonSchema.ScoreNumeric},
			"clarity":   {Type: jsonSchema.ScoreNumeric},
		},
		AggregationMethod: jsonSchema.AggregateWeightedAverage,
	}

	aggregate := processor.calculateAggregate(scores, criteria)
	expected := (80.0 + 90.0 + 70.0) / 3.0

	if aggregate != expected {
		t.Errorf("Expected average %v, got %v", expected, aggregate)
	}
}

func TestRecursiveLoopProcessor_CalculateAggregate_WeightedAverage(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	scores := map[string]float64{
		"quality":   80.0,
		"relevance": 90.0,
	}

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality":   {Type: jsonSchema.ScoreNumeric, Weight: 0.7},
			"relevance": {Type: jsonSchema.ScoreNumeric, Weight: 0.3},
		},
		AggregationMethod: jsonSchema.AggregateWeightedAverage,
	}

	aggregate := processor.calculateAggregate(scores, criteria)
	expected := (80.0 * 0.7) + (90.0 * 0.3)

	if aggregate != expected {
		t.Errorf("Expected weighted average %v, got %v", expected, aggregate)
	}
}

func TestRecursiveLoopProcessor_CalculateAggregate_Minimum(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	scores := map[string]float64{
		"quality":   80.0,
		"relevance": 90.0,
		"clarity":   70.0,
	}

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality":   {Type: jsonSchema.ScoreNumeric},
			"relevance": {Type: jsonSchema.ScoreNumeric},
			"clarity":   {Type: jsonSchema.ScoreNumeric},
		},
		AggregationMethod: jsonSchema.AggregateMinimum,
	}

	aggregate := processor.calculateAggregate(scores, criteria)
	if aggregate != 70.0 {
		t.Errorf("Expected minimum 70.0, got %v", aggregate)
	}
}

func TestRecursiveLoopProcessor_CalculateAggregate_Maximum(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	scores := map[string]float64{
		"quality":   80.0,
		"relevance": 90.0,
		"clarity":   70.0,
	}

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality":   {Type: jsonSchema.ScoreNumeric},
			"relevance": {Type: jsonSchema.ScoreNumeric},
			"clarity":   {Type: jsonSchema.ScoreNumeric},
		},
		AggregationMethod: jsonSchema.AggregateMaximum,
	}

	aggregate := processor.calculateAggregate(scores, criteria)
	if aggregate != 90.0 {
		t.Errorf("Expected maximum 90.0, got %v", aggregate)
	}
}

func TestRecursiveLoopProcessor_SelectResult_HighestScore(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{
			result:    domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()),
			scores:    map[string]float64{"_aggregate": 70.0},
			iteration: 1,
		},
		{
			result:    domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()),
			scores:    map[string]float64{"_aggregate": 90.0},
			iteration: 2,
		},
		{
			result:    domain.NewTaskResult("3", "test", "result-3", domain.NewResultMetadata()),
			scores:    map[string]float64{"_aggregate": 80.0},
			iteration: 3,
		},
	}

	result, err := processor.selectResult(results, jsonSchema.SelectHighestScore, "test")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Value() != "result-2" {
		t.Errorf("Expected highest scoring 'result-2', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_SelectResult_LowestScore(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{
			result:    domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()),
			scores:    map[string]float64{"_aggregate": 70.0},
			iteration: 1,
		},
		{
			result:    domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()),
			scores:    map[string]float64{"_aggregate": 90.0},
			iteration: 2,
		},
		{
			result:    domain.NewTaskResult("3", "test", "result-3", domain.NewResultMetadata()),
			scores:    map[string]float64{"_aggregate": 80.0},
			iteration: 3,
		},
	}

	result, err := processor.selectResult(results, jsonSchema.SelectLowestScore, "test")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Value() != "result-1" {
		t.Errorf("Expected lowest scoring 'result-1', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_SelectResult_Latest(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{result: domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()), iteration: 1},
		{result: domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()), iteration: 2},
		{result: domain.NewTaskResult("3", "test", "result-3", domain.NewResultMetadata()), iteration: 3},
	}

	result, err := processor.selectResult(results, jsonSchema.SelectLatest, "test")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Value() != "result-3" {
		t.Errorf("Expected latest 'result-3', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_SelectResult_First(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{result: domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()), iteration: 1},
		{result: domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()), iteration: 2},
		{result: domain.NewTaskResult("3", "test", "result-3", domain.NewResultMetadata()), iteration: 3},
	}

	result, err := processor.selectResult(results, jsonSchema.SelectFirst, "test")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Value() != "result-1" {
		t.Errorf("Expected first 'result-1', got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_SelectResult_All(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{result: domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()), iteration: 1},
		{result: domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()), iteration: 2},
		{result: domain.NewTaskResult("3", "test", "result-3", domain.NewResultMetadata()), iteration: 3},
	}

	result, err := processor.selectResult(results, jsonSchema.SelectAll, "test")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	allValues, ok := result.Value().([]interface{})
	if !ok {
		t.Fatalf("Expected array of results, got %T", result.Value())
	}

	if len(allValues) != 3 {
		t.Errorf("Expected 3 results in array, got %d", len(allValues))
	}

	if allValues[0] != "result-1" || allValues[1] != "result-2" || allValues[2] != "result-3" {
		t.Error("Expected all results to be included in order")
	}
}

func TestRecursiveLoopProcessor_SelectResult_NoResults(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{}

	_, err := processor.selectResult(results, jsonSchema.SelectLatest, "test")
	if err == nil {
		t.Error("Expected error for empty results")
	}
}

func TestRecursiveLoopProcessor_SelectByScore_NoScores(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{result: domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()), iteration: 1},
		{result: domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()), iteration: 2},
	}

	result := processor.selectByScore(results, true)

	// Should return first result when no scores available
	if result.Value() != "result-1" {
		t.Errorf("Expected first result when no scores, got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_SelectByScore_CalculatedAggregate(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{
			result:    domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()),
			scores:    map[string]float64{"quality": 80.0, "relevance": 90.0}, // avg = 85
			iteration: 1,
		},
		{
			result:    domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()),
			scores:    map[string]float64{"quality": 95.0, "relevance": 90.0}, // avg = 92.5
			iteration: 2,
		},
	}

	result := processor.selectByScore(results, true)

	// Should calculate average and select highest
	if result.Value() != "result-2" {
		t.Errorf("Expected result-2 with higher calculated average, got %v", result.Value())
	}
}

func TestRecursiveLoopProcessor_EnhanceContextWithFeedback(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	loop := &jsonSchema.RecursiveLoop{
		FeedbackPrompt: "Improve based on previous attempt:",
	}

	previousIter := iterationResult{
		result:    domain.NewTaskResult("1", "test", "previous content", domain.NewResultMetadata()),
		scores:    map[string]float64{"quality": 75.0, "relevance": 80.0},
		iteration: 1,
	}

	req := domain.NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})
	execContext := domain.NewExecutionContext(req)

	initialPromptCount := len(execContext.PromptContext().Prompts)

	processor.enhanceContextWithFeedback(loop, previousIter, execContext)

	// Context should be enhanced with feedback
	finalPromptCount := len(execContext.PromptContext().Prompts)
	if finalPromptCount <= initialPromptCount {
		t.Error("Expected context to be enhanced with feedback")
	}
}

func TestRecursiveLoopProcessor_EnhanceContextWithFeedback_NoFeedbackPrompt(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	loop := &jsonSchema.RecursiveLoop{
		FeedbackPrompt: "", // No feedback prompt
	}

	previousIter := iterationResult{
		result:    domain.NewTaskResult("1", "test", "content", domain.NewResultMetadata()),
		iteration: 1,
	}

	req := domain.NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})
	execContext := domain.NewExecutionContext(req)

	initialPrompts := execContext.PromptContext().Prompts

	processor.enhanceContextWithFeedback(loop, previousIter, execContext)

	// Context should not be modified
	finalPrompts := execContext.PromptContext().Prompts
	if len(finalPrompts) != len(initialPrompts) {
		t.Error("Expected context to remain unchanged without feedback prompt")
	}
}

func TestToFloat64_Conversions(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected float64
		ok       bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"float32", float32(3.14), 3.14, true},
		{"int", int(42), 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"string", "not a number", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
		{"struct", struct{}{}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.value)
			if ok != tt.ok {
				t.Errorf("Expected ok=%v, got %v", tt.ok, ok)
			}
			if ok {
				// For float32, allow small precision differences
				diff := result - tt.expected
				if diff < 0 {
					diff = -diff
				}
				if diff > 0.0001 {
					t.Errorf("Expected %v, got %v (diff: %v)", tt.expected, result, diff)
				}
			}
		})
	}
}

func TestRecursiveLoopProcessor_Process_WithFeedback(t *testing.T) {
	callCount := 0
	var capturedContext *domain.ExecutionContext

	baseProcessor := &mockTypeProcessor{
		processFunc: func(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
			callCount++
			capturedContext = execContext
			return domain.NewTaskResult(task.ID(), task.Key(), fmt.Sprintf("result-%d", callCount), domain.NewResultMetadata()), nil
		},
	}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	ctx := context.Background()

	recursiveLoop := &jsonSchema.RecursiveLoop{
		MaxIterations:  3,
		Selection:      jsonSchema.SelectLatest,
		FeedbackPrompt: "Improve based on the previous attempt:",
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: recursiveLoop,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	_, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The context should have feedback added after first iteration
	if capturedContext == nil {
		t.Fatal("Expected context to be captured")
	}

	// Check that feedback was added to prompt context
	promptsCount := len(capturedContext.PromptContext().Prompts)
	if promptsCount == 0 {
		t.Error("Expected feedback to be added to prompt context")
	}
}

func TestRecursiveLoopProcessor_SelectResult_DefaultStrategy(t *testing.T) {
	baseProcessor := &mockTypeProcessor{}
	gen := &mockGenerator{}
	processor := NewRecursiveLoopProcessor(baseProcessor, gen, nil)

	results := []iterationResult{
		{result: domain.NewTaskResult("1", "test", "result-1", domain.NewResultMetadata()), iteration: 1},
		{result: domain.NewTaskResult("2", "test", "result-2", domain.NewResultMetadata()), iteration: 2},
		{result: domain.NewTaskResult("3", "test", "result-3", domain.NewResultMetadata()), iteration: 3},
	}

	// Use an unknown strategy - should default to latest
	result, err := processor.selectResult(results, "unknown_strategy", "test")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Value() != "result-3" {
		t.Errorf("Expected default to latest 'result-3', got %v", result.Value())
	}
}
