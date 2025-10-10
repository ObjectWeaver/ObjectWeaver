package analysis

import (
	"firechimp/orchestration/jos/domain"
	"testing"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

func TestDetermineProcessingOrder_NoDependencies(t *testing.T) {
	analyzer := NewDefaultSchemaAnalyzer()

	fields := []*domain.FieldDefinition{
		{Key: "field1", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
		{Key: "field2", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
		{Key: "field3", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
	}

	tasks, err := analyzer.DetermineProcessingOrder(fields)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// All tasks should have no dependencies
	for _, task := range tasks {
		if task.HasDependencies() {
			t.Errorf("Task %s should have no dependencies", task.Key())
		}
	}
}

func TestDetermineProcessingOrder_WithDependencies(t *testing.T) {
	analyzer := NewDefaultSchemaAnalyzer()

	fields := []*domain.FieldDefinition{
		{
			Key:        "country",
			Definition: &jsonSchema.Definition{Type: jsonSchema.String},
		},
		{
			Key: "city",
			Definition: &jsonSchema.Definition{
				Type:            jsonSchema.String,
				ProcessingOrder: []string{"country"},
			},
		},
		{
			Key: "address",
			Definition: &jsonSchema.Definition{
				Type:            jsonSchema.String,
				ProcessingOrder: []string{"city", "country"},
			},
		},
	}

	tasks, err := analyzer.DetermineProcessingOrder(fields)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Create task map for easy lookup
	taskMap := make(map[string]*domain.FieldTask)
	for _, task := range tasks {
		taskMap[task.Key()] = task
	}

	// Verify country has no dependencies
	countryTask := taskMap["country"]
	if countryTask.HasDependencies() {
		t.Errorf("Country should have no dependencies, got %v", countryTask.Dependencies())
	}

	// Verify city depends on country
	cityTask := taskMap["city"]
	if !cityTask.HasDependencies() {
		t.Error("City should have dependencies")
	}
	deps := cityTask.Dependencies()
	if len(deps) != 1 || deps[0] != "country" {
		t.Errorf("City should depend on country, got %v", deps)
	}

	// Verify address depends on city and country
	addressTask := taskMap["address"]
	if !addressTask.HasDependencies() {
		t.Error("Address should have dependencies")
	}
	deps = addressTask.Dependencies()
	if len(deps) != 2 {
		t.Errorf("Address should have 2 dependencies, got %d", len(deps))
	}
	// Check both dependencies exist (order might vary)
	hasCity := false
	hasCountry := false
	for _, dep := range deps {
		if dep == "city" {
			hasCity = true
		}
		if dep == "country" {
			hasCountry = true
		}
	}
	if !hasCity || !hasCountry {
		t.Errorf("Address should depend on city and country, got %v", deps)
	}
}

func TestDetermineProcessingOrder_InvalidDependencies(t *testing.T) {
	analyzer := NewDefaultSchemaAnalyzer()

	fields := []*domain.FieldDefinition{
		{
			Key: "field1",
			Definition: &jsonSchema.Definition{
				Type: jsonSchema.String,
				// References a field that doesn't exist
				ProcessingOrder: []string{"nonexistent", "field2"},
			},
		},
		{
			Key:        "field2",
			Definition: &jsonSchema.Definition{Type: jsonSchema.String},
		},
	}

	tasks, err := analyzer.DetermineProcessingOrder(fields)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Find field1 task
	var field1Task *domain.FieldTask
	for _, task := range tasks {
		if task.Key() == "field1" {
			field1Task = task
			break
		}
	}

	if field1Task == nil {
		t.Fatal("Could not find field1 task")
	}

	// Should only have field2 as dependency (nonexistent should be filtered out)
	deps := field1Task.Dependencies()
	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency (invalid one filtered), got %d: %v", len(deps), deps)
	}
	if deps[0] != "field2" {
		t.Errorf("Expected dependency on field2, got %s", deps[0])
	}
}

func TestDetermineProcessingOrder_ComplexGraph(t *testing.T) {
	analyzer := NewDefaultSchemaAnalyzer()

	// Create a complex dependency graph:
	// A -> C -> E
	// B -> C -> E
	// D (independent)
	fields := []*domain.FieldDefinition{
		{Key: "A", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
		{Key: "B", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
		{
			Key: "C",
			Definition: &jsonSchema.Definition{
				Type:            jsonSchema.String,
				ProcessingOrder: []string{"A", "B"},
			},
		},
		{Key: "D", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
		{
			Key: "E",
			Definition: &jsonSchema.Definition{
				Type:            jsonSchema.String,
				ProcessingOrder: []string{"C"},
			},
		},
	}

	tasks, err := analyzer.DetermineProcessingOrder(fields)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(tasks) != 5 {
		t.Fatalf("Expected 5 tasks, got %d", len(tasks))
	}

	taskMap := make(map[string]*domain.FieldTask)
	for _, task := range tasks {
		taskMap[task.Key()] = task
	}

	// A and B and D should have no dependencies
	for _, key := range []string{"A", "B", "D"} {
		if taskMap[key].HasDependencies() {
			t.Errorf("%s should have no dependencies", key)
		}
	}

	// C should depend on A and B
	cDeps := taskMap["C"].Dependencies()
	if len(cDeps) != 2 {
		t.Errorf("C should have 2 dependencies, got %d", len(cDeps))
	}

	// E should depend on C
	eDeps := taskMap["E"].Dependencies()
	if len(eDeps) != 1 || eDeps[0] != "C" {
		t.Errorf("E should depend on C, got %v", eDeps)
	}
}

func TestDetermineProcessingOrder_EmptyFields(t *testing.T) {
	analyzer := NewDefaultSchemaAnalyzer()

	fields := []*domain.FieldDefinition{}

	tasks, err := analyzer.DetermineProcessingOrder(fields)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestDetermineProcessingOrder_CircularDependency(t *testing.T) {
	// Note: The analyzer itself doesn't detect circular dependencies
	// That's handled by the DependencyAwareStrategy during execution
	// This test just verifies the analyzer creates tasks with the dependencies
	analyzer := NewDefaultSchemaAnalyzer()

	fields := []*domain.FieldDefinition{
		{
			Key: "A",
			Definition: &jsonSchema.Definition{
				Type:            jsonSchema.String,
				ProcessingOrder: []string{"B"},
			},
		},
		{
			Key: "B",
			Definition: &jsonSchema.Definition{
				Type:            jsonSchema.String,
				ProcessingOrder: []string{"A"},
			},
		},
	}

	tasks, err := analyzer.DetermineProcessingOrder(fields)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	taskMap := make(map[string]*domain.FieldTask)
	for _, task := range tasks {
		taskMap[task.Key()] = task
	}

	// Both should have dependencies (circular will be handled by strategy)
	if !taskMap["A"].HasDependencies() {
		t.Error("A should have dependencies")
	}
	if !taskMap["B"].HasDependencies() {
		t.Error("B should have dependencies")
	}
}
