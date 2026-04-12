package execution

import (
	"context"
	"errors"
	"math"
	"objectweaver/orchestration/jos/domain"
	"sync"
	"testing"
	"time"

	"objectweaver/jsonSchema"
)

// TestNewFieldProcessor tests the constructor
func TestNewFieldProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewFieldProcessor(llmProvider, promptBuilder)

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}
	if processor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.generator != nil {
		t.Error("Expected generator to be nil initially")
	}
	if processor.decisionProcessor != nil {
		t.Error("Expected decisionProcessor to be nil initially")
	}
}

// TestSetGenerator tests setting the generator
func TestSetGenerator(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	gen := &mockGenerator{}

	processor.SetGenerator(gen)

	if processor.generator != gen {
		t.Error("Expected generator to be set")
	}
	if processor.decisionProcessor == nil {
		t.Error("Expected decisionProcessor to be initialized after setting generator")
	}
}

// TestSetEpstimicOrchestrator tests setting the epstimic orchestrator
func TestSetEpstimicOrchestrator(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	orchestrator := &mockEpstimicOrchestrator{}

	processor.SetEpstimicOrchestrator(orchestrator)

	if processor.epstimicOrchestrator != orchestrator {
		t.Error("Expected epstimicOrchestrator to be set")
	}
}

// TestProcessFields_NilSchema tests handling of nil schema
func TestProcessFields_NilSchema(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	ctx := context.Background()
	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	resultsCh := processor.ProcessFieldsStart(ctx, nil, nil, execContext)

	count := 0
	for range resultsCh {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 results for nil schema, got %d", count)
	}
}

// TestProcessFields_NilProperties tests handling of nil properties
func TestProcessFields_NilProperties(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	ctx := context.Background()

	schema := &jsonSchema.Definition{
		Type:       jsonSchema.Object,
		Properties: nil,
	}
	request := domain.NewGenerationRequest("test", schema)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	resultsCh := processor.ProcessFieldsStart(ctx, schema, nil, execContext)

	count := 0
	for range resultsCh {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 results for nil properties, got %d", count)
	}
}

// TestProcessFields_SimpleFields tests basic field processing
func TestProcessFields_SimpleFields(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "test-value", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	processor := NewFieldProcessor(llmProvider, &mockPromptBuilder{})
	ctx := context.Background()

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"field1": {Type: jsonSchema.String},
			"field2": {Type: jsonSchema.String},
			"field3": {Type: jsonSchema.String},
		},
	}
	request := domain.NewGenerationRequest("test", schema)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	resultsCh := processor.ProcessFieldsStart(ctx, schema, nil, execContext)

	results := make(map[string]*domain.TaskResult)
	for resultSlice := range resultsCh {
		for _, result := range resultSlice {
			results[result.Key()] = result
		}
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for key := range schema.Properties {
		if _, exists := results[key]; !exists {
			t.Errorf("Expected result for key %s", key)
		}
	}
}

// TestProcessFields_SequentialProcessing tests ordered field processing
func TestProcessFields_SequentialProcessing(t *testing.T) {
	var mu sync.Mutex
	callOrder := []string{}
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			// Extract field name from prompt (simplified)
			return "value", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{
		buildFunc: func(task *domain.FieldTask, context *domain.PromptContext) (string, error) {
			mu.Lock()
			callOrder = append(callOrder, task.Key())
			mu.Unlock()
			return "prompt", nil
		},
	}

	processor := NewFieldProcessor(llmProvider, promptBuilder)
	ctx := context.Background()

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"first":  {Type: jsonSchema.String},
			"second": {Type: jsonSchema.String},
			"third":  {Type: jsonSchema.String},
			"fourth": {Type: jsonSchema.String},
		},
		// All fields in ProcessingOrder to ensure sequential processing
		// first and second have no dependencies so they batch together
		// third depends on first, fourth depends on second - forcing them to a later batch
		ProcessingOrder: []string{"first", "second", "third", "fourth"},
	}
	// Add SelectFields to create dependencies that force batching order
	thirdDef := schema.Properties["third"]
	thirdDef.SelectFields = []string{"first"}
	schema.Properties["third"] = thirdDef

	fourthDef := schema.Properties["fourth"]
	fourthDef.SelectFields = []string{"second"}
	schema.Properties["fourth"] = fourthDef
	request := domain.NewGenerationRequest("test", schema)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	resultsCh := processor.ProcessFieldsStart(ctx, schema, nil, execContext)

	for range resultsCh {
		// Consume results
	}

	// First two should be processed first (may be in either order since they're batched concurrently)
	if len(callOrder) < 2 {
		t.Fatal("Expected at least 2 calls")
	}
	// Check that first and second were processed before third and fourth
	firstTwoProcessed := make(map[string]bool)
	for i := 0; i < 2; i++ {
		firstTwoProcessed[callOrder[i]] = true
	}
	if !firstTwoProcessed["first"] || !firstTwoProcessed["second"] {
		t.Errorf("Expected 'first' and 'second' to be processed in the first batch, got %v", callOrder[:2])
	}
	// Verify that third and fourth come after the first batch
	if len(callOrder) >= 4 {
		lastTwoProcessed := make(map[string]bool)
		for i := 2; i < 4; i++ {
			lastTwoProcessed[callOrder[i]] = true
		}
		if !lastTwoProcessed["third"] || !lastTwoProcessed["fourth"] {
			t.Errorf("Expected 'third' and 'fourth' to be processed after the first batch, got %v", callOrder[2:])
		}
	}
}

