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
package jobSubmitter

import (
	"testing"

	"objectweaver/llmManagement/LLM"
	"objectweaver/llmManagement/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

func TestDefaultJobEntryPoint_SubmitJob(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "local")
	t.Setenv("LLM_API_URL", "http://localhost:8080")
	entryPoint := NewDefaultJobEntryPoint()
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

	response, usage, err := entryPoint.SubmitJob(model, def, newPrompt, systemPrompt, outStream)
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

func TestDefaultJobEntryPoint_SubmitJob_DefNil(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "local")
	t.Setenv("LLM_API_URL", "http://localhost:8080")
	entryPoint := NewDefaultJobEntryPoint()
	model := string("gpt-3.5-turbo")
	var def *jsonSchema.Definition = nil
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

	response, _, err := entryPoint.SubmitJob(model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if response != "Response" {
		t.Errorf("Expected 'Response', got %s", response)
	}
}

func TestDefaultJobEntryPoint_SubmitJob_SetsModel(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "local")
	t.Setenv("LLM_API_URL", "http://localhost:8080")
	entryPoint := NewDefaultJobEntryPoint()
	model := string("gpt-4")
	def := &jsonSchema.Definition{
		// Model not set initially
	}
	newPrompt := "Prompt"
	systemPrompt := "Sys"
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
						Content: "Ok",
					},
				},
			},
			Usage: openai.Usage{},
		}
		job.Result <- domain.CreateJobResult(response, nil)
	}()

	_, _, err := entryPoint.SubmitJob(model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if def.Model != model {
		t.Errorf("Expected model %s, got %s", model, def.Model)
	}
}

func TestDefaultJobEntryPoint_SubmitJob_InitializesSendImage(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "local")
	t.Setenv("LLM_API_URL", "http://localhost:8080")
	entryPoint := NewDefaultJobEntryPoint()
	model := string("gpt-3.5-turbo")
	def := &jsonSchema.Definition{
		Model: model,
		// SendImage nil
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

	_, _, err := entryPoint.SubmitJob(model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if def.SendImage == nil {
		t.Error("Expected SendImage to be initialized")
	}
}
