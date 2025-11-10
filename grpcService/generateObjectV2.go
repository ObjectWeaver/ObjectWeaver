package grpcService

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/objectweaver/go-sdk/client"
	"github.com/objectweaver/go-sdk/converison"
	pb "github.com/objectweaver/go-sdk/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"objectweaver/checks"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
	"objectweaver/service"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// GenerateObjectV2 - Uses V2 architecture for synchronous object generation
func (s *Server) GenerateObjectV2(ctx context.Context, req *pb.RequestBody) (*pb.Response, error) {
	// Debug: Log the incoming protobuf to see if fields are present
	log.Printf("[GenerateObjectV2] Received request")
	if req.Definition != nil {
		log.Printf("[GenerateObjectV2] Definition has ScoringCriteria: %v", req.Definition.ScoringCriteria != nil)
		log.Printf("[GenerateObjectV2] Definition has DecisionPoint: %v", req.Definition.DecisionPoint != nil)

		// Check nested properties
		if req.Definition.Properties != nil {
			for key, prop := range req.Definition.Properties {
				log.Printf("[GenerateObjectV2] Property '%s' has ScoringCriteria: %v, DecisionPoint: %v",
					key, prop.ScoringCriteria != nil, prop.DecisionPoint != nil)
			}
		}
	}

	// Convert protobuf request to internal format
	body := client.RequestBody{
		Prompt:     req.Prompt,
		Definition: converison.ConvertProtoToModel(req.Definition),
	}

	// Debug: Check if conversion preserved the fields
	log.Printf("[GenerateObjectV2] After conversion - Definition has ScoringCriteria: %v, DecisionPoint: %v",
		body.Definition.ScoringCriteria != nil, body.Definition.DecisionPoint != nil)
	if body.Definition.Properties != nil {
		for key, prop := range body.Definition.Properties {
			log.Printf("[GenerateObjectV2] After conversion - Property '%s' has ScoringCriteria: %v, DecisionPoint: %v",
				key, prop.ScoringCriteria != nil, prop.DecisionPoint != nil)
		}
	}

	// TODO: Post-process to add missing fields if SDK conversion is incomplete
	// enrichDefinitionWithMissingFields(body.Definition, req.Definition)

	// Check for circular definitions
	if checks.CheckCircularDefinitions(body.Definition) {
		return nil, errors.New("circular definitions found")
	}

	// Determine processing mode based on schema configuration
	config := s.createGeneratorConfig(body.Definition)

	// Create generator using factory
	generatorFactory := factory.NewGeneratorFactory(config)
	generator, err := generatorFactory.Create()
	if err != nil {
		log.Printf("Failed to create generator: %v", err)
		return nil, err
	}

	// Create generation request
	print, _ := json.Marshal(body.Definition)
	service.PrettyPrintJSON(print)
	request := domain.NewGenerationRequest(body.Prompt, body.Definition).
		WithContext(ctx)

	// Generate the object
	result, err := generator.Generate(request)
	if err != nil {
		log.Printf("Generation failed: %v", err)
		return nil, err
	}

	if !result.IsSuccess() {
		return nil, errors.New("generation failed with errors")
	}

	// Get the generated data
	data := result.Data()

	// Marshal for debugging if in development
	if os.Getenv("ENVIRONMENT") == "development" {
		bytes, _ := json.Marshal(data)
		service.PrettyPrintJSON(bytes)
	}

	// Convert map to protobuf struct
	toStruct, err := converison.ConvertMapToStruct(data)
	if err != nil {
		log.Printf("Failed to convert to protobuf struct: %v", err)
		return nil, err
	}

	// Extract metadata
	metadata := result.Metadata()
	usdCost := 0.0
	if metadata != nil {
		usdCost = metadata.Cost
	}

	// Build detailed data with metadata if available
	var detailedDataMap map[string]*pb.DetailedField
	if result.HasDetailedData() {
		detailedDataMap = make(map[string]*pb.DetailedField)
		for key, fieldResult := range result.DetailedData() {
			// Convert field value to protobuf struct
			fieldValueStruct, err := converison.ConvertMapToStruct(map[string]interface{}{
				"value": fieldResult.Value,
			})
			if err != nil {
				log.Printf("Warning: Failed to convert field %s value to struct: %v", key, err)
				continue
			}

			// Extract actual value from struct
			var valueStruct *structpb.Struct
			if fieldValueStruct != nil && fieldValueStruct.Fields != nil && fieldValueStruct.Fields["value"] != nil {
				if val, ok := fieldValueStruct.Fields["value"].GetKind().(*structpb.Value_StructValue); ok {
					valueStruct = val.StructValue
				} else {
					// If it's not a struct, wrap the value itself
					valueStruct = &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"value": fieldValueStruct.Fields["value"],
						},
					}
				}
			}

			// Build field metadata
			var fieldMeta *pb.FieldMetadata
			if fieldResult.Metadata != nil {
				// Convert choices
				var choices []*pb.Choice
				for _, choice := range fieldResult.Metadata.Choices {
					// Convert choice.Completion (any) into a Struct
					choiceValueStruct, _ := converison.ConvertMapToStruct(map[string]interface{}{
						"value": choice.Completion,
					})

					choices = append(choices, &pb.Choice{
						Score:      int32(choice.Score),
						Confidence: choice.Confidence,
						Value:      choiceValueStruct,
						Embedding:  choice.Embedding,
					})
				}

				fieldMeta = &pb.FieldMetadata{
					TokensUsed: int32(fieldResult.Metadata.TokensUsed),
					Cost:       fieldResult.Metadata.Cost,
					ModelUsed:  fieldResult.Metadata.ModelUsed,
					Choices:    choices,
				}
			}

			detailedDataMap[key] = &pb.DetailedField{
				Value:    valueStruct,
				Metadata: fieldMeta,
			}
		}
	}

	// Create response
	response := &pb.Response{
		Data:         toStruct,
		UsdCost:      usdCost,
		DetailedData: detailedDataMap,
	}

	return response, nil
}

// createGeneratorConfig creates a generator config based on schema settings
func (s *Server) createGeneratorConfig(schema *jsonSchema.Definition) *factory.GeneratorConfig {
	config := factory.DefaultGeneratorConfig()

	// Check if streaming is enabled
	if schema.Stream {
		// For non-streaming gRPC calls, we still use sync mode
		// Streaming is handled by StreamGeneratedObjectsV2
		config.Mode = factory.ModeSync
	} else {
		// Use parallel mode (now uses recursive architecture internally)
		config.Mode = factory.ModeParallel
	}

	// Configure based on schema properties
	config.MaxConcurrency = 10
	config.EnableCache = true

	return config
}
