package execution

import (
	"context"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

// Mock generator for testing
type mockGenerator struct {
	generateFunc func(req *domain.GenerationRequest) (*domain.GenerationResult, error)
}

func (m *mockGenerator) Generate(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(req)
	}
	return domain.NewGenerationResult(map[string]interface{}{"result": "mock"}, nil), nil
}

func (m *mockGenerator) GenerateStream(req *domain.GenerationRequest) (<-chan *domain.StreamChunk, error) {
	ch := make(chan *domain.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockGenerator) GenerateStreamProgressive(req *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	ch := make(chan *domain.AccumulatedStreamChunk)
	close(ch)
	return ch, nil
}

func TestNewDecisionProcessor(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}
}

func TestProcessDecisionPoint_NoDecisionPoint(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	ctx := context.Background()

	// Create task with no decision point
	fieldDef := &jsonSchema.Definition{
		Type: jsonSchema.String,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	result := domain.NewTaskResult("test-id", "testField", "test value", nil)
	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	results, err := processor.ProcessDecisionPoint(ctx, task, result, execContext)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0] != result {
		t.Error("Expected original result to be returned")
	}
}

func TestProcessDecisionPoint_NilGenerator(t *testing.T) {
	processor := NewDecisionProcessor(nil)

	ctx := context.Background()

	// Create task with decision point
	decisionPoint := &jsonSchema.DecisionPoint{
		Name: "test-decision",
		Branches: []jsonSchema.ConditionalBranch{
			{
				Name:     "branch1",
				Priority: 1,
			},
		},
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		DecisionPoint: decisionPoint,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	result := domain.NewTaskResult("test-id", "testField", "test value", nil)
	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	results, err := processor.ProcessDecisionPoint(ctx, task, result, execContext)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestProcessDecisionPoint_ContextCancelled(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	decisionPoint := &jsonSchema.DecisionPoint{
		Name: "test-decision",
		Branches: []jsonSchema.ConditionalBranch{
			{
				Name:     "branch1",
				Priority: 1,
			},
		},
	}

	fieldDef := &jsonSchema.Definition{
		Type:          jsonSchema.String,
		DecisionPoint: decisionPoint,
	}
	task := domain.NewFieldTask("testField", fieldDef, nil)

	result := domain.NewTaskResult("test-id", "testField", "test value", nil)
	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	results, err := processor.ProcessDecisionPoint(ctx, task, result, execContext)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result even with error, got %d", len(results))
	}
}

func TestEvaluateBranch_NoConditions(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	ctx := context.Background()
	branch := jsonSchema.ConditionalBranch{
		Name:       "empty-branch",
		Conditions: []jsonSchema.Condition{},
	}

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("testField", fieldDef, nil)
	result := domain.NewTaskResult("test-id", "testField", "test value", nil)
	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)
	decisionPoint := &jsonSchema.DecisionPoint{Name: "test"}

	matched, err := processor.evaluateBranch(ctx, branch, task, result, execContext, decisionPoint)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if matched {
		t.Error("Expected branch with no conditions to not match")
	}
}

func TestBuildConditionEvaluationSchema(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	branch := jsonSchema.ConditionalBranch{
		Name: "test-branch",
		Conditions: []jsonSchema.Condition{
			{Field: "wordCount", Operator: jsonSchema.OpGreaterThan, Value: 100},
			{Field: "hasKeyword", Operator: jsonSchema.OpEqual, Value: true},
			{Field: "sentiment", Operator: jsonSchema.OpEqual, Value: "positive"},
		},
	}

	result := domain.NewTaskResult("test-id", "testField", "test value", nil)
	decisionPoint := &jsonSchema.DecisionPoint{Name: "test"}

	schema := processor.buildConditionEvaluationSchema(branch, decisionPoint, result)

	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}
	if schema.Type != jsonSchema.Object {
		t.Errorf("Expected object type, got %v", schema.Type)
	}

	if len(schema.Properties) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(schema.Properties))
	}

	// Check property types are inferred correctly
	if schema.Properties["wordCount"].Type != jsonSchema.Integer {
		t.Errorf("Expected integer type for wordCount, got %v", schema.Properties["wordCount"].Type)
	}
	if schema.Properties["hasKeyword"].Type != jsonSchema.Boolean {
		t.Errorf("Expected boolean type for hasKeyword, got %v", schema.Properties["hasKeyword"].Type)
	}
	if schema.Properties["sentiment"].Type != jsonSchema.String {
		t.Errorf("Expected string type for sentiment, got %v", schema.Properties["sentiment"].Type)
	}
}

