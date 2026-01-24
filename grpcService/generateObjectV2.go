package grpcService

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	pb "github.com/objectweaver/go-sdk/grpc"

	"objectweaver/service"
)

// GenerateObjectV2 - Uses V2 architecture for synchronous object generation
func (s *Server) GenerateObjectV2(ctx context.Context, req *pb.RequestBody) (*pb.Response, error) {
	// Stage 1: Convert request
	body := s.requestConverter.Convert(req)

	// Validate that definition exists
	if body.Definition == nil {
		return nil, errors.New("invalid request: definition is required")
	}

	// Stage 2: Check for circular definitions
	if s.circularChecker.Check(body.Definition) {
		return nil, errors.New("circular definitions found")
	}

	// Stage 3: Create generator configuration
	config := s.configFactory.CreateConfig(body.Definition)

	// Stage 4: Create generator
	generator, err := s.generatorService.CreateGenerator(config)
	if err != nil {
		return nil, err
	}

	// Debug printing in development
	if os.Getenv("ENVIRONMENT") == "development" {
		print, _ := json.Marshal(body.Definition)
		service.PrettyPrintJSON(print)
	}

	// Stage 5: Generate the object
	result, err := s.generatorService.Generate(ctx, generator, body.Prompt, body.Definition)
	if err != nil {
		return nil, err
	}

	// Debug printing in development
	if os.Getenv("ENVIRONMENT") == "development" {
		bytes, _ := json.Marshal(result.Data())
		service.PrettyPrintJSON(bytes)
	}

	// Stage 6: Build response
	response, err := s.responseBuilder.BuildResponse(result)
	if err != nil {
		return nil, err
	}

	return response, nil
}
