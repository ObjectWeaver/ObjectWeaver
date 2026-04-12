package domain

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

// ============================================
// GenerationRequest Tests
// ============================================

func TestNewGenerationRequest(t *testing.T) {
	prompt := "test prompt"
	schema := &jsonSchema.Definition{Type: jsonSchema.String}

	req := NewGenerationRequest(prompt, schema)

	if req.Prompt() != prompt {
		t.Errorf("Expected prompt %s, got %s", prompt, req.Prompt())
	}
	if req.Schema() != schema {
		t.Errorf("Expected schema to match")
	}
	if req.Context() == nil {
		t.Error("Expected context to be initialized")
	}
	if req.Metadata() == nil {
		t.Error("Expected metadata to be initialized")
	}
	if req.Constraints() == nil {
		t.Error("Expected constraints to be initialized")
	}
}

func TestGenerationRequest_Getters(t *testing.T) {
	prompt := "test prompt"
	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	req := NewGenerationRequest(prompt, schema)

	// Test Prompt()
	if req.Prompt() != prompt {
		t.Errorf("Prompt() getter failed, expected %s, got %s", prompt, req.Prompt())
	}

	// Test Schema()
	if req.Schema() != schema {
		t.Error("Schema() getter failed")
	}

	// Test Context()
	if req.Context() == nil {
		t.Error("Context() getter failed, got nil")
	}

	// Test Metadata()
	if req.Metadata() == nil {
		t.Error("Metadata() getter failed, got nil")
	}

	// Test Constraints()
	if req.Constraints() == nil {
		t.Error("Constraints() getter failed, got nil")
	}
}

func TestGenerationRequest_WithContext(t *testing.T) {
	prompt := "test prompt"
	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	req := NewGenerationRequest(prompt, schema)

	newCtx := context.WithValue(context.Background(), "key", "value")
	newReq := req.WithContext(newCtx)

	// Original should be unchanged
	if req.Context() == newCtx {
		t.Error("Original request context should not change")
	}

	// New request should have new context
	if newReq.Context() != newCtx {
		t.Error("New request should have new context")
	}

	// Other fields should be preserved
	if newReq.Prompt() != prompt {
		t.Error("Prompt should be preserved")
	}
	if newReq.Schema() != schema {
		t.Error("Schema should be preserved")
	}
	if newReq.Constraints() != req.Constraints() {
		t.Error("Constraints should be preserved")
	}
}

func TestGenerationRequest_WithMetadata(t *testing.T) {
	prompt := "test prompt"
	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	req := NewGenerationRequest(prompt, schema)

	key := "testKey"
	value := "testValue"
	newReq := req.WithMetadata(key, value)

	// Original should be unchanged
	if req.Metadata()[key] == value {
		t.Error("Original request metadata should not change")
	}

	// New request should have new metadata
	if newReq.Metadata()[key] != value {
		t.Errorf("New request should have metadata key %s with value %v", key, value)
	}

	// Other fields should be preserved
	if newReq.Prompt() != prompt {
		t.Error("Prompt should be preserved")
	}
	if newReq.Schema() != schema {
		t.Error("Schema should be preserved")
	}
}

func TestGenerationRequest_WithMetadata_Multiple(t *testing.T) {
	req := NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})

	req = req.WithMetadata("key1", "value1")
	req = req.WithMetadata("key2", "value2")
	req = req.WithMetadata("key3", 123)

	if req.Metadata()["key1"] != "value1" {
		t.Error("key1 should be preserved")
	}
	if req.Metadata()["key2"] != "value2" {
		t.Error("key2 should be preserved")
	}
	if req.Metadata()["key3"] != 123 {
		t.Error("key3 should be preserved")
	}
}

