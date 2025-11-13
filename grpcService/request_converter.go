package grpcService

import (
	"log"

	"github.com/objectweaver/go-sdk/client"
	"github.com/objectweaver/go-sdk/converison"
	pb "github.com/objectweaver/go-sdk/grpc"
)

// DefaultRequestConverter is the default implementation of RequestConverter
type DefaultRequestConverter struct{}

// NewDefaultRequestConverter creates a new DefaultRequestConverter
func NewDefaultRequestConverter() RequestConverter {
	return &DefaultRequestConverter{}
}

// Convert converts a protobuf request to internal request format
func (c *DefaultRequestConverter) Convert(req *pb.RequestBody) *client.RequestBody {
	// Debug: Log the incoming protobuf to see if fields are present
	log.Printf("[RequestConverter] Received request")
	if req.Definition != nil {
		log.Printf("[RequestConverter] Definition has ScoringCriteria: %v", req.Definition.ScoringCriteria != nil)
		log.Printf("[RequestConverter] Definition has DecisionPoint: %v", req.Definition.DecisionPoint != nil)

		// Check nested properties
		if req.Definition.Properties != nil {
			for key, prop := range req.Definition.Properties {
				log.Printf("[RequestConverter] Property '%s' has ScoringCriteria: %v, DecisionPoint: %v",
					key, prop.ScoringCriteria != nil, prop.DecisionPoint != nil)
			}
		}
	}

	// Convert protobuf request to internal format
	body := &client.RequestBody{
		Prompt:     req.Prompt,
		Definition: converison.ConvertProtoToModel(req.Definition),
	}

	// Debug: Check if conversion preserved the fields
	if body.Definition != nil {
		log.Printf("[RequestConverter] After conversion - Definition has ScoringCriteria: %v, DecisionPoint: %v",
			body.Definition.ScoringCriteria != nil, body.Definition.DecisionPoint != nil)
		if body.Definition.Properties != nil {
			for key, prop := range body.Definition.Properties {
				log.Printf("[RequestConverter] After conversion - Property '%s' has ScoringCriteria: %v, DecisionPoint: %v",
					key, prop.ScoringCriteria != nil, prop.DecisionPoint != nil)
			}
		}
	}

	// TODO: Post-process to add missing fields if SDK conversion is incomplete
	// enrichDefinitionWithMissingFields(body.Definition, req.Definition)

	return body
}
