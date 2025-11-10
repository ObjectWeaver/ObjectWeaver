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

import "github.com/objectweaver/go-sdk/jsonSchema"

// TypeProcessor - Handles generation for specific JSON schema types
//
// Implementations (infrastructure/execution/*_processor.go):
//   - primitive_processor.go: PrimitiveProcessor (string, basic types)
//   - object_processor.go: ObjectProcessor (nested objects)
//   - array_processor.go: ArrayProcessor (arrays)
//   - boolean_processor.go: BooleanProcessor (boolean values)
//   - number_processor.go: NumberProcessor (numbers)
//   - byte_processor.go: ByteProcessor (TTS, Image, STT operations)
//   - map_processor.go: MapProcessor (key-value maps)
//   - streaming_processors.go: Streaming variants
//
// Created by: factory/generator_factory.go:createTypeProcessors()
// Used by: CompositeTaskExecutor (routes tasks to appropriate processor)
//
// Responsibilities:
//   - Check if it can handle a specific schema type
//   - Generate content for that type using LLMProvider
//   - Build appropriate prompts using PromptBuilder
//   - Parse and validate LLM response
//   - Return TaskResult with generated value
type TypeProcessor interface {
	CanProcess(schemaType jsonSchema.DataType) bool
	Process(task *FieldTask, context *ExecutionContext) (*TaskResult, error)
}

// StreamingTypeProcessor - Type processor with token-level streaming support
type StreamingTypeProcessor interface {
	TypeProcessor
	ProcessStreaming(task *FieldTask, context *ExecutionContext) (<-chan *TokenStreamChunk, error)
}
