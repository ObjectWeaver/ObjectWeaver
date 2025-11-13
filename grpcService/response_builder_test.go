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

func TestDefaultResponseBuilder_BuildResponse_WithDetailedData(t *testing.T) {
	builder := NewDefaultResponseBuilder()

	tests := []struct {
		name           string
		result         *domain.GenerationResult
		wantErr        bool
		validateResult func(t *testing.T, result interface{})
	}{
		{
			name: "detailed data with simple field value (non-struct)",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.001
				metadata.TokensUsed = 50

				fieldMeta := domain.NewResultMetadata()
				fieldMeta.Cost = 0.0005
				fieldMeta.TokensUsed = 25
				fieldMeta.ModelUsed = "gpt-4"

				detailedData := map[string]*domain.FieldResultWithMetadata{
					"name": domain.NewFieldResultWithMetadata("John Doe", fieldMeta),
				}

				return domain.NewGenerationResultWithDetailedData(
					map[string]interface{}{
						"name": "John Doe",
					},
					detailedData,
					metadata,
				)
			}(),
			wantErr: false,
			validateResult: func(t *testing.T, result interface{}) {
				// This test covers the non-struct value wrapping case
			},
		},
		{
			name: "detailed data with struct field value",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.002

				fieldMeta := domain.NewResultMetadata()
				fieldMeta.Cost = 0.001
				fieldMeta.TokensUsed = 30
				fieldMeta.ModelUsed = "gpt-3.5-turbo"

				detailedData := map[string]*domain.FieldResultWithMetadata{
					"address": domain.NewFieldResultWithMetadata(
						map[string]interface{}{
							"street": "123 Main St",
							"city":   "Boston",
						},
						fieldMeta,
					),
				}

				return domain.NewGenerationResultWithDetailedData(
					map[string]interface{}{
						"address": map[string]interface{}{
							"street": "123 Main St",
							"city":   "Boston",
						},
					},
					detailedData,
					metadata,
				)
			}(),
			wantErr: false,
		},
		{
			name: "detailed data with field metadata and choices",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.003

				fieldMeta := domain.NewResultMetadata()
				fieldMeta.Cost = 0.0015
				fieldMeta.TokensUsed = 40
				fieldMeta.ModelUsed = "claude-3"
				fieldMeta.Choices = []domain.Choice{
					{
						Score:      95,
						Confidence: 0.95,
						Completion: "software engineer",
						Embedding:  []float64{0.1, 0.2, 0.3},
					},
					{
						Score:      87,
						Confidence: 0.87,
						Completion: "developer",
						Embedding:  []float64{0.15, 0.25, 0.35},
					},
				}

				detailedData := map[string]*domain.FieldResultWithMetadata{
					"profession": domain.NewFieldResultWithMetadata("software engineer", fieldMeta),
				}

				return domain.NewGenerationResultWithDetailedData(
					map[string]interface{}{
						"profession": "software engineer",
					},
					detailedData,
					metadata,
				)
			}(),
			wantErr: false,
		},
		{
			name: "detailed data with multiple fields and complex choices",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.005

				nameMeta := domain.NewResultMetadata()
				nameMeta.Cost = 0.002
				nameMeta.TokensUsed = 20
				nameMeta.ModelUsed = "gpt-4"

				ageMeta := domain.NewResultMetadata()
				ageMeta.Cost = 0.001
				ageMeta.TokensUsed = 15
				ageMeta.ModelUsed = "gpt-3.5-turbo"
				ageMeta.Choices = []domain.Choice{
					{
						Score:      90,
						Confidence: 0.9,
						Completion: 30,
						Embedding:  []float64{0.5, 0.6},
					},
				}

				locationMeta := domain.NewResultMetadata()
				locationMeta.Cost = 0.002
				locationMeta.TokensUsed = 25
				locationMeta.ModelUsed = "claude-3"
				locationMeta.Choices = []domain.Choice{
					{
						Score:      85,
						Confidence: 0.85,
						Completion: map[string]interface{}{
							"city":    "New York",
							"country": "USA",
						},
						Embedding: []float64{0.7, 0.8, 0.9},
					},
					{
						Score:      78,
						Confidence: 0.78,
						Completion: map[string]interface{}{
							"city":    "Boston",
							"country": "USA",
						},
						Embedding: []float64{0.65, 0.75, 0.85},
					},
				}

				detailedData := map[string]*domain.FieldResultWithMetadata{
					"name": domain.NewFieldResultWithMetadata("Alice", nameMeta),
					"age":  domain.NewFieldResultWithMetadata(30, ageMeta),
					"location": domain.NewFieldResultWithMetadata(
						map[string]interface{}{
							"city":    "New York",
							"country": "USA",
						},
						locationMeta,
					),
				}

				return domain.NewGenerationResultWithDetailedData(
					map[string]interface{}{
						"name": "Alice",
						"age":  30,
						"location": map[string]interface{}{
							"city":    "New York",
							"country": "USA",
						},
					},
					detailedData,
					metadata,
				)
			}(),
			wantErr: false,
		},
		{
			name: "detailed data without metadata on field",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.001

				detailedData := map[string]*domain.FieldResultWithMetadata{
					"email": domain.NewFieldResultWithMetadata("test@example.com", nil),
				}

				return domain.NewGenerationResultWithDetailedData(
					map[string]interface{}{
						"email": "test@example.com",
					},
					detailedData,
					metadata,
				)
			}(),
			wantErr: false,
		},
		{
			name: "detailed data with empty choices array",
			result: func() *domain.GenerationResult {
				metadata := domain.NewResultMetadata()
				metadata.Cost = 0.001

				fieldMeta := domain.NewResultMetadata()
				fieldMeta.Cost = 0.0005
				fieldMeta.TokensUsed = 10
				fieldMeta.ModelUsed = "gpt-4"
				fieldMeta.Choices = []domain.Choice{} // Empty choices

				detailedData := map[string]*domain.FieldResultWithMetadata{
					"status": domain.NewFieldResultWithMetadata("active", fieldMeta),
				}

				return domain.NewGenerationResultWithDetailedData(
					map[string]interface{}{
						"status": "active",
					},
					detailedData,
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
				return
			}
			if response.Data == nil {
				t.Error("Expected response to have data")
			}

			// Validate detailed data is present
			if response.DetailedData == nil {
				t.Error("Expected response to have detailed data")
				return
			}

			// Validate field count matches
			if len(response.DetailedData) != len(tt.result.DetailedData()) {
				t.Errorf("Expected %d detailed fields, got %d", len(tt.result.DetailedData()), len(response.DetailedData))
			}

			// Additional validation if provided
			if tt.validateResult != nil {
				tt.validateResult(t, response)
			}
		})
	}
}

