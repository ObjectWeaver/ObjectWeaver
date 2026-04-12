package grpcService

import (
	"errors"
	"objectweaver/logger"

	"objectweaver/converison"
	pb "objectweaver/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"objectweaver/orchestration/jos/domain"
)

// DefaultResponseBuilder is the default implementation of ResponseBuilder
type DefaultResponseBuilder struct{}

// NewDefaultResponseBuilder creates a new DefaultResponseBuilder
func NewDefaultResponseBuilder() ResponseBuilder {
	return &DefaultResponseBuilder{}
}

// BuildResponse builds a protobuf response from a generation result
func (b *DefaultResponseBuilder) BuildResponse(result *domain.GenerationResult) (*pb.Response, error) {
	if !result.IsSuccess() {
		return nil, errors.New("generation failed with errors")
	}

	// Get the generated data
	data := result.Data()

	// Convert map to protobuf struct
	toStruct, err := converison.ConvertMapToStruct(data)
	if err != nil {
		logger.Printf("Failed to convert to protobuf struct: %v", err)
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
				logger.Printf("Warning: Failed to convert field %s value to struct: %v", key, err)
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