func TestGenerationRequest_WithConstraints(t *testing.T) {
	req := NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})
	originalConstraints := req.Constraints()

	newConstraints := &Constraints{
		MaxRetries:     5,
		Timeout:        10 * time.Minute,
		MaxConcurrency: 20,
		VoteQuality:    90,
	}

	newReq := req.WithConstraints(newConstraints)

	// Original should be unchanged
	if req.Constraints() != originalConstraints {
		t.Error("Original request constraints should not change")
	}

	// New request should have new constraints
	if newReq.Constraints() != newConstraints {
		t.Error("New request should have new constraints")
	}
	if newReq.Constraints().MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", newReq.Constraints().MaxRetries)
	}
	if newReq.Constraints().Timeout != 10*time.Minute {
		t.Errorf("Expected Timeout 10m, got %v", newReq.Constraints().Timeout)
	}
}

func TestGenerationRequest_copyMetadata(t *testing.T) {
	req := NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.String})
	req = req.WithMetadata("key1", "value1")
	req = req.WithMetadata("key2", 123)

	copied := req.copyMetadata()

	// Should be equal
	if !reflect.DeepEqual(req.Metadata(), copied) {
		t.Error("Copied metadata should be equal to original")
	}

	// Should be different map instances
	copied["key3"] = "newValue"
	if req.Metadata()["key3"] != nil {
		t.Error("Modifying copied metadata should not affect original")
	}
}

// ============================================
// FieldResultWithMetadata Tests
// ============================================

func TestNewFieldResultWithMetadata(t *testing.T) {
	value := "test value"
	metadata := NewResultMetadata()

	result := NewFieldResultWithMetadata(value, metadata)

	if result.Value != value {
		t.Errorf("Expected value %v, got %v", value, result.Value)
	}
	if result.Metadata != metadata {
		t.Error("Expected metadata to match")
	}
}

func TestFieldResultWithMetadata_Fields(t *testing.T) {
	value := map[string]interface{}{"key": "value"}
	metadata := &ResultMetadata{
		TokensUsed: 100,
		Cost:       0.5,
	}

	result := NewFieldResultWithMetadata(value, metadata)

	if !reflect.DeepEqual(result.Value, value) {
		t.Error("Value field should match")
	}
	if result.Metadata != metadata {
		t.Error("Metadata field should match")
	}
	if result.Metadata.TokensUsed != 100 {
		t.Errorf("Expected TokensUsed 100, got %d", result.Metadata.TokensUsed)
	}
}

// ============================================
// GenerationResult Tests
// ============================================

func TestNewGenerationResult(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	metadata := NewResultMetadata()

	result := NewGenerationResult(data, metadata)

	if !reflect.DeepEqual(result.Data(), data) {
		t.Error("Expected data to match")
	}
	if result.Metadata() != metadata {
		t.Error("Expected metadata to match")
	}
	if result.DetailedData() != nil {
		t.Error("Expected detailed data to be nil")
	}
	if len(result.Errors()) != 0 {
		t.Error("Expected no errors")
	}
	if result.HasDetailedData() {
		t.Error("Expected HasDetailedData to be false")
	}
}

func TestNewGenerationResultWithDetailedData(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	detailedData := map[string]*FieldResultWithMetadata{
		"field1": NewFieldResultWithMetadata("value1", NewResultMetadata()),
	}
	metadata := NewResultMetadata()

	result := NewGenerationResultWithDetailedData(data, detailedData, metadata)

	if !reflect.DeepEqual(result.Data(), data) {
		t.Error("Expected data to match")
	}
	if !reflect.DeepEqual(result.DetailedData(), detailedData) {
		t.Error("Expected detailed data to match")
	}
	if result.Metadata() != metadata {
		t.Error("Expected metadata to match")
	}
	if !result.HasDetailedData() {
		t.Error("Expected HasDetailedData to be true")
	}
	if len(result.Errors()) != 0 {
		t.Error("Expected no errors")
	}
}

func TestNewGenerationResultWithError(t *testing.T) {
	err := errors.New("test error")
	result := NewGenerationResultWithError(err)

	if result.Data() != nil {
		t.Error("Expected data to be nil")
	}
	if result.DetailedData() != nil {
		t.Error("Expected detailed data to be nil")
	}
	if result.Metadata() != nil {
		t.Error("Expected metadata to be nil")
	}
	if len(result.Errors()) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors()))
	}
	if result.Errors()[0] != err {
		t.Error("Expected error to match")
	}
	if result.IsSuccess() {
		t.Error("Expected IsSuccess to be false")
	}
}

