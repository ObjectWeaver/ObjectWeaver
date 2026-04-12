package grpcService

import (
	"context"
	"errors"
	"testing"

	"objectweaver/jsonSchema"

	pb "objectweaver/grpc"

	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
)

// Mock implementations for testing

type mockRequestConverter struct {
	convertFunc func(*pb.RequestBody) *jsonSchema.RequestBody
}

func (m *mockRequestConverter) Convert(req *pb.RequestBody) *jsonSchema.RequestBody {
	if m.convertFunc != nil {
		return m.convertFunc(req)
	}
	return &jsonSchema.RequestBody{
		Prompt:     req.Prompt,
		Definition: &jsonSchema.Definition{Type: "object"},
	}
}

type mockCircularChecker struct {
	checkFunc func(*jsonSchema.Definition) bool
}

func (m *mockCircularChecker) Check(definition *jsonSchema.Definition) bool {
	if m.checkFunc != nil {
		return m.checkFunc(definition)
	}
	return false
}

type mockConfigFactory struct {
	createConfigFunc func(*jsonSchema.Definition) *factory.GeneratorConfig
}

func (m *mockConfigFactory) CreateConfig(schema *jsonSchema.Definition) *factory.GeneratorConfig {
	if m.createConfigFunc != nil {
		return m.createConfigFunc(schema)
	}
	return factory.DefaultGeneratorConfig()
}

type mockGeneratorService struct {
	createGeneratorFunc func(*factory.GeneratorConfig) (domain.Generator, error)
	generateFunc        func(context.Context, domain.Generator, string, *jsonSchema.Definition) (*domain.GenerationResult, error)
}

func (m *mockGeneratorService) CreateGenerator(config *factory.GeneratorConfig) (domain.Generator, error) {
	if m.createGeneratorFunc != nil {
		return m.createGeneratorFunc(config)
	}
	return &mockGenerator{}, nil
}

func (m *mockGeneratorService) Generate(ctx context.Context, generator domain.Generator, prompt string, definition *jsonSchema.Definition) (*domain.GenerationResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, generator, prompt, definition)
	}
	metadata := domain.NewResultMetadata()
	metadata.Cost = 0.001
	return domain.NewGenerationResult(
		map[string]interface{}{"test": "data"},
		metadata,
	), nil
}

type mockGenerator struct{}

func (m *mockGenerator) Generate(request *domain.GenerationRequest) (*domain.GenerationResult, error) {
	metadata := domain.NewResultMetadata()
	metadata.Cost = 0.001
	return domain.NewGenerationResult(
		map[string]interface{}{"test": "data"},
		metadata,
	), nil
}

