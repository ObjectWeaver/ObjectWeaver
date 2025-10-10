package application

import (
	"errors"
	"testing"

	"firechimp/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

// Mock implementations for testing StreamingGenerator

type mockStreamingSchemaAnalyzer struct {
	analyzeFunc                  func(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error)
	determineProcessingOrderFunc func(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error)
}

func (m *mockStreamingSchemaAnalyzer) Analyze(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(schema)
	}
	return &domain.SchemaAnalysis{Fields: []*domain.FieldDefinition{
		{Key: "testField", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
	}}, nil
}

func (m *mockStreamingSchemaAnalyzer) ExtractFields(schema *jsonSchema.Definition) ([]*domain.FieldDefinition, error) {
	return []*domain.FieldDefinition{
		{Key: "testField", Definition: &jsonSchema.Definition{Type: jsonSchema.String}},
	}, nil
}

func (m *mockStreamingSchemaAnalyzer) DetermineProcessingOrder(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error) {
	if m.determineProcessingOrderFunc != nil {
		return m.determineProcessingOrderFunc(fields)
	}
	tasks := make([]*domain.FieldTask, len(fields))
	for i, field := range fields {
		tasks[i] = domain.NewFieldTask(field.Key, field.Definition, nil)
	}
	return tasks, nil
}

type mockStreamingTaskExecutor struct{}

func (m *mockStreamingTaskExecutor) Execute(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	return domain.NewTaskResult(task.ID(), task.Key(), "mock_value", domain.NewResultMetadata()), nil
}

func (m *mockStreamingTaskExecutor) ExecuteBatch(tasks []*domain.FieldTask, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, len(tasks))
	for i, task := range tasks {
		results[i] = domain.NewTaskResult(task.ID(), task.Key(), "mock_value", domain.NewResultMetadata())
	}
	return results, nil
}

type mockStreamingAssembler struct {
	assembleStreamingFunc func(results <-chan *domain.TaskResult) (<-chan *domain.StreamChunk, error)
}

func (m *mockStreamingAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	data := make(map[string]interface{})
	for _, result := range results {
		data[result.Key()] = result.Value()
	}
	return domain.NewGenerationResult(data, domain.NewResultMetadata()), nil
}

func (m *mockStreamingAssembler) AssembleStreaming(results <-chan *domain.TaskResult) (<-chan *domain.StreamChunk, error) {
	if m.assembleStreamingFunc != nil {
		return m.assembleStreamingFunc(results)
	}
	// Default mock implementation
	out := make(chan *domain.StreamChunk, 100)
	go func() {
		defer close(out)
		for result := range results {
			chunk := domain.NewStreamChunk(result.Key(), result.Value().(string))
			out <- chunk
		}
		// Send final chunk
		finalChunk := &domain.StreamChunk{
			Key:     "",
			Value:   nil,
			IsFinal: true,
		}
		out <- finalChunk
	}()
	return out, nil
}

type mockStreamingExecutionStrategy struct {
	scheduleFunc func(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error)
	executeFunc  func(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error)
}

func (m *mockStreamingExecutionStrategy) Schedule(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
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

func (m *mockStreamingExecutionStrategy) Execute(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
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

func TestNewStreamingGenerator(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

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

func TestStreamingGenerate_Success(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

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
	if result.Data()["testField"] != "mock_value" {
		t.Errorf("expected testField to be mock_value, got %v", result.Data()["testField"])
	}
}

func TestStreamingGenerate_PreProcessingError(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

	// Mock a plugin that returns an error
	gen.plugins.Register(&mockStreamingPlugin{preProcessFunc: func(req *domain.GenerationRequest) (*domain.GenerationRequest, error) {
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
	// Since error in goroutine, stream is empty, so data is empty
	if len(result.Data()) != 0 {
		t.Errorf("expected empty data on error, got %v", result.Data())
	}
}

func TestStreamingGenerate_AnalyzeError(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{
		analyzeFunc: func(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error) {
			return nil, errors.New("analyze error")
		},
	}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Data()) != 0 {
		t.Errorf("expected empty data on error, got %v", result.Data())
	}
}

func TestStreamingGenerate_DetermineOrderError(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{
		determineProcessingOrderFunc: func(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error) {
			return nil, errors.New("determine order error")
		},
	}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Data()) != 0 {
		t.Errorf("expected empty data on error, got %v", result.Data())
	}
}

func TestStreamingGenerate_ScheduleError(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{
		scheduleFunc: func(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
			return nil, errors.New("schedule error")
		},
	}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	result, err := gen.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Data()) != 0 {
		t.Errorf("expected empty data on error, got %v", result.Data())
	}
}

func TestStreamingGenerateStream_Success(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

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
	// Check that we have a non-final chunk and a final chunk
	hasNonFinal := false
	hasFinal := false
	for _, chunk := range chunks {
		if chunk.IsFinal {
			hasFinal = true
		} else {
			hasNonFinal = true
		}
	}
	if !hasNonFinal {
		t.Error("expected at least one non-final chunk")
	}
	if !hasFinal {
		t.Error("expected a final chunk")
	}
}

func TestStreamingGenerateStreamProgressive_NotSupported(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

	schema := &jsonSchema.Definition{Type: jsonSchema.Object}
	request := domain.NewGenerationRequest("test prompt", schema)

	stream, err := gen.GenerateStreamProgressive(request)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if stream != nil {
		t.Error("expected nil stream when not supported")
	}
}

func TestStreamingRegisterPlugin(t *testing.T) {
	analyzer := &mockStreamingSchemaAnalyzer{}
	executor := &mockStreamingTaskExecutor{}
	assembler := &mockStreamingAssembler{}
	strategy := &mockStreamingExecutionStrategy{}

	gen := NewStreamingGenerator(analyzer, executor, assembler, strategy)

	plugin := &mockStreamingPlugin{}

	gen.RegisterPlugin(plugin)

	// Check if plugin is registered (assuming plugins has a way to check, but since it's internal, just ensure no panic)
}

// Mock plugin for testing
type mockStreamingPlugin struct {
	preProcessFunc func(req *domain.GenerationRequest) (*domain.GenerationRequest, error)
}

func (m *mockStreamingPlugin) Name() string {
	return "mock"
}

func (m *mockStreamingPlugin) Version() string {
	return "1.0"
}

func (m *mockStreamingPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (m *mockStreamingPlugin) PreProcess(req *domain.GenerationRequest) (*domain.GenerationRequest, error) {
	if m.preProcessFunc != nil {
		return m.preProcessFunc(req)
	}
	return req, nil
}