func TestGenerationResult_IsSuccess(t *testing.T) {
	// Success case
	successResult := NewGenerationResult(map[string]interface{}{}, NewResultMetadata())
	if !successResult.IsSuccess() {
		t.Error("Result with no errors should be successful")
	}

	// Error case
	errorResult := NewGenerationResultWithError(errors.New("test error"))
	if errorResult.IsSuccess() {
		t.Error("Result with errors should not be successful")
	}
}

func TestGenerationResult_Getters(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	detailedData := map[string]*FieldResultWithMetadata{
		"field1": NewFieldResultWithMetadata("value1", NewResultMetadata()),
	}
	metadata := &ResultMetadata{TokensUsed: 100}

	result := NewGenerationResultWithDetailedData(data, detailedData, metadata)

	// Test Data()
	if !reflect.DeepEqual(result.Data(), data) {
		t.Error("Data() getter failed")
	}

	// Test DetailedData()
	if !reflect.DeepEqual(result.DetailedData(), detailedData) {
		t.Error("DetailedData() getter failed")
	}

	// Test Metadata()
	if result.Metadata() != metadata {
		t.Error("Metadata() getter failed")
	}

	// Test Errors()
	if len(result.Errors()) != 0 {
		t.Error("Errors() getter failed")
	}
}

func TestGenerationResult_HasDetailedData(t *testing.T) {
	// Without detailed data
	result1 := NewGenerationResult(map[string]interface{}{}, NewResultMetadata())
	if result1.HasDetailedData() {
		t.Error("Expected HasDetailedData to be false when not set")
	}

	// With detailed data
	detailedData := map[string]*FieldResultWithMetadata{
		"field1": NewFieldResultWithMetadata("value1", NewResultMetadata()),
	}
	result2 := NewGenerationResultWithDetailedData(map[string]interface{}{}, detailedData, NewResultMetadata())
	if !result2.HasDetailedData() {
		t.Error("Expected HasDetailedData to be true when set")
	}

	// With detailed data nil but flag set would be false
	result3 := NewGenerationResultWithDetailedData(map[string]interface{}{}, nil, NewResultMetadata())
	if result3.HasDetailedData() {
		t.Error("Expected HasDetailedData to be false when detailed data is nil")
	}
}

// ============================================
// ResultMetadata Tests
// ============================================

func TestNewResultMetadata(t *testing.T) {
	metadata := NewResultMetadata()

	if metadata.Duration != 0 {
		t.Errorf("Expected Duration 0, got %v", metadata.Duration)
	}
	if metadata.ModelUsed != "" {
		t.Errorf("Expected ModelUsed to be empty, got %s", metadata.ModelUsed)
	}
	if metadata.FieldCount != 0 {
		t.Errorf("Expected FieldCount 0, got %d", metadata.FieldCount)
	}
	if metadata.TokensUsed != 0 {
		t.Errorf("Expected TokensUsed 0, got %d", metadata.TokensUsed)
	}
	if metadata.Cost != 0 {
		t.Errorf("Expected Cost 0, got %f", metadata.Cost)
	}
}

func TestResultMetadata_AddCost(t *testing.T) {
	metadata := NewResultMetadata()

	metadata.AddCost(1.5)
	if metadata.Cost != 1.5 {
		t.Errorf("Expected Cost 1.5, got %f", metadata.Cost)
	}

	metadata.AddCost(0.5)
	if metadata.Cost != 2.0 {
		t.Errorf("Expected Cost 2.0, got %f", metadata.Cost)
	}

	metadata.AddCost(2.5)
	if metadata.Cost != 4.5 {
		t.Errorf("Expected Cost 4.5, got %f", metadata.Cost)
	}
}

func TestResultMetadata_AddTokens(t *testing.T) {
	metadata := NewResultMetadata()

	metadata.AddTokens(100)
	if metadata.TokensUsed != 100 {
		t.Errorf("Expected TokensUsed 100, got %d", metadata.TokensUsed)
	}

	metadata.AddTokens(50)
	if metadata.TokensUsed != 150 {
		t.Errorf("Expected TokensUsed 150, got %d", metadata.TokensUsed)
	}

	metadata.AddTokens(25)
	if metadata.TokensUsed != 175 {
		t.Errorf("Expected TokensUsed 175, got %d", metadata.TokensUsed)
	}
}