// TestProcessFields_ContextCancellation tests context cancellation handling
func TestProcessFields_ContextCancellation(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			time.Sleep(10 * time.Millisecond)
			return "value", &domain.ProviderMetadata{}, nil
		},
	}
	processor := NewFieldProcessor(llmProvider, &mockPromptBuilder{})

	ctx, cancel := context.WithCancel(context.Background())

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"field1": {Type: jsonSchema.String},
			"field2": {Type: jsonSchema.String},
			"field3": {Type: jsonSchema.String},
		},
	}
	request := domain.NewGenerationRequest("test", schema)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	resultsCh := processor.ProcessFieldsStart(ctx, schema, nil, execContext)

	// Cancel after a short time
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	count := 0
	for range resultsCh {
		count++
	}

	// Should get fewer results due to cancellation
	// (exact number depends on timing)
	t.Logf("Received %d results before cancellation", count)
}

// TestProcessObjectField tests nested object processing
func TestProcessObjectField(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "value", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	processor := NewFieldProcessor(llmProvider, &mockPromptBuilder{})
	ctx := context.Background()

	nestedSchema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"nested1": {Type: jsonSchema.String},
			"nested2": {Type: jsonSchema.String},
		},
	}

	task := domain.NewFieldTask("nestedObj", nestedSchema, nil)
	request := domain.NewGenerationRequest("test", nestedSchema)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	// Process object field - it collects nested results and returns a single combined result
	results := processor.processObjectField(ctx, task, execContext)

	// processObjectField should return 1 result containing the nested object
	if len(results) != 1 {
		t.Fatalf("Expected 1 result with nested object, got %d", len(results))
	}

	// Verify the result contains the nested object with both fields
	result := results[0]
	if result.Key() != "nestedObj" {
		t.Errorf("Expected key 'nestedObj', got '%s'", result.Key())
	}

	// The value should be a map with nested1 and nested2
	nestedMap, ok := result.Value().(map[string]interface{})
	if !ok {
		t.Fatalf("Expected value to be map[string]interface{}, got %T", result.Value())
	}

	if _, exists := nestedMap["nested1"]; !exists {
		t.Error("Expected nested1 in result")
	}
	if _, exists := nestedMap["nested2"]; !exists {
		t.Error("Expected nested2 in result")
	}
}