func TestDefaultResponseBuilder_BuildResponse_ConversionError(t *testing.T) {
	builder := NewDefaultResponseBuilder()

	t.Run("error converting main data to struct", func(t *testing.T) {
		// Create a result with data that might cause conversion issues
		// Using a channel or function, which cannot be converted to protobuf
		metadata := domain.NewResultMetadata()
		metadata.Cost = 0.001

		// Channels and functions cannot be converted to protobuf structs
		result := domain.NewGenerationResult(
			map[string]interface{}{
				"channel": make(chan int),
			},
			metadata,
		)

		response, err := builder.BuildResponse(result)
		if err == nil {
			t.Error("Expected error when converting unconvertible types")
		}
		if response != nil {
			t.Error("Expected nil response on conversion error")
		}
	})

	t.Run("error converting field value in detailed data", func(t *testing.T) {
		// Create a result where field value conversion will fail
		metadata := domain.NewResultMetadata()
		fieldMeta := domain.NewResultMetadata()
		fieldMeta.TokensUsed = 10

		// This field has a channel value that cannot be converted
		detailedData := map[string]*domain.FieldResultWithMetadata{
			"validField": domain.NewFieldResultWithMetadata("valid value", fieldMeta),
			"invalidField": domain.NewFieldResultWithMetadata(
				make(chan int), // This will fail conversion
				fieldMeta,
			),
		}

		result := domain.NewGenerationResultWithDetailedData(
			map[string]interface{}{
				"validField":   "valid value",
				"invalidField": "fallback", // Main data is valid
			},
			detailedData,
			metadata,
		)

		response, err := builder.BuildResponse(result)
		// The function should continue despite field conversion errors
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if response == nil {
			t.Error("Expected non-nil response")
			return
		}
		// Should have detailed data but missing the invalid field
		if response.DetailedData == nil {
			t.Error("Expected detailed data")
			return
		}
		// The valid field should be present
		if _, exists := response.DetailedData["validField"]; !exists {
			t.Error("Expected validField to be present in detailed data")
		}
		// The invalid field should be skipped (continue was called)
		if _, exists := response.DetailedData["invalidField"]; exists {
			t.Error("Expected invalidField to be skipped due to conversion error")
		}
	})
}