func (m *mockGenerator) GenerateStream(request *domain.GenerationRequest) (<-chan *domain.StreamChunk, error) {
	ch := make(chan *domain.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockGenerator) GenerateStreamProgressive(request *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	ch := make(chan *domain.AccumulatedStreamChunk)
	close(ch)
	return ch, nil
}

type mockResponseBuilder struct {
	buildResponseFunc func(*domain.GenerationResult) (*pb.Response, error)
}

func (m *mockResponseBuilder) BuildResponse(result *domain.GenerationResult) (*pb.Response, error) {
	if m.buildResponseFunc != nil {
		return m.buildResponseFunc(result)
	}
	return &pb.Response{UsdCost: 0.001}, nil
}

// Tests

func TestServer_GenerateObjectV2_Success(t *testing.T) {
	server := NewServer(
		&mockRequestConverter{},
		&mockCircularChecker{},
		&mockConfigFactory{},
		&mockGeneratorService{},
		&mockResponseBuilder{},
	)

	req := &pb.RequestBody{
		Prompt: "Test prompt",
		Definition: &pb.Definition{
			Type: "object",
		},
	}

	ctx := context.Background()
	response, err := server.GenerateObjectV2(ctx, req)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if response == nil {
		t.Error("Expected non-nil response")
	}
}

func TestServer_GenerateObjectV2_CircularDefinitionError(t *testing.T) {
	server := NewServer(
		&mockRequestConverter{},
		&mockCircularChecker{
			checkFunc: func(*jsonSchema.Definition) bool {
				return true // simulate circular definition
			},
		},
		&mockConfigFactory{},
		&mockGeneratorService{},
		&mockResponseBuilder{},
	)

	req := &pb.RequestBody{
		Prompt: "Test prompt",
		Definition: &pb.Definition{
			Type: "object",
		},
	}

	ctx := context.Background()
	response, err := server.GenerateObjectV2(ctx, req)

	if err == nil {
		t.Error("Expected error for circular definition")
	}
	if response != nil {
		t.Error("Expected nil response for circular definition")
	}
	if err != nil && err.Error() != "circular definitions found" {
		t.Errorf("Expected 'circular definitions found' error, got %v", err)
	}
}

func TestServer_GenerateObjectV2_GeneratorCreationError(t *testing.T) {
	server := NewServer(
		&mockRequestConverter{},
		&mockCircularChecker{},
		&mockConfigFactory{},
		&mockGeneratorService{
			createGeneratorFunc: func(*factory.GeneratorConfig) (domain.Generator, error) {
				return nil, errors.New("generator creation failed")
			},
		},
		&mockResponseBuilder{},
	)

	req := &pb.RequestBody{
		Prompt: "Test prompt",
		Definition: &pb.Definition{
			Type: "object",
		},
	}

	ctx := context.Background()
	response, err := server.GenerateObjectV2(ctx, req)

	if err == nil {
		t.Error("Expected error for generator creation failure")
	}
	if response != nil {
		t.Error("Expected nil response for generator creation failure")
	}
}

func TestServer_GenerateObjectV2_GenerationError(t *testing.T) {
	server := NewServer(
		&mockRequestConverter{},
		&mockCircularChecker{},
		&mockConfigFactory{},
		&mockGeneratorService{
			generateFunc: func(context.Context, domain.Generator, string, *jsonSchema.Definition) (*domain.GenerationResult, error) {
				return nil, errors.New("generation failed")
			},
		},
		&mockResponseBuilder{},
	)

	req := &pb.RequestBody{
		Prompt: "Test prompt",
		Definition: &pb.Definition{
			Type: "object",
		},
	}

	ctx := context.Background()
	response, err := server.GenerateObjectV2(ctx, req)

	if err == nil {
		t.Error("Expected error for generation failure")
	}
	if response != nil {
		t.Error("Expected nil response for generation failure")
	}
}

func TestServer_GenerateObjectV2_ResponseBuildError(t *testing.T) {
	server := NewServer(
		&mockRequestConverter{},
		&mockCircularChecker{},
		&mockConfigFactory{},
		&mockGeneratorService{},
		&mockResponseBuilder{
			buildResponseFunc: func(*domain.GenerationResult) (*pb.Response, error) {
				return nil, errors.New("response build failed")
			},
		},
	)

	req := &pb.RequestBody{
		Prompt: "Test prompt",
		Definition: &pb.Definition{
			Type: "object",
		},
	}

	ctx := context.Background()
	response, err := server.GenerateObjectV2(ctx, req)

	if err == nil {
		t.Error("Expected error for response build failure")
	}
	if response != nil {
		t.Error("Expected nil response for response build failure")
	}
}

func TestNewDefaultServer(t *testing.T) {
	server := NewDefaultServer()

	if server == nil {
		t.Fatal("Expected non-nil server")
	}
	if server.requestConverter == nil {
		t.Error("Expected non-nil requestConverter")
	}
	if server.circularChecker == nil {
		t.Error("Expected non-nil circularChecker")
	}
	if server.configFactory == nil {
		t.Error("Expected non-nil configFactory")
	}
	if server.generatorService == nil {
		t.Error("Expected non-nil generatorService")
	}
	if server.responseBuilder == nil {
		t.Error("Expected non-nil responseBuilder")
	}
}

func TestNewServer(t *testing.T) {
	converter := &mockRequestConverter{}
	checker := &mockCircularChecker{}
	configFactory := &mockConfigFactory{}
	genService := &mockGeneratorService{}
	respBuilder := &mockResponseBuilder{}

	server := NewServer(converter, checker, configFactory, genService, respBuilder)

	if server == nil {
		t.Fatal("Expected non-nil server")
	}
	if server.requestConverter != converter {
		t.Error("Expected injected requestConverter")
	}
	if server.circularChecker != checker {
		t.Error("Expected injected circularChecker")
	}
	if server.configFactory != configFactory {
		t.Error("Expected injected configFactory")
	}
	if server.generatorService != genService {
		t.Error("Expected injected generatorService")
	}
	if server.responseBuilder != respBuilder {
		t.Error("Expected injected responseBuilder")
	}
}
