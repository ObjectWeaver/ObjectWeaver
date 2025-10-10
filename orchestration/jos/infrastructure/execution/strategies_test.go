package execution

import (
	"errors"
	"testing"

	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// Mock implementations for testing
type mockTaskExecutor struct {
	executeFunc func(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error)
}

func (m *mockTaskExecutor) Execute(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(task, context)
	}
	return domain.NewTaskResult(task.ID(), task.Key(), "mock_value", nil), nil
}

func (m *mockTaskExecutor) ExecuteBatch(tasks []*domain.FieldTask, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, len(tasks))
	for i, task := range tasks {
		result, err := m.Execute(task, context)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func createMockFieldTask(key string, deps []string) *domain.FieldTask {
	task := domain.NewFieldTask(key, &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	for _, dep := range deps {
		task = task.WithDependency(dep)
	}
	return task
}

func createMockExecutionContext() *domain.ExecutionContext {
	return domain.NewExecutionContext(nil)
}

func TestNewSequentialStrategy(t *testing.T) {
	strategy := NewSequentialStrategy()
	if strategy == nil {
		t.Fatal("NewSequentialStrategy returned nil")
	}
}

func TestSequentialStrategy_Schedule(t *testing.T) {
	strategy := NewSequentialStrategy()
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
	}

	plan, err := strategy.Schedule(tasks)
	if err != nil {
		t.Fatalf("Schedule returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("Schedule returned nil plan")
	}
	if len(plan.Stages) != 1 {
		t.Fatalf("Expected 1 stage, got %d", len(plan.Stages))
	}
	if plan.Stages[0].Parallel {
		t.Fatal("Sequential strategy should not have parallel stages")
	}
	if len(plan.Stages[0].Tasks) != 2 {
		t.Fatalf("Expected 2 tasks in stage, got %d", len(plan.Stages[0].Tasks))
	}
}

func TestSequentialStrategy_Execute(t *testing.T) {
	strategy := NewSequentialStrategy()
	executor := &mockTaskExecutor{}
	context := createMockExecutionContext()
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
	}

	plan, _ := strategy.Schedule(tasks)
	results, err := strategy.Execute(plan, executor, context)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	if results[0].Key() != "task1" {
		t.Errorf("Expected first result key 'task1', got '%s'", results[0].Key())
	}
	if results[1].Key() != "task2" {
		t.Errorf("Expected second result key 'task2', got '%s'", results[1].Key())
	}
}

func TestSequentialStrategy_Execute_WithError(t *testing.T) {
	strategy := NewSequentialStrategy()
	executor := &mockTaskExecutor{
		executeFunc: func(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
			if task.Key() == "task1" {
				return nil, errors.New("execution error")
			}
			return domain.NewTaskResult(task.ID(), task.Key(), "value", nil), nil
		},
	}
	context := createMockExecutionContext()
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
	}

	plan, _ := strategy.Schedule(tasks)
	_, err := strategy.Execute(plan, executor, context)
	if err == nil {
		t.Fatal("Expected error from Execute, got nil")
	}
	if err.Error() != "execution error" {
		t.Errorf("Expected 'execution error', got '%s'", err.Error())
	}
}

func TestNewParallelStrategy(t *testing.T) {
	strategy := NewParallelStrategy(5)
	if strategy == nil {
		t.Fatal("NewParallelStrategy returned nil")
	}
	if strategy.maxConcurrency != 5 {
		t.Errorf("Expected maxConcurrency 5, got %d", strategy.maxConcurrency)
	}
}

func TestParallelStrategy_Schedule(t *testing.T) {
	strategy := NewParallelStrategy(5)
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
	}

	plan, err := strategy.Schedule(tasks)
	if err != nil {
		t.Fatalf("Schedule returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("Schedule returned nil plan")
	}
	if len(plan.Stages) != 1 {
		t.Fatalf("Expected 1 stage, got %d", len(plan.Stages))
	}
	if !plan.Stages[0].Parallel {
		t.Fatal("Parallel strategy should have parallel stages")
	}
	if len(plan.Stages[0].Tasks) != 2 {
		t.Fatalf("Expected 2 tasks in stage, got %d", len(plan.Stages[0].Tasks))
	}
}

func TestParallelStrategy_Execute(t *testing.T) {
	strategy := NewParallelStrategy(5)
	executor := &mockTaskExecutor{}
	context := createMockExecutionContext()
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
		createMockFieldTask("task3", nil),
	}

	plan, _ := strategy.Schedule(tasks)
	results, err := strategy.Execute(plan, executor, context)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}
	// Results may not be in order due to parallelism
	resultKeys := make(map[string]bool)
	for _, result := range results {
		resultKeys[result.Key()] = true
	}
	if len(resultKeys) != 3 {
		t.Errorf("Expected 3 unique result keys, got %d", len(resultKeys))
	}
	if !resultKeys["task1"] || !resultKeys["task2"] || !resultKeys["task3"] {
		t.Errorf("Missing expected keys in results")
	}
}

