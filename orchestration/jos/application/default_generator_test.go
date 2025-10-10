package application

import (
	"errors"
	"testing"

	"objectGeneration/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

// Mock implementations for testing

type mockSchemaAnalyzer struct {
	analyzeFunc                  func(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error)
	determineProcessingOrderFunc func(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error)
}

func (m *mockSchemaAnalyzer) Analyze(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(schema)
	}
	return &domain.SchemaAnalysis{Fields: []*domain.FieldDefinition{}}, nil
}

func (m *mockSchemaAnalyzer) ExtractFields(schema *jsonSchema.Definition) ([]*domain.FieldDefinition, error) {
	return nil, nil
}

func (m *mockSchemaAnalyzer) DetermineProcessingOrder(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error) {
	if m.determineProcessingOrderFunc != nil {
		return m.determineProcessingOrderFunc(fields)
	}
	return []*domain.FieldTask{}, nil
}

type mockTaskExecutor struct{}

func (m *mockTaskExecutor) Execute(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	return domain.NewTaskResult(task.ID(), task.Key(), "mock_value", domain.NewResultMetadata()), nil
}

func (m *mockTaskExecutor) ExecuteBatch(tasks []*domain.FieldTask, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, len(tasks))
	for i, task := range tasks {
		results[i] = domain.NewTaskResult(task.ID(), task.Key(), "mock_value", domain.NewResultMetadata())
	}
	return results, nil
}

type mockResultAssembler struct {
	assembleFunc func(results []*domain.TaskResult) (*domain.GenerationResult, error)
}

func (m *mockResultAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	if m.assembleFunc != nil {
		return m.assembleFunc(results)
	}
	data := make(map[string]interface{})
	for _, result := range results {
		data[result.Key()] = result.Value()
	}
	return domain.NewGenerationResult(data, domain.NewResultMetadata()), nil
}

type mockExecutionStrategy struct {
	scheduleFunc func(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error)
	executeFunc  func(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error)
}

func (m *mockExecutionStrategy) Schedule(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
	if m.scheduleFunc != nil {
		return m.scheduleFunc(tasks)
	}
	return &domain.ExecutionPlan{
		Stages: []domain.ExecutionStage{
			{Tasks: tasks, Parallel: false},
		},
		Metadata: make(map[string]interface{}),
	}, nil
}

func (m *mockExecutionStrategy) Execute(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(plan, executor, context)
	}
	var results []*domain.TaskResult
	for _, stage := range plan.Stages {
		for _, task := range stage.Tasks {
			result, err := executor.Execute(task, context)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func TestNewDefaultGenerator(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	if gen.analyzer != analyzer {
		t.Error("analyzer not set correctly")
	}
	if gen.executor != executor {
		t.Error("executor not set correctly")
	}
	if gen.assembler != assembler {
		t.Error("assembler not set correctly")
	}
	if gen.strategy != strategy {
		t.Error("strategy not set correctly")
	}
	if gen.plugins == nil {
		t.Error("plugins registry not initialized")
	}
}

func TestGenerate_Success(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if !result.IsSuccess() {
		t.Error("expected successful result")
	}
}

func TestGenerate_PreProcessingError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	// Mock a pre-processor that fails
	gen.plugins.Register(&mockPreProcessor{shouldError: true})

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from pre-processing")
	}
	if err.Error() != "pre-processing failed: mock error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_CacheHit(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	// Mock cache plugin
	cache := &mockCachePlugin{}
	gen.plugins.Register(cache)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	// First call should cache
	result1, err1 := gen.Generate(request)
	if err1 != nil {
		t.Fatalf("first generate failed: %v", err1)
	}

	// Second call should hit cache
	result2, err2 := gen.Generate(request)
	if err2 != nil {
		t.Fatalf("second generate failed: %v", err2)
	}

	if result1 != result2 {
		t.Error("expected same cached result")
	}
}

