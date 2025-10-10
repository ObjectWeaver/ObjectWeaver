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

	"objectweaver/checks"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
	"objectweaver/service"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// GenerateObjectV2 - Uses V2 architecture for synchronous object generation
func (s *Server) GenerateObjectV2(ctx context.Context, req *pb.RequestBody) (*pb.Response, error) {
	// Convert protobuf request to internal format
	body := client.RequestBody{
		Prompt:     req.Prompt,
		Definition: converison.ConvertProtoToModel(req.Definition),
	}

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

	// Create response
	response := &pb.Response{
		Data:    toStruct,
		UsdCost: usdCost,
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
		// Use dependency-aware mode for optimal execution
		config.Mode = factory.ModeDependencyAware
	}

	// Configure based on schema properties
	config.MaxConcurrency = 10
	config.EnableCache = true

	return config
}
