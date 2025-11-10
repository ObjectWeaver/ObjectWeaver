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
package execution

import (
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// MapProcessor handles map-type fields
type MapProcessor struct {
	llmProvider    domain.LLMProvider
	promptBuilder  domain.PromptBuilder
	fieldProcessor *FieldProcessor
}

func NewMapProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *MapProcessor {
	return &MapProcessor{
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
	}
}

func NewMapProcessorWithFieldProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder, fieldProcessor *FieldProcessor) *MapProcessor {
	return &MapProcessor{
		llmProvider:    llmProvider,
		promptBuilder:  promptBuilder,
		fieldProcessor: fieldProcessor,
	}
}

func (p *MapProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Map
}

func (p *MapProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Simplified map processing
	result := make(map[string]interface{})
	metadata := domain.NewResultMetadata()

	taskResult := domain.NewTaskResult(task.ID(), task.Key(), result, metadata)
	return taskResult.WithPath(task.Path()), nil
}
