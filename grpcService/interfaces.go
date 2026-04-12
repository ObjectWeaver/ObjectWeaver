package grpcService

import (
	"context"
	"objectweaver/jsonSchema"

	pb "objectweaver/grpc"

	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
)

// RequestConverter converts protobuf requests to internal request format
type RequestConverter interface {
	Convert(req *pb.RequestBody) *jsonSchema.RequestBody
}

// CircularDefinitionChecker checks for circular definitions in schemas
type CircularDefinitionChecker interface {
	Check(definition *jsonSchema.Definition) bool
}

// ConfigFactory creates generator configurations based on schema
type ConfigFactory interface {
	CreateConfig(schema *jsonSchema.Definition) *factory.GeneratorConfig
}

// GeneratorService creates and manages generators
type GeneratorService interface {
	CreateGenerator(config *factory.GeneratorConfig) (domain.Generator, error)
	Generate(ctx context.Context, generator domain.Generator, prompt string, definition *jsonSchema.Definition) (*domain.GenerationResult, error)
}

// ResponseBuilder builds protobuf responses from generation results
type ResponseBuilder interface {
	BuildResponse(result *domain.GenerationResult) (*pb.Response, error)
}
