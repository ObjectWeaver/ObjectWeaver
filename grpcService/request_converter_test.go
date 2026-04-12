package grpcService

import (
	"objectweaver/jsonSchema"
	"testing"

	pb "objectweaver/grpc"
)

func TestDefaultRequestConverter_Convert(t *testing.T) {
	converter := NewDefaultRequestConverter()

	tests := []struct {
		name     string
		req      *pb.RequestBody
		validate func(*testing.T, *jsonSchema.RequestBody)
	}{
		{
			name: "basic request with prompt",
			req: &pb.RequestBody{
				Prompt: "Test prompt",
				Definition: &pb.Definition{
					Type: "object",
				},
			},
			validate: func(t *testing.T, body *jsonSchema.RequestBody) {
				if body.Prompt != "Test prompt" {
					t.Errorf("Expected prompt 'Test prompt', got '%s'", body.Prompt)
				}
				if body.Definition == nil {
					t.Error("Expected definition to be non-nil")
				}
			},
		},
		{
			name: "request with nil definition",
			req: &pb.RequestBody{
				Prompt:     "Test",
				Definition: nil,
			},
			validate: func(t *testing.T, body *jsonSchema.RequestBody) {
				if body.Definition != nil {
					t.Error("Expected definition to be nil")
				}
			},
		},
		{
			name: "request with properties",
			req: &pb.RequestBody{
				Prompt: "Test",
				Definition: &pb.Definition{
					Type: "object",
					Properties: map[string]*pb.Definition{
						"name": {Type: "string"},
						"age":  {Type: "number"},
					},
				},
			},
			validate: func(t *testing.T, body *jsonSchema.RequestBody) {
				if body.Definition == nil {
					t.Fatal("Expected definition to be non-nil")
				}
				if body.Definition.Properties == nil {
					t.Error("Expected properties to be non-nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.req)
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			tt.validate(t, result)
		})
	}
}

func TestDefaultRequestConverter_ImplementsInterface(t *testing.T) {
	var _ RequestConverter = NewDefaultRequestConverter()
}
