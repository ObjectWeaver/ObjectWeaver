package domain

import (
	"context"

	"objectweaver/jsonSchema"
)

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
	Process(ctx context.Context, task *FieldTask, execContext *ExecutionContext) (*TaskResult, error)
}

// StreamingTypeProcessor - Type processor with token-level streaming support
type StreamingTypeProcessor interface {
	TypeProcessor
	ProcessStreaming(ctx context.Context, task *FieldTask, execContext *ExecutionContext) (<-chan *TokenStreamChunk, error)
}
