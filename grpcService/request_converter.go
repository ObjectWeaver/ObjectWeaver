package grpcService

import (
	"github.com/ObjectWeaver/ObjectWeaver/converison"
	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
	"github.com/ObjectWeaver/ObjectWeaver/logger"

	pb "github.com/ObjectWeaver/ObjectWeaver/grpc"
)

// DefaultRequestConverter is the default implementation of RequestConverter
type DefaultRequestConverter struct{}

// NewDefaultRequestConverter creates a new DefaultRequestConverter
func NewDefaultRequestConverter() RequestConverter {
	return &DefaultRequestConverter{}
}

// Convert converts a protobuf request to internal request format
func (c *DefaultRequestConverter) Convert(req *pb.RequestBody) *jsonSchema.RequestBody {
	if req.Definition != nil {
		// Check nested properties
		if req.Definition.Properties != nil {
			for key, prop := range req.Definition.Properties {
				logger.Printf("[RequestConverter] Property '%s' has ScoringCriteria: %v, DecisionPoint: %v",
					key, prop.ScoringCriteria != nil, prop.DecisionPoint != nil)
			}
		}
	}

	// Convert protobuf request to internal format
	body := &jsonSchema.RequestBody{
		Prompt:     req.Prompt,
		Definition: converison.ConvertProtoToModel(req.Definition),
	}

	// Debug: Check if conversion preserved the fields
	if body.Definition != nil {
		if body.Definition.Properties != nil {
			for key, prop := range body.Definition.Properties {
				logger.Printf("[RequestConverter] After conversion - Property '%s' has ScoringCriteria: %v, DecisionPoint: %v",
					key, prop.ScoringCriteria != nil, prop.DecisionPoint != nil)
			}
		}
	}

	// TODO: Post-process to add missing fields if SDK conversion is incomplete
	// enrichDefinitionWithMissingFields(body.Definition, req.Definition)

	return body
}
