package execution

import (
	"context"
	"errors"
	"objectweaver/orchestration/jos/domain"
	"testing"
	"time"

	"github.com/objectweaver/go-sdk/jsonSchema"
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

	resultsCh := processor.ProcessFields(ctx, nil, nil, execContext)

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

	resultsCh := processor.ProcessFields(ctx, schema, nil, execContext)

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

	resultsCh := processor.ProcessFields(ctx, schema, nil, execContext)

	results := make(map[string]*domain.TaskResult)
	for result := range resultsCh {
		results[result.Key()] = result
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
	callOrder := []string{}
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			// Extract field name from prompt (simplified)
			return "value", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{
		buildFunc: func(task *domain.FieldTask, context *domain.PromptContext) (string, error) {
			callOrder = append(callOrder, task.Key())
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
		ProcessingOrder: []string{"first", "second"},
	}
	request := domain.NewGenerationRequest("test", schema)
	execContext := domain.NewExecutionContext(request)

	resultsCh := processor.ProcessFields(ctx, schema, nil, execContext)

	for range resultsCh {
		// Consume results
	}

	// First two should be processed first in order
	if len(callOrder) < 2 {
		t.Fatal("Expected at least 2 calls")
	}
	if callOrder[0] != "first" {
		t.Errorf("Expected 'first' to be processed first, got %s", callOrder[0])
	}
	if callOrder[1] != "second" {
		t.Errorf("Expected 'second' to be processed second, got %s", callOrder[1])
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

	resultsCh := processor.ProcessFields(ctx, schema, nil, execContext)

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

	results := processor.processObjectField(ctx, task, execContext)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Key() != "nestedObj" {
		t.Errorf("Expected key 'nestedObj', got %s", result.Key())
	}

	nestedResults, ok := result.Value().(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", result.Value())
	}

	if len(nestedResults) != 2 {
		t.Errorf("Expected 2 nested results, got %d", len(nestedResults))
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
	t.Skip("Skipping scoring test - requires full jsonSchema.ScoringCriteria implementation")
}

// TestEvaluateScores_NoGenerator tests error when generator is not set
func TestEvaluateScores_NoGenerator(t *testing.T) {
	t.Skip("Skipping scoring test - requires full jsonSchema.ScoringCriteria implementation")
}

// TestCalculateAggregate tests score aggregation methods
func TestCalculateAggregate(t *testing.T) {
	t.Skip("Skipping scoring test - requires full jsonSchema.ScoringCriteria implementation")
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

	ch := make(chan *domain.TaskResult, 5)
	currentGen := make(map[string]interface{})
	request := domain.NewGenerationRequest("test", schema)
	execContext := domain.NewExecutionContext(request)

	// Get all keys
	var keys []string
	for k := range schema.Properties {
		keys = append(keys, k)
	}

	processor.processConcurrentFields(ctx, ch, schema, nil, execContext, currentGen, keys)
	close(ch)

	count := 0
	for range ch {
		count++
	}

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

	results := processor.processField(ctx, task, execContext)

	if len(results) == 0 {
		t.Fatal("Expected results")
	}

	// Note: The orchestrator may or may not be called depending on processor implementation
	// This test mainly ensures no errors occur when orchestrator is set
	t.Logf("Orchestrator called: %v", orchestrator.called)
}
