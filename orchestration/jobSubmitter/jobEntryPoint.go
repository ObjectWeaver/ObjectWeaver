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
package jobSubmitter

import (
	"log"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/LLM"
	"objectweaver/llmManagement/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

func NewDefaultJobEntryPoint() JobEntryPoint {
	return &DefaultJobEntryPoint{}
}

// JobEntryPoint handles job submission logic.
type JobEntryPoint interface {
	SubmitJob(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *openai.Usage, error)
}

type DefaultJobEntryPoint struct{}

func (js *DefaultJobEntryPoint) SubmitJob(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *openai.Usage, error) {
	// If def is nil, create a minimal definition with the model
	if def == nil {
		def = &jsonSchema.Definition{
			Model: model,
		}
	} else {
		// CRITICAL FIX: Always set the model field even when def is not nil
		// This ensures SendImage requests have the correct model
		def.Model = model
	}

	log.Printf("[JobEntryPoint] Submitting job with model: %s", model)

	input := llmManagement.Inputs{
		Prompt:       newPrompt,
		SystemPrompt: systemPrompt,
		Def:          def,
	}

	job := &LLM.Job{
		Inputs: &input,
		Result: make(chan *domain.JobResult),
		Tokens: 0,
	}

	if input.Def.SendImage == nil {
		input.Def.SendImage = &jsonSchema.SendImage{}
		input.Def.SendImage.ImagesData = nil
	}

	submitter := LLM.JobSubmitterFactory(LLM.DefaultSubmitter)
	completion, usage, err := submitter.SubmitJob(job, LLM.WorkerChannel)

	if err != nil {
		log.Printf("[JobEntryPoint ERROR] Job submission failed: %v", err)
		return completion, usage, err
	}

	log.Printf("[JobEntryPoint] Job completed successfully, response: %v", completion)

	return completion, usage, err
}
