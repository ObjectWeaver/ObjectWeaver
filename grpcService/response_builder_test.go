package grpcService

import (
	"errors"
	"testing"

	"objectweaver/orchestration/jos/domain"
)

func TestDefaultResponseBuilder_BuildResponse(t *testing.T) {
	builder := NewDefaultResponseBuilder()

	tests := []struct {
		name    string
		result  *domain.GenerationResult
		wantErr bool
	}{
		{
			name: "successful result with data",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.001
				metadata.TokensUsed = 50
				return domain.NewGenerationResult(
					map[string]interface{}{
						"name": "John",
						"age":  30,
					},
					metadata,
				)
			}(),
			wantErr: false,
		},
		{
			name: "successful result with empty data",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.0
				return domain.NewGenerationResult(
					map[string]interface{}{},
					metadata,
				)
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := builder.BuildResponse(tt.result)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if response == nil {
				t.Error("Expected non-nil response")
			}
			if response != nil && response.Data == nil {
				t.Error("Expected response to have data")
			}
		})
	}
}

func TestDefaultResponseBuilder_BuildResponse_FailedResult(t *testing.T) {
	builder := NewDefaultResponseBuilder()

	// Create a failed result using the error constructor
	result := domain.NewGenerationResultWithError(errors.New("test error"))

	response, err := builder.BuildResponse(result)
	if err == nil {
		t.Error("Expected error for failed result")
	}
	if response != nil {
		t.Error("Expected nil response for failed result")
	}
}

func TestDefaultResponseBuilder_ImplementsInterface(t *testing.T) {
	var _ ResponseBuilder = NewDefaultResponseBuilder()
}