func TestResultMetadata_IncrementFieldCount(t *testing.T) {
	metadata := NewResultMetadata()

	if metadata.FieldCount != 0 {
		t.Errorf("Expected initial FieldCount 0, got %d", metadata.FieldCount)
	}

	metadata.IncrementFieldCount()
	if metadata.FieldCount != 1 {
		t.Errorf("Expected FieldCount 1, got %d", metadata.FieldCount)
	}

	metadata.IncrementFieldCount()
	if metadata.FieldCount != 2 {
		t.Errorf("Expected FieldCount 2, got %d", metadata.FieldCount)
	}

	metadata.IncrementFieldCount()
	if metadata.FieldCount != 3 {
		t.Errorf("Expected FieldCount 3, got %d", metadata.FieldCount)
	}
}

func TestResultMetadata_AllFields(t *testing.T) {
	metadata := &ResultMetadata{
		TokensUsed: 500,
		Cost:       2.5,
		Duration:   100 * time.Millisecond,
		ModelUsed:  "gpt-4",
		FieldCount: 10,
		Choices:    []Choice{},
	}

	if metadata.TokensUsed != 500 {
		t.Errorf("Expected TokensUsed 500, got %d", metadata.TokensUsed)
	}
	if metadata.Cost != 2.5 {
		t.Errorf("Expected Cost 2.5, got %f", metadata.Cost)
	}
	if metadata.Duration != 100*time.Millisecond {
		t.Errorf("Expected Duration 100ms, got %v", metadata.Duration)
	}
	if metadata.ModelUsed != "gpt-4" {
		t.Errorf("Expected ModelUsed gpt-4, got %s", metadata.ModelUsed)
	}
	if metadata.FieldCount != 10 {
		t.Errorf("Expected FieldCount 10, got %d", metadata.FieldCount)
	}
}

// ============================================
// Constraints Tests
// ============================================

func TestDefaultConstraints(t *testing.T) {
	constraints := DefaultConstraints()

	if constraints.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", constraints.MaxRetries)
	}
	if constraints.Timeout != 5*time.Minute {
		t.Errorf("Expected Timeout 5m, got %v", constraints.Timeout)
	}
	if constraints.MaxConcurrency != 10 {
		t.Errorf("Expected MaxConcurrency 10, got %d", constraints.MaxConcurrency)
	}
	if constraints.VoteQuality != 85 {
		t.Errorf("Expected VoteQuality 85, got %d", constraints.VoteQuality)
	}
}

func TestConstraints_CustomValues(t *testing.T) {
	constraints := &Constraints{
		MaxRetries:     5,
		Timeout:        10 * time.Minute,
		MaxConcurrency: 20,
		VoteQuality:    95,
	}

	if constraints.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", constraints.MaxRetries)
	}
	if constraints.Timeout != 10*time.Minute {
		t.Errorf("Expected Timeout 10m, got %v", constraints.Timeout)
	}
	if constraints.MaxConcurrency != 20 {
		t.Errorf("Expected MaxConcurrency 20, got %d", constraints.MaxConcurrency)
	}
	if constraints.VoteQuality != 95 {
		t.Errorf("Expected VoteQuality 95, got %d", constraints.VoteQuality)
	}
}

// ============================================
// FieldTask Tests
// ============================================

func TestNewFieldTask(t *testing.T) {
	key := "testKey"
	definition := &jsonSchema.Definition{Type: jsonSchema.String}

	task := NewFieldTask(key, definition, nil)

	if task.Key() != key {
		t.Errorf("Expected key %s, got %s", key, task.Key())
	}
	if task.Definition() != definition {
		t.Error("Expected definition to match")
	}
	if task.Parent() != nil {
		t.Error("Expected parent to be nil")
	}
	if len(task.Dependencies()) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(task.Dependencies()))
	}
	if task.Priority() != 0 {
		t.Errorf("Expected priority 0, got %d", task.Priority())
	}
	if task.ID() == "" {
		t.Error("Expected ID to be generated")
	}
	if len(task.Path()) != 1 || task.Path()[0] != key {
		t.Error("Expected path to contain only key")
	}
}

