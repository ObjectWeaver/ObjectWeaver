package grpcService

import (
	"context"

	"objectweaver/jsonSchema"

	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
)

// DefaultGeneratorService is the default implementation of GeneratorService
type DefaultGeneratorService struct{}

// NewDefaultGeneratorService creates a new DefaultGeneratorService
func NewDefaultGeneratorService() GeneratorService {
	return &DefaultGeneratorService{}
}

// CreateGenerator creates a generator using the factory
func (s *DefaultGeneratorService) CreateGenerator(config *factory.GeneratorConfig) (domain.Generator, error) {
	generatorFactory := factory.NewGeneratorFactory(config)
	generator, err := generatorFactory.Create()
	if err != nil {
		return nil, err
	}
	return generator, nil
}

// Generate executes the generation process
func (s *DefaultGeneratorService) Generate(ctx context.Context, generator domain.Generator, prompt string, definition *jsonSchema.Definition) (*domain.GenerationResult, error) {
	// Create generation request
	request := domain.NewGenerationRequest(prompt, definition).WithContext(ctx)

	// Generate the object
	result, err := generator.Generate(request)
	if err != nil {
		return nil, err
	}

	return result, nil
}
