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

import "sync"

// TaskExecutor - Executes field generation tasks by delegating to type-specific processors
//
// Implementation: infrastructure/execution/task_executor.go (CompositeTaskExecutor)
// Created by: factory/generator_factory.go:createExecutor()
// Used by: ExecutionStrategy implementations
//
// Responsibilities:
//   - Route tasks to appropriate TypeProcessor based on schema type
//   - Handle special cases (byte operations: TTS, Image, STT)
//   - Coordinate with LLMProvider and PromptBuilder
//   - Execute batches of tasks
//   - Process decision points and return additional results from branches
//
// Execute returns a slice of TaskResults to support decision points generating
// multiple sibling fields. The first result is the primary field, subsequent
// results are additional fields from decision point branches.
type TaskExecutor interface {
	Execute(task *FieldTask, context *ExecutionContext) ([]*TaskResult, error)
	ExecuteBatch(tasks []*FieldTask, context *ExecutionContext) ([]*TaskResult, error)
}

// ExecutionContext provides context for task execution
type ExecutionContext struct {
	request          *GenerationRequest
	parentContext    *ExecutionContext
	generatedValues  map[string]interface{}
	metadata         map[string]interface{}
	promptContext    *PromptContext
	generationConfig *GenerationConfig
	mu               sync.RWMutex // Protects generatedValues map from concurrent access
}

func NewExecutionContext(request *GenerationRequest) *ExecutionContext {
	return &ExecutionContext{
		request:          request,
		generatedValues:  make(map[string]interface{}),
		metadata:         make(map[string]interface{}),
		promptContext:    NewPromptContext(),
		generationConfig: DefaultGenerationConfig(),
	}
}

func (e *ExecutionContext) WithParent(parent *FieldTask) *ExecutionContext {
	return &ExecutionContext{
		request:          e.request,
		parentContext:    e,
		generatedValues:  e.copyGeneratedValues(),
		metadata:         e.copyMetadata(),
		promptContext:    e.promptContext,
		generationConfig: e.generationConfig,
	}
}

func (e *ExecutionContext) Request() *GenerationRequest         { return e.request }
func (e *ExecutionContext) PromptContext() *PromptContext       { return e.promptContext }
func (e *ExecutionContext) GenerationConfig() *GenerationConfig { return e.generationConfig }

// GeneratedValues returns a copy of the generated values map for thread safety
func (e *ExecutionContext) GeneratedValues() map[string]interface{} {
	return e.copyGeneratedValues()
}

// GetGeneratedValue safely retrieves a single value from the generated values map
func (e *ExecutionContext) GetGeneratedValue(key string) (interface{}, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	value, exists := e.generatedValues[key]
	return value, exists
}

func (e *ExecutionContext) SetGeneratedValue(key string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.generatedValues[key] = value
}

func (e *ExecutionContext) copyGeneratedValues() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copied := make(map[string]interface{})
	for k, v := range e.generatedValues {
		copied[k] = v
	}
	return copied
}

func (e *ExecutionContext) copyMetadata() map[string]interface{} {
	copied := make(map[string]interface{})
	for k, v := range e.metadata {
		copied[k] = v
	}
	return copied
}