func TestGenerate_AnalysisError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{
		analyzeFunc: func(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error) {
			return nil, errors.New("analysis error")
		},
	}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from analysis")
	}
	if err.Error() != "schema analysis failed: analysis error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_PlanningError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{
		determineProcessingOrderFunc: func(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error) {
			return nil, errors.New("planning error")
		},
	}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from planning")
	}
	if err.Error() != "task planning failed: planning error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_SchedulingError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{
		scheduleFunc: func(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
			return nil, errors.New("scheduling error")
		},
	}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from scheduling")
	}
	if err.Error() != "scheduling failed: scheduling error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_ExecutionError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{
		executeFunc: func(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
			return nil, errors.New("execution error")
		},
	}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from execution")
	}
	if err.Error() != "execution failed: execution error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_AssemblyError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{
		assembleFunc: func(results []*domain.TaskResult) (*domain.GenerationResult, error) {
			return nil, errors.New("assembly error")
		},
	}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from assembly")
	}
	if err.Error() != "assembly failed: assembly error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_PostProcessingError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	// Mock a post-processor that fails
	gen.plugins.Register(&mockPostProcessor{shouldError: true})

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from post-processing")
	}
	if err.Error() != "post-processing failed: mock error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_ValidationError(t *testing.T) {
	analyzer := &mockSchemaAnalyzer{}
	executor := &mockTaskExecutor{}
	assembler := &mockResultAssembler{}
	strategy := &mockExecutionStrategy{}

	gen := NewDefaultGenerator(analyzer, executor, assembler, strategy)

	// Mock a validator that fails
	gen.plugins.Register(&mockValidationPlugin{shouldError: true})

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.Generate(request)

	if err == nil {
		t.Fatal("expected error from validation")
	}
	if err.Error() != "validation failed: mock error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerateStream_NotSupported(t *testing.T) {
	gen := NewDefaultGenerator(nil, nil, nil, nil)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.GenerateStream(request)

	if err == nil {
		t.Fatal("expected error for unsupported streaming")
	}
	if err.Error() != "streaming not supported by DefaultGenerator" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerateStreamProgressive_NotSupported(t *testing.T) {
	gen := NewDefaultGenerator(nil, nil, nil, nil)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	_, err := gen.GenerateStreamProgressive(request)

	if err == nil {
		t.Fatal("expected error for unsupported progressive streaming")
	}
	if err.Error() != "progressive streaming not supported by DefaultGenerator" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegisterPlugin(t *testing.T) {
	gen := NewDefaultGenerator(nil, nil, nil, nil)

	plugin := &mockPreProcessor{}

	gen.RegisterPlugin(plugin)

	// Check if plugin was registered
	if len(gen.plugins.preProcessors) != 1 {
		t.Error("plugin not registered")
	}
}

func TestGenerateCacheKey(t *testing.T) {
	schema := &jsonSchema.Definition{Instruction: "test instruction"}
	request := domain.NewGenerationRequest("test prompt", schema)

	key := generateCacheKey(request)

	expected := "test prompt_test instruction"
	if key != expected {
		t.Errorf("expected cache key %s, got %s", expected, key)
	}
}

// Mock plugins for testing

type mockPreProcessor struct {
	shouldError bool
}

func (m *mockPreProcessor) Name() string                                   { return "mock-pre" }
func (m *mockPreProcessor) Version() string                                { return "1.0" }
func (m *mockPreProcessor) Initialize(config map[string]interface{}) error { return nil }
func (m *mockPreProcessor) PreProcess(request *domain.GenerationRequest) (*domain.GenerationRequest, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return request, nil
}

type mockPostProcessor struct {
	shouldError bool
}

func (m *mockPostProcessor) Name() string                                   { return "mock-post" }
func (m *mockPostProcessor) Version() string                                { return "1.0" }
func (m *mockPostProcessor) Initialize(config map[string]interface{}) error { return nil }
func (m *mockPostProcessor) PostProcess(result *domain.GenerationResult) (*domain.GenerationResult, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return result, nil
}

type mockValidationPlugin struct {
	shouldError bool
}

func (m *mockValidationPlugin) Name() string                                   { return "mock-validator" }
func (m *mockValidationPlugin) Version() string                                { return "1.0" }
func (m *mockValidationPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *mockValidationPlugin) Validate(result *domain.GenerationResult, schema *jsonSchema.Definition) ([]domain.ValidationError, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return []domain.ValidationError{}, nil
}

type mockCachePlugin struct {
	cache map[string]*domain.GenerationResult
}

func (m *mockCachePlugin) Name() string    { return "mock-cache" }
func (m *mockCachePlugin) Version() string { return "1.0" }
func (m *mockCachePlugin) Initialize(config map[string]interface{}) error {
	m.cache = make(map[string]*domain.GenerationResult)
	return nil
}
func (m *mockCachePlugin) Get(key string) (*domain.GenerationResult, bool) {
	if m.cache == nil {
		m.cache = make(map[string]*domain.GenerationResult)
	}
	result, exists := m.cache[key]
	return result, exists
}
func (m *mockCachePlugin) Set(key string, result *domain.GenerationResult) error {
	if m.cache == nil {
		m.cache = make(map[string]*domain.GenerationResult)
	}
	m.cache[key] = result
	return nil
}
