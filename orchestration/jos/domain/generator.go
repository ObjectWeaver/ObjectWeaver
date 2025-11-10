package domain

// Generator - Primary domain service interface (Main API boundary)
//
// Implementations:
//   - application/default_generator.go: DefaultGenerator (synchronous)
//   - application/streaming_generator.go: StreamingGenerator (field-level streaming)
//   - application/progressive_generator.go: ProgressiveGenerator (token-level streaming)
//
// Created by: factory/generator_factory.go
// Used by: service/objectGen.go, grpcService/*.go
type Generator interface {
	Generate(request *GenerationRequest) (*GenerationResult, error)
	GenerateStream(request *GenerationRequest) (<-chan *StreamChunk, error)
	GenerateStreamProgressive(request *GenerationRequest) (<-chan *AccumulatedStreamChunk, error)
}
