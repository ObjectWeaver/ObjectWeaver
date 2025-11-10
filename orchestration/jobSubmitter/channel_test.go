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
	"testing"

	"objectweaver/llmManagement/LLM"
	"objectweaver/llmManagement/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

func TestChannelJobSubmitter_SubmitJob(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "local")
	t.Setenv("LLM_API_URL", "http://localhost:8080")
	submitter := &ChannelJobSubmitter{}
	model := string("gpt-3.5-turbo")
	def := &jsonSchema.Definition{
		Model: model,
	}
	newPrompt := "Test prompt"
	systemPrompt := "System prompt"
	outStream := make(chan interface{}, 1)

	// Temporarily replace the global WorkerChannel to avoid interference from the real system
	originalChannel := LLM.WorkerChannel
	LLM.WorkerChannel = make(chan *LLM.Job)
	defer func() { LLM.WorkerChannel = originalChannel }()

	// Start a goroutine to simulate the worker on the new channel
	go func() {
		job := <-LLM.WorkerChannel
		// Simulate processing
		response := &openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "Test response",
					},
				},
			},
			Usage: openai.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
		job.Result <- domain.CreateJobResult(response, nil)
	}()

	response, usage, err := submitter.SubmitJob(model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if response != "Test response" {
		t.Errorf("Expected 'Test response', got %s", response)
	}
	if usage.TotalTokens != 30 {
		t.Errorf("Expected 30 total tokens, got %d", usage.TotalTokens)
	}
}

func TestChannelJobSubmitter_SubmitJob_InitializesSendImage(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "local")
	t.Setenv("LLM_API_URL", "http://localhost:8080")
	submitter := &ChannelJobSubmitter{}
	model := string("gpt-3.5-turbo")
	def := &jsonSchema.Definition{
		Model: model,
		// SendImage is nil
	}
	newPrompt := "Test prompt"
	systemPrompt := "System prompt"
	outStream := make(chan interface{}, 1)

	// Temporarily replace the global WorkerChannel to avoid interference from the real system
	originalChannel := LLM.WorkerChannel
	LLM.WorkerChannel = make(chan *LLM.Job)
	defer func() { LLM.WorkerChannel = originalChannel }()

	// Start a goroutine to simulate the worker on the new channel
	go func() {
		job := <-LLM.WorkerChannel
		response := &openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "Response",
					},
				},
			},
			Usage: openai.Usage{},
		}
		job.Result <- domain.CreateJobResult(response, nil)
	}()

	_, _, err := submitter.SubmitJob(model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if def.SendImage == nil {
		t.Error("Expected SendImage to be initialized")
	}
}