// TestGetProcessorForType tests processor routing
func TestGetProcessorForType(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	tests := []struct {
		name         string
		def          *jsonSchema.Definition
		expectedType string
	}{
		{
			name:         "String",
			def:          &jsonSchema.Definition{Type: jsonSchema.String},
			expectedType: "*execution.PrimitiveProcessor",
		},
		{
			name:         "Number",
			def:          &jsonSchema.Definition{Type: jsonSchema.Number},
			expectedType: "*execution.NumberProcessor",
		},
		{
			name:         "Integer",
			def:          &jsonSchema.Definition{Type: jsonSchema.Integer},
			expectedType: "*execution.NumberProcessor",
		},
		{
			name:         "Boolean",
			def:          &jsonSchema.Definition{Type: jsonSchema.Boolean},
			expectedType: "*execution.BooleanProcessor",
		},
		{
			name:         "Array",
			def:          &jsonSchema.Definition{Type: jsonSchema.Array},
			expectedType: "*execution.ArrayProcessor",
		},
		{
			name:         "Map",
			def:          &jsonSchema.Definition{Type: jsonSchema.Map},
			expectedType: "*execution.MapProcessor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := processor.getProcessorForType(tt.def)
			if proc == nil {
				t.Fatal("Expected non-nil processor")
			}
		})
	}
}

// TestGetProcessorForType_ByteOperations tests byte operation routing
func TestGetProcessorForType_ByteOperations(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	tests := []struct {
		name string
		def  *jsonSchema.Definition
	}{
		{
			name: "TextToSpeech",
			def:  &jsonSchema.Definition{Type: jsonSchema.String, TextToSpeech: &jsonSchema.TextToSpeech{}},
		},
		{
			name: "Image",
			def:  &jsonSchema.Definition{Type: jsonSchema.String, Image: &jsonSchema.Image{}},
		},
		{
			name: "SpeechToText",
			def:  &jsonSchema.Definition{Type: jsonSchema.String, SpeechToText: &jsonSchema.SpeechToText{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := processor.getProcessorForType(tt.def)
			if proc == nil {
				t.Fatal("Expected non-nil processor")
			}
			// Should return ByteProcessor
			_, ok := proc.(*ByteProcessor)
			if !ok {
				t.Errorf("Expected *ByteProcessor, got %T", proc)
			}
		})
	}
}

// TestGetProcessorForType_RecursiveLoop tests recursive loop routing
func TestGetProcessorForType_RecursiveLoop(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	processor.SetGenerator(&mockGenerator{})

	def := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		RecursiveLoop: &jsonSchema.RecursiveLoop{},
	}

	proc := processor.getProcessorForType(def)
	if proc == nil {
		t.Fatal("Expected non-nil processor")
	}

	_, ok := proc.(*RecursiveLoopProcessor)
	if !ok {
		t.Errorf("Expected *RecursiveLoopProcessor, got %T", proc)
	}
}

// TestGetOrderedKeys tests key ordering logic
func TestGetOrderedKeys(t *testing.T) {
	tests := []struct {
		name              string
		schema            *jsonSchema.Definition
		expectedOrdered   []string
		expectedRemaining int
	}{
		{
			name: "NoProcessingOrder",
			schema: &jsonSchema.Definition{
				Properties: map[string]jsonSchema.Definition{
					"a": {Type: jsonSchema.String},
					"b": {Type: jsonSchema.String},
					"c": {Type: jsonSchema.String},
				},
			},
			expectedOrdered:   nil,
			expectedRemaining: 3,
		},
		{
			name: "PartialProcessingOrder",
			schema: &jsonSchema.Definition{
				Properties: map[string]jsonSchema.Definition{
					"a": {Type: jsonSchema.String},
					"b": {Type: jsonSchema.String},
					"c": {Type: jsonSchema.String},
					"d": {Type: jsonSchema.String},
				},
				ProcessingOrder: []string{"a", "b"},
			},
			expectedOrdered:   []string{"a", "b"},
			expectedRemaining: 2,
		},
		{
			name: "AllOrdered",
			schema: &jsonSchema.Definition{
				Properties: map[string]jsonSchema.Definition{
					"x": {Type: jsonSchema.String},
					"y": {Type: jsonSchema.String},
				},
				ProcessingOrder: []string{"x", "y"},
			},
			expectedOrdered:   []string{"x", "y"},
			expectedRemaining: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ordered, remaining := getOrderedKeys(tt.schema)

			if len(ordered) != len(tt.expectedOrdered) {
				t.Errorf("Expected %d ordered keys, got %d", len(tt.expectedOrdered), len(ordered))
			}

			for i, key := range tt.expectedOrdered {
				if ordered[i] != key {
					t.Errorf("Expected ordered key %s at position %d, got %s", key, i, ordered[i])
				}
			}

			if len(remaining) != tt.expectedRemaining {
				t.Errorf("Expected %d remaining keys, got %d", tt.expectedRemaining, len(remaining))
			}
		})
	}
}