func TestNewFieldTask_WithParent(t *testing.T) {
	parentDef := &jsonSchema.Definition{Type: jsonSchema.Object}
	parent := NewFieldTask("parent", parentDef, nil)

	childKey := "child"
	childDef := &jsonSchema.Definition{Type: jsonSchema.String}
	child := NewFieldTask(childKey, childDef, parent)

	if child.Parent() != parent {
		t.Error("Expected parent to be set")
	}
	if len(child.Path()) != 2 {
		t.Errorf("Expected path length 2, got %d", len(child.Path()))
	}
	if child.Path()[0] != "parent" || child.Path()[1] != "child" {
		t.Errorf("Expected path [parent, child], got %v", child.Path())
	}
}

func TestFieldTask_Getters(t *testing.T) {
	key := "testKey"
	definition := &jsonSchema.Definition{Type: jsonSchema.String}
	parent := NewFieldTask("parent", &jsonSchema.Definition{Type: jsonSchema.Object}, nil)

	task := NewFieldTask(key, definition, parent)

	// Test ID()
	if task.ID() == "" {
		t.Error("ID() getter failed")
	}

	// Test Key()
	if task.Key() != key {
		t.Errorf("Key() getter failed, expected %s, got %s", key, task.Key())
	}

	// Test Definition()
	if task.Definition() != definition {
		t.Error("Definition() getter failed")
	}

	// Test Parent()
	if task.Parent() != parent {
		t.Error("Parent() getter failed")
	}

	// Test Dependencies()
	if task.Dependencies() == nil {
		t.Error("Dependencies() getter failed")
	}

	// Test Path()
	if task.Path() == nil {
		t.Error("Path() getter failed")
	}

	// Test Priority()
	if task.Priority() != 0 {
		t.Error("Priority() getter failed")
	}
}

func TestFieldTask_HasDependencies(t *testing.T) {
	task := NewFieldTask("test", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	// Initially no dependencies
	if task.HasDependencies() {
		t.Error("Expected HasDependencies to be false")
	}

	// Add a dependency
	taskWithDep := task.WithDependency("dep1")
	if !taskWithDep.HasDependencies() {
		t.Error("Expected HasDependencies to be true")
	}

	// Original should be unchanged
	if task.HasDependencies() {
		t.Error("Original task should not have dependencies")
	}
}

func TestFieldTask_WithDependency(t *testing.T) {
	task := NewFieldTask("test", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	dep1 := "dependency1"
	newTask := task.WithDependency(dep1)

	// Original should be unchanged
	if len(task.Dependencies()) != 0 {
		t.Error("Original task dependencies should be unchanged")
	}

	// New task should have dependency
	if len(newTask.Dependencies()) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(newTask.Dependencies()))
	}
	if newTask.Dependencies()[0] != dep1 {
		t.Errorf("Expected dependency %s, got %s", dep1, newTask.Dependencies()[0])
	}

	// Add multiple dependencies
	dep2 := "dependency2"
	newerTask := newTask.WithDependency(dep2)
	if len(newerTask.Dependencies()) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(newerTask.Dependencies()))
	}
	if newerTask.Dependencies()[1] != dep2 {
		t.Errorf("Expected second dependency %s, got %s", dep2, newerTask.Dependencies()[1])
	}
}

func TestFieldTask_WithPriority(t *testing.T) {
	task := NewFieldTask("test", &jsonSchema.Definition{Type: jsonSchema.String}, nil)

	priority := 5
	newTask := task.WithPriority(priority)

	// Original should be unchanged
	if task.Priority() != 0 {
		t.Error("Original task priority should be unchanged")
	}

	// New task should have priority
	if newTask.Priority() != priority {
		t.Errorf("Expected priority %d, got %d", priority, newTask.Priority())
	}
}