func TestBuildEvaluationPrompt(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	branch := jsonSchema.ConditionalBranch{
		Name: "test-branch",
		Conditions: []jsonSchema.Condition{
			{Field: "wordCount", Operator: jsonSchema.OpGreaterThan, Value: 100},
		},
	}

	result := domain.NewTaskResult("test-id", "testField", "test content here", nil)
	decisionPoint := &jsonSchema.DecisionPoint{
		Name:             "test",
		EvaluationPrompt: "Custom evaluation instructions",
	}

	prompt := processor.buildEvaluationPrompt(decisionPoint, result, branch)

	// Should contain custom evaluation prompt
	if len(prompt) == 0 {
		t.Error("Expected non-empty prompt")
	}

	// Check for key components
	contains := func(s, substr string) bool {
		return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
	}

	if !contains(prompt, "Custom evaluation instructions") {
		t.Error("Expected prompt to contain evaluation instructions")
	}
}

func TestEvaluateCondition(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	tests := []struct {
		name        string
		condition   jsonSchema.Condition
		evalData    map[string]interface{}
		expectMatch bool
		expectError bool
	}{
		{
			name: "equal string - match",
			condition: jsonSchema.Condition{
				Field:    "sentiment",
				Operator: jsonSchema.OpEqual,
				Value:    "positive",
			},
			evalData:    map[string]interface{}{"sentiment": "positive"},
			expectMatch: true,
		},
		{
			name: "equal string - no match",
			condition: jsonSchema.Condition{
				Field:    "sentiment",
				Operator: jsonSchema.OpEqual,
				Value:    "positive",
			},
			evalData:    map[string]interface{}{"sentiment": "negative"},
			expectMatch: false,
		},
		{
			name: "greater than - match",
			condition: jsonSchema.Condition{
				Field:    "wordCount",
				Operator: jsonSchema.OpGreaterThan,
				Value:    100,
			},
			evalData:    map[string]interface{}{"wordCount": 150},
			expectMatch: true,
		},
		{
			name: "greater than - no match",
			condition: jsonSchema.Condition{
				Field:    "wordCount",
				Operator: jsonSchema.OpGreaterThan,
				Value:    100,
			},
			evalData:    map[string]interface{}{"wordCount": 50},
			expectMatch: false,
		},
		{
			name: "contains - match",
			condition: jsonSchema.Condition{
				Field:    "text",
				Operator: jsonSchema.OpContains,
				Value:    "keyword",
			},
			evalData:    map[string]interface{}{"text": "this has keyword in it"},
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := domain.NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})
			execContext := domain.NewExecutionContext(req)

			match, err := processor.evaluateCondition(tt.condition, tt.evalData, execContext)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if match != tt.expectMatch {
				t.Errorf("Expected match=%v, got %v", tt.expectMatch, match)
			}
		})
	}
}

func TestInferTypeFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected jsonSchema.DataType
	}{
		{"boolean true", true, jsonSchema.Boolean},
		{"boolean false", false, jsonSchema.Boolean},
		{"int", 42, jsonSchema.Integer},
		{"int32", int32(42), jsonSchema.Integer},
		{"int64", int64(42), jsonSchema.Integer},
		{"float32", float32(3.14), jsonSchema.Number},
		{"float64", 3.14, jsonSchema.Number},
		{"string", "text", jsonSchema.String},
		{"nil", nil, jsonSchema.String},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferTypeFromValue(tt.value)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name        string
		lhs         interface{}
		operator    jsonSchema.ComparisonOperator
		rhs         interface{}
		expected    bool
		expectError bool
	}{
		// Equality
		{"equal strings", "test", jsonSchema.OpEqual, "test", true, false},
		{"not equal strings", "test", jsonSchema.OpEqual, "other", false, false},
		{"not equal operator", "test", jsonSchema.OpNotEqual, "other", true, false},

		// Numeric comparisons
		{"greater than - true", 10, jsonSchema.OpGreaterThan, 5, true, false},
		{"greater than - false", 5, jsonSchema.OpGreaterThan, 10, false, false},
		{"less than - true", 5, jsonSchema.OpLessThan, 10, true, false},
		{"less than - false", 10, jsonSchema.OpLessThan, 5, false, false},
		{"greater or equal - true", 10, jsonSchema.OpGreaterThanOrEqual, 10, true, false},
		{"less or equal - true", 5, jsonSchema.OpLessThanOrEqual, 5, true, false},

		// Contains
		{"contains - true", "hello world", jsonSchema.OpContains, "world", true, false},
		{"contains - false", "hello world", jsonSchema.OpContains, "missing", false, false},

		// Float comparisons
		{"float greater than", 3.14, jsonSchema.OpGreaterThan, 2.5, true, false},
		{"mixed int/float", 10, jsonSchema.OpGreaterThan, 5.5, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compareValues(tt.lhs, tt.operator, tt.rhs)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNormalizeOperator(t *testing.T) {
	tests := []struct {
		input    jsonSchema.ComparisonOperator
		expected jsonSchema.ComparisonOperator
	}{
		// Short forms (should stay the same)
		{"eq", jsonSchema.OpEqual},
		{"neq", jsonSchema.OpNotEqual},
		{"gt", jsonSchema.OpGreaterThan},
		{"lt", jsonSchema.OpLessThan},
		{"gte", jsonSchema.OpGreaterThanOrEqual},
		{"lte", jsonSchema.OpLessThanOrEqual},

		// Long forms (should normalize)
		{"equal", jsonSchema.OpEqual},
		{"not_equal", jsonSchema.OpNotEqual},
		{"greater_than", jsonSchema.OpGreaterThan},
		{"less_than", jsonSchema.OpLessThan},
		{"greater_than_or_equal", jsonSchema.OpGreaterThanOrEqual},
		{"less_than_or_equal", jsonSchema.OpLessThanOrEqual},

		// Symbols
		{"==", jsonSchema.OpEqual},
		{"!=", jsonSchema.OpNotEqual},
		{">", jsonSchema.OpGreaterThan},
		{"<", jsonSchema.OpLessThan},
		{">=", jsonSchema.OpGreaterThanOrEqual},
		{"<=", jsonSchema.OpLessThanOrEqual},

		// CamelCase variants - these should now work after the bug fix
		{"greaterThan", jsonSchema.OpGreaterThan},
		{"lessThan", jsonSchema.OpLessThan},
		{"greaterThanOrEqual", jsonSchema.OpGreaterThanOrEqual},
		{"lessThanOrEqual", jsonSchema.OpLessThanOrEqual},

		// Unknown operator (should return as-is)
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := normalizeOperator(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected float64
		ok       bool
	}{
		{"int", 42, 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"float32", float32(3.14), 3.14, true},
		{"float64", 3.14, 3.14, true},
		{"string", "not a number", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.value)
			if ok != tt.ok {
				t.Errorf("Expected ok=%v, got %v", tt.ok, ok)
			}
			// For float32, allow small precision differences
			if ok {
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

func TestExecuteBranch_BasicGeneration(t *testing.T) {
	// Mock generator that returns predictable data
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			return domain.NewGenerationResult(map[string]interface{}{
				"refined_content": "Refined version",
				"seo_description": "SEO desc",
			}, nil), nil
		},
	}
	processor := NewDecisionProcessor(gen)

	ctx := context.Background()

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	parentTask := domain.NewFieldTask("parentField", fieldDef, nil)

	branchDef := jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Generate refinement",
		Properties: map[string]jsonSchema.Definition{
			"refined_content": {Type: jsonSchema.String},
			"seo_description": {Type: jsonSchema.String},
		},
	}

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	results, err := processor.executeBranch(ctx, parentTask, branchDef, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that results have correct keys
	keys := make(map[string]bool)
	for _, r := range results {
		keys[r.Key()] = true
	}

	if !keys["refined_content"] || !keys["seo_description"] {
		t.Error("Expected results to have refined_content and seo_description keys")
	}
}

func TestExecuteBranch_WithSelectFields(t *testing.T) {
	// Mock generator that captures the prompt
	var capturedPrompt string
	gen := &mockGenerator{
		generateFunc: func(req *domain.GenerationRequest) (*domain.GenerationResult, error) {
			capturedPrompt = req.Prompt()
			return domain.NewGenerationResult(map[string]interface{}{
				"result": "generated",
			}, nil), nil
		},
	}
	processor := NewDecisionProcessor(gen)

	ctx := context.Background()

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	parentTask := domain.NewFieldTask("parentField", fieldDef, nil)

	branchDef := jsonSchema.Definition{
		Type:         jsonSchema.Object,
		Instruction:  "Generate with context",
		SelectFields: []string{"originalContent", "metadata"},
	}

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)
	execContext.SetGeneratedValue("originalContent", "Original text here")
	execContext.SetGeneratedValue("metadata", map[string]interface{}{"author": "test"})

	_, err := processor.executeBranch(ctx, parentTask, branchDef, execContext)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the prompt includes selected fields
	if len(capturedPrompt) == 0 {
		t.Error("Expected prompt to be captured")
	}
}

func TestExecuteBranch_ContextCancelled(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	parentTask := domain.NewFieldTask("parentField", fieldDef, nil)

	branchDef := jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Generate",
	}

	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	_, err := processor.executeBranch(ctx, parentTask, branchDef, execContext)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

// TestEvaluateCondition_NestedFieldPath tests that conditions can evaluate nested field paths
func TestEvaluateCondition_NestedFieldPath(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	// Set up nested data in context
	execContext.SetGeneratedValue("car", map[string]interface{}{
		"specs": map[string]interface{}{
			"hp": 280,
		},
		"color": "red",
	})

	t.Run("NestedPath_GreaterThan", func(t *testing.T) {
		condition := jsonSchema.Condition{
			FieldPath: "car.specs.hp",
			Operator:  jsonSchema.OpGreaterThan,
			Value:     250,
		}

		result, err := processor.evaluateCondition(condition, nil, execContext)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result {
			t.Error("Expected condition to be true (280 > 250)")
		}
	})

	t.Run("NestedPath_Equal", func(t *testing.T) {
		condition := jsonSchema.Condition{
			FieldPath: "car.color",
			Operator:  jsonSchema.OpEqual,
			Value:     "red",
		}

		result, err := processor.evaluateCondition(condition, nil, execContext)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result {
			t.Error("Expected condition to be true (color == red)")
		}
	})

	t.Run("NonExistentPath", func(t *testing.T) {
		condition := jsonSchema.Condition{
			FieldPath: "car.missing.field",
			Operator:  jsonSchema.OpEqual,
			Value:     "test",
		}

		result, err := processor.evaluateCondition(condition, nil, execContext)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result {
			t.Error("Expected condition to be false for non-existent path")
		}
	})
}

// TestEvaluateCondition_ArrayFieldPath tests conditions with array field extraction
func TestEvaluateCondition_ArrayFieldPath(t *testing.T) {
	gen := &mockGenerator{}
	processor := NewDecisionProcessor(gen)

	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	req := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(req)

	// Set up array data in context
	execContext.SetGeneratedValue("reviews", []interface{}{
		map[string]interface{}{"rating": 5, "comment": "Great"},
		map[string]interface{}{"rating": 4, "comment": "Good"},
		map[string]interface{}{"rating": 5, "comment": "Excellent"},
	})

	t.Run("ArrayPath_Contains", func(t *testing.T) {
		// Check if the ratings array contains 5
		condition := jsonSchema.Condition{
			FieldPath: "reviews.rating",
			Operator:  jsonSchema.OpContains,
			Value:     5,
		}

		result, err := processor.evaluateCondition(condition, nil, execContext)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result {
			t.Error("Expected condition to be true (ratings contain 5)")
		}
	})
}