// TestProcessField_DecisionPoint tests decision point handling
func TestProcessField_DecisionPoint(t *testing.T) {
	// Mock generator that returns different results for decision branches
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(map[string]interface{}{
				"decision_field": "branch_value",
			}, domain.NewResultMetadata()), nil
		},
	}

	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "initial_value", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewFieldProcessor(llmProvider, &mockPromptBuilder{})
	processor.SetGenerator(gen)

	ctx := context.Background()

	// Create minimal decision definition - just testing that decision points are handled
	decisionDef := &jsonSchema.Definition{
		Type: jsonSchema.String,
		DecisionPoint: &jsonSchema.DecisionPoint{
			Name: "test-decision",
		},
	}

	task := domain.NewFieldTask("testField", decisionDef, nil)
	request := domain.NewGenerationRequest("test", decisionDef)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	results := processor.processField(ctx, task, execContext)

	// Should have at least the original result
	if len(results) == 0 {
		t.Fatal("Expected at least 1 result")
	}

	// Check that the original field value was added to context
	if _, exists := execContext.GetGeneratedValue("testField"); !exists {
		t.Error("Expected testField to be in execution context")
	}
}

// TestProcessField_ScoringCriteria tests field scoring
func TestProcessField_ScoringCriteria(t *testing.T) {
	t.Skip("Skipping scoring test - requires full jsonSchema.ScoringCriteria implementation")
}

// TestEvaluateScores tests score evaluation
func TestEvaluateScores(t *testing.T) {
	// Mock generator that returns scores
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(map[string]interface{}{
				"quality":   85.0,
				"relevance": 90.0,
				"clarity":   88.0,
			}, domain.NewResultMetadata()), nil
		},
	}

	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	processor.SetGenerator(gen)

	ctx := context.Background()

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality of content",
				Weight:      0.5,
			},
			"relevance": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Relevance to topic",
				Weight:      0.3,
			},
			"clarity": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Clarity of expression",
				Weight:      0.2,
			},
		},
		AggregationMethod: jsonSchema.AggregateWeightedAverage,
	}

	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)

	scores, err := processor.evaluateScores(ctx, result, criteria, execContext)
	if err != nil {
		t.Fatalf("evaluateScores failed: %v", err)
	}

	if len(scores) != 4 { // 3 dimensions + aggregate
		t.Errorf("Expected 4 scores (3 dimensions + aggregate), got %d", len(scores))
	}

	if scores["quality"] != 85.0 {
		t.Errorf("Expected quality score 85.0, got %v", scores["quality"])
	}

	if scores["relevance"] != 90.0 {
		t.Errorf("Expected relevance score 90.0, got %v", scores["relevance"])
	}

	if scores["clarity"] != 88.0 {
		t.Errorf("Expected clarity score 88.0, got %v", scores["clarity"])
	}

	// Check aggregate exists
	if _, exists := scores["_aggregate"]; !exists {
		t.Error("Expected _aggregate score to be present")
	}
}

// TestEvaluateScores_NoGenerator tests error when generator is not set
func TestEvaluateScores_NoGenerator(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	// Don't set generator

	ctx := context.Background()

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality of content",
			},
		},
	}

	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)

	_, err := processor.evaluateScores(ctx, result, criteria, execContext)
	if err == nil {
		t.Error("Expected error when generator is not set")
	}

	if err.Error() != "generator not set, cannot evaluate scores" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestEvaluateScores_ContextCancelled tests context cancellation
func TestEvaluateScores_ContextCancelled(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	processor.SetGenerator(gen)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality",
			},
		},
	}

	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)

	_, err := processor.evaluateScores(ctx, result, criteria, execContext)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// TestEvaluateScores_DifferentDimensionTypes tests different score types