func TestParallelStrategy_Execute_WithError(t *testing.T) {
	strategy := NewParallelStrategy(5)
	executor := &mockTaskExecutor{
		executeFunc: func(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
			if task.Key() == "task2" {
				return nil, errors.New("parallel execution error")
			}
			return domain.NewTaskResult(task.ID(), task.Key(), "value", nil), nil
		},
	}
	context := createMockExecutionContext()
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
		createMockFieldTask("task3", nil),
	}

	plan, _ := strategy.Schedule(tasks)
	_, err := strategy.Execute(plan, executor, context)
	if err == nil {
		t.Fatal("Expected error from Execute, got nil")
	}
	if err.Error() != "parallel execution error" {
		t.Errorf("Expected 'parallel execution error', got '%s'", err.Error())
	}
}

func TestNewDependencyAwareStrategy(t *testing.T) {
	strategy := NewDependencyAwareStrategy(5)
	if strategy == nil {
		t.Fatal("NewDependencyAwareStrategy returned nil")
	}
	if strategy.maxConcurrency != 5 {
		t.Errorf("Expected maxConcurrency 5, got %d", strategy.maxConcurrency)
	}
}

func TestDependencyAwareStrategy_Schedule_NoDependencies(t *testing.T) {
	strategy := NewDependencyAwareStrategy(5)
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
	}

	plan, err := strategy.Schedule(tasks)
	if err != nil {
		t.Fatalf("Schedule returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("Schedule returned nil plan")
	}
	if len(plan.Stages) != 1 {
		t.Fatalf("Expected 1 stage, got %d", len(plan.Stages))
	}
	if !plan.Stages[0].Parallel {
		t.Fatal("Stage with multiple tasks should be parallel")
	}
	if len(plan.Stages[0].Tasks) != 2 {
		t.Fatalf("Expected 2 tasks in stage, got %d", len(plan.Stages[0].Tasks))
	}
}

func TestDependencyAwareStrategy_Schedule_WithDependencies(t *testing.T) {
	strategy := NewDependencyAwareStrategy(5)
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", []string{"task1"}),
		createMockFieldTask("task3", []string{"task2"}),
	}

	plan, err := strategy.Schedule(tasks)
	if err != nil {
		t.Fatalf("Schedule returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("Schedule returned nil plan")
	}
	if len(plan.Stages) != 3 {
		t.Fatalf("Expected 3 stages, got %d", len(plan.Stages))
	}
	// First stage: task1
	if len(plan.Stages[0].Tasks) != 1 || plan.Stages[0].Tasks[0].Key() != "task1" {
		t.Errorf("Stage 0 should have task1")
	}
	// Second stage: task2
	if len(plan.Stages[1].Tasks) != 1 || plan.Stages[1].Tasks[0].Key() != "task2" {
		t.Errorf("Stage 1 should have task2")
	}
	// Third stage: task3
	if len(plan.Stages[2].Tasks) != 1 || plan.Stages[2].Tasks[0].Key() != "task3" {
		t.Errorf("Stage 2 should have task3")
	}
}

func TestDependencyAwareStrategy_Execute(t *testing.T) {
	strategy := NewDependencyAwareStrategy(5)
	executor := &mockTaskExecutor{}
	context := createMockExecutionContext()
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", []string{"task1"}),
	}

	plan, _ := strategy.Schedule(tasks)
	results, err := strategy.Execute(plan, executor, context)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}

func TestNewDependencyGraph(t *testing.T) {
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", []string{"task1"}),
	}

	graph := NewDependencyGraph(tasks)
	if graph == nil {
		t.Fatal("NewDependencyGraph returned nil")
	}
	if len(graph.tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(graph.tasks))
	}
	if len(graph.graph) != 2 {
		t.Errorf("Expected graph with 2 entries, got %d", len(graph.graph))
	}
}

func TestDependencyGraph_TopoSort_NoDependencies(t *testing.T) {
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", nil),
	}

	graph := NewDependencyGraph(tasks)
	stages := graph.TopoSort()

	if len(stages) != 1 {
		t.Fatalf("Expected 1 stage, got %d", len(stages))
	}
	if len(stages[0].Tasks) != 2 {
		t.Fatalf("Expected 2 tasks in stage, got %d", len(stages[0].Tasks))
	}
	if !stages[0].Parallel {
		t.Fatal("Stage should be parallel")
	}
}

func TestDependencyGraph_TopoSort_WithDependencies(t *testing.T) {
	tasks := []*domain.FieldTask{
		createMockFieldTask("task1", nil),
		createMockFieldTask("task2", []string{"task1"}),
		createMockFieldTask("task3", []string{"task1"}),
		createMockFieldTask("task4", []string{"task2", "task3"}),
	}

	graph := NewDependencyGraph(tasks)
	stages := graph.TopoSort()

	if len(stages) < 2 {
		t.Fatalf("Expected at least 2 stages, got %d", len(stages))
	}
	// Check that task1 is in an early stage
	foundTask1 := false
	for _, stage := range stages {
		for _, task := range stage.Tasks {
			if task.Key() == "task1" {
				foundTask1 = true
				break
			}
		}
		if foundTask1 {
			break
		}
	}
	if !foundTask1 {
		t.Fatal("task1 not found in any stage")
	}
}
