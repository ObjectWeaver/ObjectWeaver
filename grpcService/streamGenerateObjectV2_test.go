package grpcService

import (
	"context"
	"errors"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	pb "github.com/ObjectWeaver/ObjectWeaver/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/factory"
)

// Mock stream for testing
type mockStreamServer struct {
	sentResponses []*pb.StreamingResponse
	err           error
	ctx           context.Context
}

func (m *mockStreamServer) Send(response *pb.StreamingResponse) error {
	if m.err != nil {
		return m.err
	}
	m.sentResponses = append(m.sentResponses, response)
	return nil
}

func (m *mockStreamServer) SetHeader(metadata.MD) error {
	return nil
}

func (m *mockStreamServer) SendHeader(metadata.MD) error {
	return nil
}

func (m *mockStreamServer) SetTrailer(metadata.MD) {
}

func (m *mockStreamServer) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *mockStreamServer) SendMsg(interface{}) error {
	return nil
}

func (m *mockStreamServer) RecvMsg(interface{}) error {
	return nil
}

// Mock generator for streaming tests
type mockStreamGenerator struct {
	streamChan             chan *domain.StreamChunk
	progressiveChan        chan *domain.AccumulatedStreamChunk
	streamErr              error
	progressiveErr         error
	generateErr            error
	shouldCloseImmediately bool
}

func (m *mockStreamGenerator) Generate(request *domain.GenerationRequest) (*domain.GenerationResult, error) {
	return nil, m.generateErr
}

func (m *mockStreamGenerator) GenerateStream(request *domain.GenerationRequest) (<-chan *domain.StreamChunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}

	if m.shouldCloseImmediately {
		close(m.streamChan)
	}

	return m.streamChan, nil
}

func (m *mockStreamGenerator) GenerateStreamProgressive(request *domain.GenerationRequest) (<-chan *domain.AccumulatedStreamChunk, error) {
	if m.progressiveErr != nil {
		return nil, m.progressiveErr
	}

	if m.shouldCloseImmediately {
		close(m.progressiveChan)
	}

	return m.progressiveChan, nil
}

func TestServer_createStreamingConfig(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name                string
		schema              *jsonSchema.Definition
		expectedMode        factory.ProcessingMode
		expectedGranularity domain.StreamGranularity
	}{
		{
			name: "progressive streaming with stream enabled",
			schema: &jsonSchema.Definition{
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"field1": {
						Type:   jsonSchema.String,
						Stream: true,
					},
				},
			},
			expectedMode:        factory.ModeStreamingProgressive,
			expectedGranularity: domain.GranularityToken,
		},
		{
			name: "complete streaming without stream flag",
			schema: &jsonSchema.Definition{
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"field1": {
						Type:   jsonSchema.String,
						Stream: false,
					},
				},
			},
			expectedMode:        factory.ModeStreamingComplete,
			expectedGranularity: domain.GranularityField,
		},
		{
			name: "complete streaming with no properties",
			schema: &jsonSchema.Definition{
				Type: jsonSchema.String,
			},
			expectedMode:        factory.ModeStreamingComplete,
			expectedGranularity: domain.GranularityField,
		},
		{
			name: "progressive streaming with nested stream",
			schema: &jsonSchema.Definition{
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"field1": {
						Type:   jsonSchema.String,
						Stream: false,
					},
					"field2": {
						Type:   jsonSchema.String,
						Stream: true,
					},
				},
			},
			expectedMode:        factory.ModeStreamingProgressive,
			expectedGranularity: domain.GranularityToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := server.createStreamingConfig(tt.schema)

			if config.Mode != tt.expectedMode {
				t.Errorf("Expected mode %v, got %v", tt.expectedMode, config.Mode)
			}

			if config.Granularity != tt.expectedGranularity {
				t.Errorf("Expected granularity %v, got %v", tt.expectedGranularity, config.Granularity)
			}

			if config.MaxConcurrency != 10 {
				t.Errorf("Expected MaxConcurrency 10, got %d", config.MaxConcurrency)
			}

			if config.EnableCache {
				t.Error("Expected EnableCache to be false for streaming")
			}
		})
	}
}

