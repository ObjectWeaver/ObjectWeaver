package LLM

import (
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/domain"
	"testing"

	"objectweaver/jsonSchema"

	gogpt "github.com/sashabaranov/go-openai"
)

func TestValidateResult(t *testing.T) {
	t.Run("nil result", func(t *testing.T) {
		content, usage, err := validateResult(nil)
		if err == nil {
			t.Error("Expected error for nil result")
		}
		if content != "" {
			t.Errorf("Expected blank content, got %s", content)
		}
		if usage != nil {
			t.Error("Expected nil usage")
		}
	})

	t.Run("empty choices", func(t *testing.T) {
		result := &gogpt.ChatCompletionResponse{
			Choices: []gogpt.ChatCompletionChoice{},
		}
		jobResult := domain.CreateJobResult(result, nil)
		content, usage, err := validateResult(jobResult)
		if err == nil {
			t.Error("Expected error for empty choices")
		}
		if content != "" {
			t.Errorf("Expected blank content, got %s", content)
		}
		if usage != nil {
			t.Error("Expected nil usage")
		}
	})

	t.Run("valid result", func(t *testing.T) {
		expectedContent := "Hello world"
		expectedUsage := gogpt.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		}
		result := &gogpt.ChatCompletionResponse{
			Choices: []gogpt.ChatCompletionChoice{
				{
					Message: gogpt.ChatCompletionMessage{
						Content: expectedContent,
					},
				},
			},
			Usage: expectedUsage,
		}
		jobResult := domain.CreateJobResult(result, nil)
		content, usage, err := validateResult(jobResult)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if content != expectedContent {
			t.Errorf("Expected content %s, got %s", expectedContent, content)
		}
		if usage == nil || *usage != expectedUsage {
			t.Errorf("Expected usage %v, got %v", expectedUsage, usage)
		}
	})
}

func TestDefaultJobSubmitter_SubmitJob(t *testing.T) {
	submitter := NewDefaultJobSubmitter()

	t.Run("nil job", func(t *testing.T) {
		workerChannel := make(chan *Job, 1)
		content, usage, err := submitter.SubmitJob(nil, workerChannel)
		if err == nil {
			t.Error("Expected error for nil job")
		}
		if content != "" {
			t.Errorf("Expected blank content, got %s", content)
		}
		if usage != nil {
			t.Error("Expected nil usage")
		}
	})

	t.Run("valid job", func(t *testing.T) {
		workerChannel := make(chan *Job, 1)
		job := &Job{
			Result: make(chan *domain.JobResult, 1),
			Inputs: &llmManagement.Inputs{
				Def: &jsonSchema.Definition{
					Type: jsonSchema.String,
				},
			},
		}
		expectedContent := "Test response"
		expectedUsage := gogpt.Usage{
			PromptTokens:     5,
			CompletionTokens: 10,
			TotalTokens:      15,
		}
		result := &gogpt.ChatCompletionResponse{
			Choices: []gogpt.ChatCompletionChoice{
				{
					Message: gogpt.ChatCompletionMessage{
						Content: expectedContent,
					},
				},
			},
			Usage: expectedUsage,
		}

		// Simulate worker
		go func() {
			receivedJob := <-workerChannel
			receivedJob.Result <- domain.CreateJobResult(result, nil)
		}()

		content, usage, err := submitter.SubmitJob(job, workerChannel)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if content != expectedContent {
			t.Errorf("Expected content %s, got %s", expectedContent, content)
		}
		if usage == nil || *usage != expectedUsage {
			t.Errorf("Expected usage %v, got %v", expectedUsage, usage)
		}
	})
}