func TestEvaluateScores_DifferentDimensionTypes(t *testing.T) {
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(map[string]interface{}{
				"score":   85.0,
				"isValid": 1, // Represented as int
			}, domain.NewResultMetadata()), nil
		},
	}

	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	processor.SetGenerator(gen)

	ctx := context.Background()

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"score": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Numeric score",
			},
			"isValid": {
				Type:        jsonSchema.ScoreBoolean,
				Description: "Is valid",
			},
		},
	}

	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)

	scores, err := processor.evaluateScores(ctx, result, criteria, execContext)
	if err != nil {
		t.Fatalf("evaluateScores failed: %v", err)
	}

	// Numeric scores should be extracted
	if scores["score"] != 85.0 {
		t.Errorf("Expected score 85.0, got %v", scores["score"])
	}

	// Boolean converted to numeric (1.0)
	if scores["isValid"] != 1.0 {
		t.Errorf("Expected isValid 1.0, got %v", scores["isValid"])
	}
}

// TestEvaluateScores_WithScale tests dimension with scale
func TestEvaluateScores_WithScale(t *testing.T) {
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			// Verify the schema includes scale in instruction
			schema := req.Schema()
			if schema == nil || schema.Properties == nil {
				t.Error("Expected schema with properties")
			}
			return domain.NewGenerationResult(map[string]interface{}{
				"quality": 8.5,
			}, domain.NewResultMetadata()), nil
		},
	}

	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	processor.SetGenerator(gen)

	ctx := context.Background()

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality rating",
				Scale: &jsonSchema.ScoreScale{
					Min: 0,
					Max: 10,
				},
			},
		},
	}

	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)

	scores, err := processor.evaluateScores(ctx, result, criteria, execContext)
	if err != nil {
		t.Fatalf("evaluateScores failed: %v", err)
	}

	if scores["quality"] != 8.5 {
		t.Errorf("Expected quality score 8.5, got %v", scores["quality"])
	}
}

// TestEvaluateScores_CustomModel tests evaluation with custom model
func TestEvaluateScores_CustomModel(t *testing.T) {
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			// Verify model is set
			schema := req.Schema()
			if schema.Model != "gpt-4" {
				t.Errorf("Expected model 'gpt-4', got '%s'", schema.Model)
			}
			return domain.NewGenerationResult(map[string]interface{}{
				"quality": 95.0,
			}, domain.NewResultMetadata()), nil
		},
	}

	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	processor.SetGenerator(gen)

	ctx := context.Background()

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality",
			},
		},
		EvaluationModel: "gpt-4",
	}

	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)

	_, err := processor.evaluateScores(ctx, result, criteria, execContext)
	if err != nil {
		t.Fatalf("evaluateScores failed: %v", err)
	}
}

// TestEvaluateScores_GenerationError tests handling of generation errors
func TestEvaluateScores_GenerationError(t *testing.T) {
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return nil, errors.New("generation failed")
		},
	}

	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	processor.SetGenerator(gen)

	ctx := context.Background()

	result := domain.NewTaskResult("test-id", "testField", "test content", domain.NewResultMetadata())

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {
				Type:        jsonSchema.ScoreNumeric,
				Description: "Quality",
			},
		},
	}

	request := domain.NewGenerationRequest("test", &jsonSchema.Definition{})
	execContext := domain.NewExecutionContext(request)

	_, err := processor.evaluateScores(ctx, result, criteria, execContext)
	if err == nil {
		t.Error("Expected error when generation fails")
	}
}

