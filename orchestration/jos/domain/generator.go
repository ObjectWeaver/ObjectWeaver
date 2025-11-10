// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
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
