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
// <https://objectweaver.dev/licensing/server-side-public-license>.
package domain

// ResultAssembler - Assembles final result from individual task results
//
// Implementations:
//   - infrastructure/assembly/default.go: DefaultAssembler (synchronous)
//   - infrastructure/assembly/streaming.go: StreamingAssembler (field-level streaming)
//   - infrastructure/assembly/complete.go: CompleteStreamingAssembler (complete-only streaming)
//   - infrastructure/assembly/progressive.go: ProgressiveObjectAssembler (token-level streaming)
//
// Created by: factory/generator_factory.go:createAssembler()
// Used by: All Generator implementations
//
// Responsibilities:
//   - Collect TaskResults into final GenerationResult
//   - Build final map[string]interface{} structure
//   - Aggregate metadata (tokens, cost)
//   - Handle streaming assembly for different granularities
type ResultAssembler interface {
	Assemble(results []*TaskResult) (*GenerationResult, error)
}

// StreamingAssembler - Assembles streaming results at field-level granularity
type StreamingAssembler interface {
	ResultAssembler
	AssembleStreaming(results <-chan *TaskResult) (<-chan *StreamChunk, error)
}

// ProgressiveAssembler - Assembles progressive token-level results
type ProgressiveAssembler interface {
	ResultAssembler
	AssembleProgressive(tokenStream <-chan *TokenStreamChunk) (<-chan *AccumulatedStreamChunk, error)
}