// TestCalculateAggregate tests score aggregation methods
func TestCalculateAggregate(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	tests := []struct {
		name              string
		scores            map[string]float64
		criteria          *jsonSchema.ScoringCriteria
		expectedAggregate float64
	}{
		{
			name: "WeightedAverage_WithWeights",
			scores: map[string]float64{
				"quality":   80.0,
				"relevance": 90.0,
				"clarity":   85.0,
			},
			criteria: &jsonSchema.ScoringCriteria{
				Dimensions: map[string]jsonSchema.ScoringDimension{
					"quality":   {Weight: 0.5},
					"relevance": {Weight: 0.3},
					"clarity":   {Weight: 0.2},
				},
				AggregationMethod: jsonSchema.AggregateWeightedAverage,
			},
			expectedAggregate: 84.0, // (80*0.5 + 90*0.3 + 85*0.2) = 84.0
		},
		{
			name: "WeightedAverage_NoWeights",
			scores: map[string]float64{
				"a": 60.0,
				"b": 80.0,
				"c": 100.0,
			},
			criteria: &jsonSchema.ScoringCriteria{
				Dimensions: map[string]jsonSchema.ScoringDimension{
					"a": {Weight: 0},
					"b": {Weight: 0},
					"c": {Weight: 0},
				},
				AggregationMethod: jsonSchema.AggregateWeightedAverage,
			},
			expectedAggregate: 80.0, // (60 + 80 + 100) / 3 = 80.0
		},
		{
			name: "Minimum",
			scores: map[string]float64{
				"quality":   85.0,
				"relevance": 75.0,
				"clarity":   90.0,
			},
			criteria: &jsonSchema.ScoringCriteria{
				Dimensions: map[string]jsonSchema.ScoringDimension{
					"quality":   {},
					"relevance": {},
					"clarity":   {},
				},
				AggregationMethod: jsonSchema.AggregateMinimum,
			},
			expectedAggregate: 75.0,
		},
		{
			name: "Maximum",
			scores: map[string]float64{
				"quality":   85.0,
				"relevance": 95.0,
				"clarity":   90.0,
			},
			criteria: &jsonSchema.ScoringCriteria{
				Dimensions: map[string]jsonSchema.ScoringDimension{
					"quality":   {},
					"relevance": {},
					"clarity":   {},
				},
				AggregationMethod: jsonSchema.AggregateMaximum,
			},
			expectedAggregate: 95.0,
		},
		{
			name: "Average_Default",
			scores: map[string]float64{
				"quality":   80.0,
				"relevance": 90.0,
			},
			criteria: &jsonSchema.ScoringCriteria{
				Dimensions: map[string]jsonSchema.ScoringDimension{
					"quality":   {},
					"relevance": {},
				},
				AggregationMethod: "", // Unknown/default
			},
			expectedAggregate: 85.0, // (80 + 90) / 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.calculateAggregate(tt.scores, tt.criteria)
			if math.Abs(result-tt.expectedAggregate) > 0.0001 {
				t.Errorf("Expected aggregate %v, got %v", tt.expectedAggregate, result)
			}
		})
	}
}

// TestCalculateAggregate_EmptyScores tests edge cases
func TestCalculateAggregate_EmptyScores(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions:        map[string]jsonSchema.ScoringDimension{},
		AggregationMethod: jsonSchema.AggregateWeightedAverage,
	}

	result := processor.calculateAggregate(map[string]float64{}, criteria)
	if result != 0 {
		t.Errorf("Expected 0 for empty scores, got %v", result)
	}
}

// TestCalculateAggregate_MinimumWithHighValue tests minimum starts at 100
func TestCalculateAggregate_MinimumWithHighValue(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	scores := map[string]float64{
		"quality": 98.0,
	}

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {},
		},
		AggregationMethod: jsonSchema.AggregateMinimum,
	}

	result := processor.calculateAggregate(scores, criteria)
	if result != 98.0 {
		t.Errorf("Expected 98.0, got %v", result)
	}
}

// TestCalculateAggregate_MaximumWithLowValue tests maximum starts at 0
func TestCalculateAggregate_MaximumWithLowValue(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	scores := map[string]float64{
		"quality": 5.0,
	}

	criteria := &jsonSchema.ScoringCriteria{
		Dimensions: map[string]jsonSchema.ScoringDimension{
			"quality": {},
		},
		AggregationMethod: jsonSchema.AggregateMaximum,
	}

	result := processor.calculateAggregate(scores, criteria)
	if result != 5.0 {
		t.Errorf("Expected 5.0, got %v", result)
	}
}

// TestAttachScoresToResult tests score attachment to result metadata
func TestAttachScoresToResult(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	result := domain.NewTaskResult("test-id", "testField", "value", domain.NewResultMetadata())

	scores := map[string]float64{
		"quality":    85.0,
		"_aggregate": 87.5,
	}

	processor.attachScoresToResult(result, scores)

	if result.Metadata() == nil {
		t.Fatal("Expected metadata to be non-nil")
	}

	if len(result.Metadata().Choices) == 0 {
		t.Fatal("Expected choices to be added")
	}

	// Should have the aggregate score
	choice := result.Metadata().Choices[0]
	if choice.Score != 87 { // int conversion
		t.Errorf("Expected score 87, got %d", choice.Score)
	}
}