func TestServer_shouldUseProgressiveStreaming(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name     string
		schema   *jsonSchema.Definition
		expected bool
	}{
		{
			name: "stream flag enabled",
			schema: &jsonSchema.Definition{
				Properties: map[string]jsonSchema.Definition{
					"field1": {Stream: true},
				},
			},
			expected: true,
		},
		{
			name: "stream flag disabled",
			schema: &jsonSchema.Definition{
				Properties: map[string]jsonSchema.Definition{
					"field1": {Stream: false},
				},
			},
			expected: false,
		},
		{
			name:     "no properties",
			schema:   &jsonSchema.Definition{},
			expected: false,
		},
		{
			name: "nil properties",
			schema: &jsonSchema.Definition{
				Properties: nil,
			},
			expected: false,
		},
		{
			name: "multiple fields, one with stream",
			schema: &jsonSchema.Definition{
				Properties: map[string]jsonSchema.Definition{
					"field1": {Stream: false},
					"field2": {Stream: true},
					"field3": {Stream: false},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.shouldUseProgressiveStreaming(tt.schema)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Note: Circular definition testing is covered in the checks package tests.
// Creating actual circular structures causes stack overflow in conversion.

func TestServer_streamComplete_Success(t *testing.T) {
	server := &Server{}

	// Create mock generator with stream channel
	streamChan := make(chan *domain.StreamChunk, 3)
	mockGen := &mockStreamGenerator{
		streamChan: streamChan,
	}

	// Send some chunks
	go func() {
		streamChan <- &domain.StreamChunk{
			Key:     "field1",
			Value:   "value1",
			IsFinal: false,
		}
		streamChan <- &domain.StreamChunk{
			Key:     "field2",
			Value:   "value2",
			IsFinal: false,
		}
		streamChan <- &domain.StreamChunk{
			Key:     "field3",
			Value:   "value3",
			IsFinal: true,
		}
		close(streamChan)
	}()

	mockStream := &mockStreamServer{
		ctx: context.Background(),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamComplete(mockGen, request, mockStream)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should have sent chunks + final response
	if len(mockStream.sentResponses) < 2 {
		t.Errorf("Expected at least 2 responses, got %d", len(mockStream.sentResponses))
	}

	// Check final response
	finalResp := mockStream.sentResponses[len(mockStream.sentResponses)-1]
	if finalResp.Status != "Completed" {
		t.Errorf("Expected final status 'Completed', got %s", finalResp.Status)
	}
}

func TestServer_streamComplete_GeneratorError(t *testing.T) {
	server := &Server{}

	mockGen := &mockStreamGenerator{
		streamErr: errors.New("generator failed"),
	}

	mockStream := &mockStreamServer{
		ctx: context.Background(),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamComplete(mockGen, request, mockStream)

	if err == nil {
		t.Error("Expected error from generator")
	}

	if err.Error() != "generator failed" {
		t.Errorf("Expected 'generator failed', got: %v", err)
	}
}

func TestServer_streamComplete_StreamSendError(t *testing.T) {
	server := &Server{}

	streamChan := make(chan *domain.StreamChunk, 1)
	mockGen := &mockStreamGenerator{
		streamChan: streamChan,
	}

	go func() {
		streamChan <- &domain.StreamChunk{
			Key:     "field1",
			Value:   "value1",
			IsFinal: false,
		}
		close(streamChan)
	}()

	mockStream := &mockStreamServer{
		ctx: context.Background(),
		err: errors.New("stream send failed"),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamComplete(mockGen, request, mockStream)

	if err == nil {
		t.Error("Expected stream send error")
	}
}

func TestServer_streamComplete_NilChunk(t *testing.T) {
	server := &Server{}

	streamChan := make(chan *domain.StreamChunk, 2)
	mockGen := &mockStreamGenerator{
		streamChan: streamChan,
	}

	go func() {
		streamChan <- nil // Should be skipped
		streamChan <- &domain.StreamChunk{
			Key:     "field1",
			Value:   "value1",
			IsFinal: true,
		}
		close(streamChan)
	}()

	mockStream := &mockStreamServer{
		ctx: context.Background(),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamComplete(mockGen, request, mockStream)

	if err != nil {
		t.Errorf("Expected no error with nil chunk, got: %v", err)
	}
}

func TestServer_streamProgressively_Success(t *testing.T) {
	server := &Server{}

	progressiveChan := make(chan *domain.AccumulatedStreamChunk, 3)
	mockGen := &mockStreamGenerator{
		progressiveChan: progressiveChan,
	}

	go func() {
		progressiveChan <- &domain.AccumulatedStreamChunk{
			NewToken: &domain.TokenStreamChunk{
				Key:      "field1",
				Partial:  "partial1",
				Complete: false,
			},
			IsFinal: false,
		}
		progressiveChan <- &domain.AccumulatedStreamChunk{
			NewToken: &domain.TokenStreamChunk{
				Key:      "field1",
				Partial:  "complete1",
				Complete: true,
			},
			IsFinal: false,
		}
		progressiveChan <- &domain.AccumulatedStreamChunk{
			IsFinal: true,
		}
		close(progressiveChan)
	}()

	mockStream := &mockStreamServer{
		ctx: context.Background(),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamProgressively(mockGen, request, mockStream)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(mockStream.sentResponses) < 2 {
		t.Errorf("Expected at least 2 responses, got %d", len(mockStream.sentResponses))
	}

	finalResp := mockStream.sentResponses[len(mockStream.sentResponses)-1]
	if finalResp.Status != "Completed" {
		t.Errorf("Expected final status 'Completed', got %s", finalResp.Status)
	}
}

func TestServer_streamProgressively_WithCurrentMap(t *testing.T) {
	server := &Server{}

	progressiveChan := make(chan *domain.AccumulatedStreamChunk, 2)
	mockGen := &mockStreamGenerator{
		progressiveChan: progressiveChan,
	}

	go func() {
		progressiveChan <- &domain.AccumulatedStreamChunk{
			CurrentMap: map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			},
			IsFinal: false,
		}
		progressiveChan <- &domain.AccumulatedStreamChunk{
			IsFinal: true,
		}
		close(progressiveChan)
	}()

	mockStream := &mockStreamServer{
		ctx: context.Background(),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamProgressively(mockGen, request, mockStream)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestServer_streamProgressively_GeneratorError(t *testing.T) {
	server := &Server{}

	mockGen := &mockStreamGenerator{
		progressiveErr: errors.New("progressive generation failed"),
	}

	mockStream := &mockStreamServer{
		ctx: context.Background(),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamProgressively(mockGen, request, mockStream)

	if err == nil {
		t.Error("Expected error from generator")
	}
}

func TestServer_streamProgressively_NilChunk(t *testing.T) {
	server := &Server{}

	progressiveChan := make(chan *domain.AccumulatedStreamChunk, 2)
	mockGen := &mockStreamGenerator{
		progressiveChan: progressiveChan,
	}

	go func() {
		progressiveChan <- nil // Should be skipped
		progressiveChan <- &domain.AccumulatedStreamChunk{
			IsFinal: true,
		}
		close(progressiveChan)
	}()

	mockStream := &mockStreamServer{
		ctx: context.Background(),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamProgressively(mockGen, request, mockStream)

	if err != nil {
		t.Errorf("Expected no error with nil chunk, got: %v", err)
	}
}

func TestServer_streamProgressively_StreamSendError(t *testing.T) {
	server := &Server{}

	progressiveChan := make(chan *domain.AccumulatedStreamChunk, 1)
	mockGen := &mockStreamGenerator{
		progressiveChan: progressiveChan,
	}

	go func() {
		progressiveChan <- &domain.AccumulatedStreamChunk{
			NewToken: &domain.TokenStreamChunk{
				Key:      "field1",
				Partial:  "test",
				Complete: false,
			},
			IsFinal: false,
		}
		close(progressiveChan)
	}()

	mockStream := &mockStreamServer{
		ctx: context.Background(),
		err: errors.New("stream send failed"),
	}

	def := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", def).WithContext(context.Background())

	err := server.streamProgressively(mockGen, request, mockStream)

	if err == nil {
		t.Error("Expected stream send error")
	}
}

func TestServer_createStreamingConfig_EdgeCases(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name   string
		schema *jsonSchema.Definition
	}{
		{
			name:   "empty schema",
			schema: &jsonSchema.Definition{},
		},
		{
			name: "empty properties map",
			schema: &jsonSchema.Definition{
				Properties: map[string]jsonSchema.Definition{},
			},
		},
		{
			name: "deeply nested without stream",
			schema: &jsonSchema.Definition{
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"level1": {
						Type: jsonSchema.Object,
						Properties: map[string]jsonSchema.Definition{
							"level2": {
								Type:   jsonSchema.String,
								Stream: false,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			config := server.createStreamingConfig(tt.schema)
			if config == nil {
				t.Error("Expected non-nil config")
			}
		})
	}
}