func TestDefaultResponseBuilder_BuildResponse_WithDetailedData_EdgeCases(t *testing.T) {
	builder := NewDefaultResponseBuilder()

	t.Run("detailed data with nil field value struct", func(t *testing.T) {
		metadata := domain.NewResultMetadata()
		fieldMeta := domain.NewResultMetadata()

		detailedData := map[string]*domain.FieldResultWithMetadata{
			"nullField": domain.NewFieldResultWithMetadata(nil, fieldMeta),
		}

		result := domain.NewGenerationResultWithDetailedData(
			map[string]interface{}{
				"nullField": nil,
			},
			detailedData,
			metadata,
		)

		response, err := builder.BuildResponse(result)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if response == nil {
			t.Error("Expected non-nil response")
		}
	})

	t.Run("detailed data with complex nested structures", func(t *testing.T) {
		metadata := domain.NewResultMetadata()
		fieldMeta := domain.NewResultMetadata()
		fieldMeta.ModelUsed = "gpt-4-turbo"
		fieldMeta.TokensUsed = 100

		complexValue := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": "deep value",
					"array":  []interface{}{1, 2, 3},
				},
			},
		}

		detailedData := map[string]*domain.FieldResultWithMetadata{
			"complex": domain.NewFieldResultWithMetadata(complexValue, fieldMeta),
		}

		result := domain.NewGenerationResultWithDetailedData(
			map[string]interface{}{
				"complex": complexValue,
			},
			detailedData,
			metadata,
		)

		response, err := builder.BuildResponse(result)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if response == nil {
			t.Error("Expected non-nil response")
			return
		}
		if response.DetailedData == nil || response.DetailedData["complex"] == nil {
			t.Error("Expected detailed data for complex field")
		}
	})

	t.Run("detailed data with choices having various completion types", func(t *testing.T) {
		metadata := domain.NewResultMetadata()
		fieldMeta := domain.NewResultMetadata()
		fieldMeta.Choices = []domain.Choice{
			{
				Score:      100,
				Confidence: 1.0,
				Completion: "string completion",
			},
			{
				Score:      95,
				Confidence: 0.95,
				Completion: 42,
			},
			{
				Score:      90,
				Confidence: 0.9,
				Completion: []interface{}{"a", "b", "c"},
			},
			{
				Score:      85,
				Confidence: 0.85,
				Completion: map[string]interface{}{"key": "value"},
			},
		}

		detailedData := map[string]*domain.FieldResultWithMetadata{
			"mixedTypes": domain.NewFieldResultWithMetadata("result", fieldMeta),
		}

		result := domain.NewGenerationResultWithDetailedData(
			map[string]interface{}{
				"mixedTypes": "result",
			},
			detailedData,
			metadata,
		)

		response, err := builder.BuildResponse(result)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		// Validate choices are present
		detailedField := response.DetailedData["mixedTypes"]
		if detailedField == nil {
			t.Error("Expected detailed field for mixedTypes")
			return
		}
		if detailedField.Metadata == nil {
			t.Error("Expected field metadata")
			return
		}
		if len(detailedField.Metadata.Choices) != 4 {
			t.Errorf("Expected 4 choices, got %d", len(detailedField.Metadata.Choices))
		}
	})
}
