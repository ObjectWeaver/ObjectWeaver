package application

import (
	"errors"
	"testing"

	"objectGeneration/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

// Mock implementations for testing ProgressiveGenerator

type mockProgressiveSchemaAnalyzer struct {
	analyzeFunc                  func(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error)
	determineProcessingOrderFunc func(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error)
}

func (m *mockProgressiveSchemaAnalyzer) Analyze(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(schema)
	}
	return &domain.SchemaAnalysis{Fields: []*domain.FieldDefinition{
		{Key: "testField", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
	}}, nil
}

func (m *mockProgressiveSchemaAnalyzer) ExtractFields(schema *jsonSchema.Definition) ([]*domain.FieldDefinition, error) {
	return []*domain.FieldDefinition{
		{Key: "testField", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
	}, nil
}

func (m *mockProgressiveSchemaAnalyzer) DetermineProcessingOrder(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error) {
	if m.determineProcessingOrderFunc != nil {
		return m.determineProcessingOrderFunc(fields)
	}
	tasks := make([]*domain.FieldTask, len(fields))
	for i, field := range fields {
		tasks[i] = domain.NewFieldTask(field.Key, field.Definition, nil)
	}
	return tasks, nil
}

type mockProgressiveTaskExecutor struct{}

func (m *mockProgressiveTaskExecutor) Execute(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	return domain.NewTaskResult(task.ID(), task.Key(), "mock_value", domain.NewResultMetadata()), nil
}

func (m *mockProgressiveTaskExecutor) ExecuteBatch(tasks []*domain.FieldTask, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, len(tasks))
	for i, task := range tasks {
		results[i] = domain.NewTaskResult(task.ID(), task.Key(), "mock_value", domain.NewResultMetadata())
	}
	return results, nil
}

type mockProgressiveAssembler struct {
	assembleProgressiveFunc func(tokenStream <-chan *domain.TokenStreamChunk) (<-chan *domain.AccumulatedStreamChunk, error)
}

func (m *mockProgressiveAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	data := make(map[string]interface{})
	for _, result := range results {
		data[result.Key()] = result.Value()
	}
	return domain.NewGenerationResult(data, domain.NewResultMetadata()), nil
}

func (m *mockProgressiveAssembler) AssembleStreaming(results <-chan *domain.TaskResult) (<-chan *domain.StreamChunk, error) {
	out := make(chan *domain.StreamChunk, 100)
	go func() {
		defer close(out)
		for result := range results {
			out <- domain.NewStreamChunk(result.Key(), result.Value().(string))
		}
	}()
	return out, nil
}

func (m *mockProgressiveAssembler) AssembleProgressive(tokenStream <-chan *domain.TokenStreamChunk) (<-chan *domain.AccumulatedStreamChunk, error) {
	if m.assembleProgressiveFunc != nil {
		return m.assembleProgressiveFunc(tokenStream)
	}
	// Default mock implementation
	out := make(chan *domain.AccumulatedStreamChunk, 100)
	go func() {
		defer close(out)
		currentMap := make(map[string]interface{})
		var lastChunk *domain.TokenStreamChunk
		for chunk := range tokenStream {
			currentMap[chunk.Key] = chunk.Partial
			lastChunk = chunk
			accumulated := &domain.AccumulatedStreamChunk{
				CurrentMap:        currentMap,
				ProgressiveFields: make(map[string]*domain.ProgressiveValue),
				NewToken:          chunk,
				Progress:          0.5,
				IsFinal:           false,
			}
			out <- accumulated
		}
		// Send final chunk
		if lastChunk != nil {
			finalAccumulated := &domain.AccumulatedStreamChunk{
				CurrentMap:        currentMap,
				ProgressiveFields: make(map[string]*domain.ProgressiveValue),
				NewToken:          nil,
				Progress:          1.0,
				IsFinal:           true,
			}
			out <- finalAccumulated
		}
	}()
	return out, nil
}

type mockProgressiveExecutionStrategy struct {
	scheduleFunc func(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error)
	executeFunc  func(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error)
}

func (m *mockProgressiveExecutionStrategy) Schedule(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
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

func (m *mockProgressiveExecutionStrategy) Execute(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
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

func TestNewProgressiveGenerator(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

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

func TestProgressiveGenerate_Success(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Errors()) > 0 {
		t.Errorf("expected no errors, got %v", result.Errors())
	}
}

func TestProgressiveGenerate_PreProcessingError(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	// Mock a plugin that returns an error
	gen.plugins.Register(&mockProgressivePlugin{preProcessFunc: func(req *domain.GenerationRequest) (*domain.GenerationRequest, error) {
		return nil, errors.New("preprocessing error")
	}})

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Errors()) == 0 {
		t.Error("expected errors, got none")
	}
}

func TestProgressiveGenerate_AnalyzeError(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{
		analyzeFunc: func(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error) {
			return nil, errors.New("analyze error")
		},
	}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Errors()) == 0 {
		t.Error("expected errors, got none")
	}
}

func TestProgressiveGenerate_DetermineOrderError(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{
		determineProcessingOrderFunc: func(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error) {
			return nil, errors.New("determine order error")
		},
	}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Errors()) == 0 {
		t.Error("expected errors, got none")
	}
}

func TestProgressiveGenerate_ScheduleError(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{
		scheduleFunc: func(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
			return nil, errors.New("schedule error")
		},
	}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Errors()) == 0 {
		t.Error("expected errors, got none")
	}
}

func TestProgressiveGenerateStream_Success(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	stream, err := gen.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stream == nil {
		t.Fatal("expected stream, got nil")
	}

	// Collect chunks
	var chunks []*domain.StreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestProgressiveGenerateStreamProgressive_Success(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	stream, err := gen.GenerateStreamProgressive(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stream == nil {
		t.Fatal("expected stream, got nil")
	}

	// Collect chunks
	var chunks []*domain.AccumulatedStreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestProgressiveRegisterPlugin(t *testing.T) {
	analyzer := &mockProgressiveSchemaAnalyzer{}
	executor := &mockProgressiveTaskExecutor{}
	assembler := &mockProgressiveAssembler{}
	strategy := &mockProgressiveExecutionStrategy{}

	gen := NewProgressiveGenerator(analyzer, executor, assembler, strategy)

	plugin := &mockProgressivePlugin{}

	gen.RegisterPlugin(plugin)

	// Check if plugin is registered (assuming plugins has a way to check, but since it's internal, just ensure no panic)
}

// Mock plugin for testing
type mockProgressivePlugin struct {
	preProcessFunc func(req *domain.GenerationRequest) (*domain.GenerationRequest, error)
}

func (m *mockProgressivePlugin) Name() string {
	return "mock"
}

func (m *mockProgressivePlugin) Version() string {
	return "1.0"
}

func (m *mockProgressivePlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (m *mockProgressivePlugin) PreProcess(req *domain.GenerationRequest) (*domain.GenerationRequest, error) {
	if m.preProcessFunc != nil {
		return m.preProcessFunc(req)
	}
	return req, nil
}