func TestGenerateTaskID(t *testing.T) {
	// Simple case
	id1 := generateTaskID("key1", []string{"key1"})
	if id1 == "" {
		t.Error("Expected non-empty ID")
	}

	// With path
	id2 := generateTaskID("key2", []string{"parent", "key2"})
	if id2 == "" {
		t.Error("Expected non-empty ID")
	}
	if !contains(id2, "key2") {
		t.Error("Expected ID to contain key")
	}

	// Different keys should produce different IDs
	if id1 == id2 {
		t.Error("Different keys should produce different IDs")
	}
}

// ============================================
// TaskResult Tests
// ============================================

func TestNewTaskResult(t *testing.T) {
	taskID := "task123"
	key := "testKey"
	value := "testValue"
	metadata := NewResultMetadata()

	result := NewTaskResult(taskID, key, value, metadata)

	if result.TaskID() != taskID {
		t.Errorf("Expected taskID %s, got %s", taskID, result.TaskID())
	}
	if result.Key() != key {
		t.Errorf("Expected key %s, got %s", key, result.Key())
	}
	if result.Value() != value {
		t.Errorf("Expected value %v, got %v", value, result.Value())
	}
	if result.Metadata() != metadata {
		t.Error("Expected metadata to match")
	}
	if result.Error() != nil {
		t.Error("Expected no error")
	}
	if !result.IsSuccess() {
		t.Error("Expected IsSuccess to be true")
	}
}

func TestNewTaskResultWithError(t *testing.T) {
	taskID := "task123"
	key := "testKey"
	err := errors.New("test error")

	result := NewTaskResultWithError(taskID, key, err)

	if result.TaskID() != taskID {
		t.Errorf("Expected taskID %s, got %s", taskID, result.TaskID())
	}
	if result.Key() != key {
		t.Errorf("Expected key %s, got %s", key, result.Key())
	}
	if result.Error() != err {
		t.Error("Expected error to match")
	}
	if result.IsSuccess() {
		t.Error("Expected IsSuccess to be false")
	}
}

func TestTaskResult_Getters(t *testing.T) {
	taskID := "task123"
	key := "testKey"
	value := map[string]interface{}{"data": "value"}
	metadata := &ResultMetadata{TokensUsed: 100}

	result := NewTaskResult(taskID, key, value, metadata)

	// Test TaskID()
	if result.TaskID() != taskID {
		t.Error("TaskID() getter failed")
	}

	// Test Key()
	if result.Key() != key {
		t.Error("Key() getter failed")
	}

	// Test Value()
	if !reflect.DeepEqual(result.Value(), value) {
		t.Error("Value() getter failed")
	}

	// Test Metadata()
	if result.Metadata() != metadata {
		t.Error("Metadata() getter failed")
	}

	// Test Path()
	if result.Path() != nil {
		t.Error("Path() should be nil initially")
	}

	// Test Error()
	if result.Error() != nil {
		t.Error("Error() getter failed")
	}
}

func TestTaskResult_IsSuccess(t *testing.T) {
	// Success case
	successResult := NewTaskResult("task1", "key1", "value1", NewResultMetadata())
	if !successResult.IsSuccess() {
		t.Error("Result with no error should be successful")
	}

	// Error case
	errorResult := NewTaskResultWithError("task2", "key2", errors.New("test error"))
	if errorResult.IsSuccess() {
		t.Error("Result with error should not be successful")
	}
}

func TestTaskResult_WithPath(t *testing.T) {
	result := NewTaskResult("task1", "key1", "value1", NewResultMetadata())
	path := []string{"parent", "child", "key1"}

	newResult := result.WithPath(path)

	// Original should be unchanged
	if result.Path() != nil {
		t.Error("Original result path should be nil")
	}

	// New result should have path
	if !reflect.DeepEqual(newResult.Path(), path) {
		t.Errorf("Expected path %v, got %v", path, newResult.Path())
	}

	// Other fields should be preserved
	if newResult.TaskID() != "task1" {
		t.Error("TaskID should be preserved")
	}
	if newResult.Key() != "key1" {
		t.Error("Key should be preserved")
	}
	if newResult.Value() != "value1" {
		t.Error("Value should be preserved")
	}
}

// ============================================
// Helper Functions
// ============================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