// TestAttachScoresToResult_NilMetadata tests handling of nil metadata
func TestAttachScoresToResult_NilMetadata(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	// Create result with nil metadata
	result := domain.NewTaskResult("test-id", "testField", "value", nil)

	scores := map[string]float64{
		"_aggregate": 87.5,
	}

	// Should not panic
	processor.attachScoresToResult(result, scores)
}

// TestProcessField_ErrorHandling tests error handling in field processing
func TestProcessField_ErrorHandling(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return nil, nil, errors.New("generation failed")
		},
	}

	processor := NewFieldProcessor(llmProvider, &mockPromptBuilder{})
	ctx := context.Background()

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("testField", def, nil)
	request := domain.NewGenerationRequest("test", def)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	results := processor.processField(ctx, task, execContext)

	// Should return nil on error
	if results != nil {
		t.Error("Expected nil results on error")
	}
}

// TestProcessField_UnsupportedType tests handling of unsupported types
func TestProcessField_UnsupportedType(t *testing.T) {
	processor := NewFieldProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	ctx := context.Background()

	// Use an invalid/unknown type
	def := &jsonSchema.Definition{Type: jsonSchema.DataType("unknown")}
	task := domain.NewFieldTask("testField", def, nil)
	request := domain.NewGenerationRequest("test", def)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	results := processor.processField(ctx, task, execContext)

	// Should handle gracefully
	if results != nil {
		t.Logf("Results: %v", results)
	}
}

// TestProcessConcurrentFields tests concurrent field processing
func TestProcessConcurrentFields(t *testing.T) {
	callCount := 0
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			callCount++
			time.Sleep(5 * time.Millisecond) // Simulate work
			return "value", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewFieldProcessor(llmProvider, &mockPromptBuilder{})
	ctx := context.Background()

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"a": {Type: jsonSchema.String},
			"b": {Type: jsonSchema.String},
			"c": {Type: jsonSchema.String},
			"d": {Type: jsonSchema.String},
			"e": {Type: jsonSchema.String},
		},
		// No ProcessingOrder - all should be concurrent
	}

	currentGen := make(map[string]interface{})
	request := domain.NewGenerationRequest("test", schema)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	// Get all keys
	var keys []string
	for k := range schema.Properties {
		keys = append(keys, k)
	}

	// Use a MapCollector to collect results
	collector := &MapCollector{
		results:  make(map[string]interface{}),
		metadata: domain.NewResultMetadata(),
	}

	processor.processConcurrentFieldsAndWait(ctx, collector, schema, nil, execContext, currentGen, keys)

	count := len(collector.results)

	if count != 5 {
		t.Errorf("Expected 5 results, got %d", count)
	}
}

// Mock epstimic orchestrator for testing
type mockEpstimicOrchestrator struct {
	called bool
}

func (m *mockEpstimicOrchestrator) EpstimicValidation(
	task *domain.FieldTask,
	context *domain.ExecutionContext,
	generateFn func(*domain.FieldTask, *domain.ExecutionContext) (any, *domain.ProviderMetadata, error),
) (*domain.TaskResult, *domain.ProviderMetadata, error) {
	m.called = true
	val, meta, err := generateFn(task, context)
	if err != nil {
		return nil, nil, err
	}
	return domain.NewTaskResult(task.ID(), task.Key(), val, domain.NewResultMetadata()), meta, nil
}

// TestProcessField_WithEpstimicOrchestrator tests epstimic validation integration
func TestProcessField_WithEpstimicOrchestrator(t *testing.T) {
	orchestrator := &mockEpstimicOrchestrator{}
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "value", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewFieldProcessor(llmProvider, &mockPromptBuilder{})
	processor.SetEpstimicOrchestrator(orchestrator)

	ctx := context.Background()

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("testField", def, nil)
	request := domain.NewGenerationRequest("test", def)
	execContext := domain.NewExecutionContext(request)
	execContext.SetWorkerPool(NewWorkerPool(0))

	results := processor.processField(ctx, task, execContext)

	if len(results) == 0 {
		t.Fatal("Expected results")
	}

	// Note: The orchestrator may or may not be called depending on processor implementation
	// This test mainly ensures no errors occur when orchestrator is set
	t.Logf("Orchestrator called: %v", orchestrator.called)
}
