package LLM

import (
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/domain"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

func TestJob_Creation(t *testing.T) {
	resultChan := make(chan *domain.JobResult, 1)
	errorChan := make(chan error, 1)

	inputs := &llmManagement.Inputs{
		Prompt: "test prompt",
	}

	job := &Job{
		Result:  resultChan,
		Tokens:  100,
		Inputs:  inputs,
		Error:   errorChan,
		Retries: 0,
	}

	if job == nil {
		t.Fatal("Expected non-nil job")
	}
	if job.Tokens != 100 {
		t.Errorf("Expected 100 tokens, got %d", job.Tokens)
	}
	if job.Retries != 0 {
		t.Errorf("Expected 0 retries, got %d", job.Retries)
	}
	if job.Inputs != inputs {
		t.Error("Expected inputs to be set correctly")
	}
	if job.Result != resultChan {
		t.Error("Expected result channel to be set correctly")
	}
	if job.Error != errorChan {
		t.Error("Expected error channel to be set correctly")
	}
}

func TestJob_Channels(t *testing.T) {
	resultChan := make(chan *domain.JobResult, 1)
	errorChan := make(chan error, 1)

	job := &Job{
		Result: resultChan,
		Error:  errorChan,
		Tokens: 50,
	}

	// Test sending result
	go func() {
		result := domain.CreateJobResult(&openai.ChatCompletionResponse{}, nil)
		resultChan <- result
	}()

	select {
	case result := <-job.Result:
		if result == nil {
			t.Error("Expected non-nil result")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for result")
	}

	// Test sending error
	go func() {
		errorChan <- &openai.APIError{Message: "test error"}
	}()

	select {
	case err := <-job.Error:
		if err == nil {
			t.Error("Expected non-nil error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for error")
	}
}

func TestJob_RetriesIncrement(t *testing.T) {
	job := &Job{
		Result:  make(chan *domain.JobResult, 1),
		Error:   make(chan error, 1),
		Tokens:  0,
		Retries: 0,
	}

	// Simulate retries
	for i := 1; i <= 3; i++ {
		job.Retries++
		if job.Retries != i {
			t.Errorf("Expected %d retries, got %d", i, job.Retries)
		}
	}
}

func TestJob_TokenTracking(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
	}{
		{"zero tokens", 0},
		{"small job", 100},
		{"medium job", 1000},
		{"large job", 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Result: make(chan *domain.JobResult, 1),
				Error:  make(chan error, 1),
				Tokens: tt.tokens,
			}

			if job.Tokens != tt.tokens {
				t.Errorf("Expected %d tokens, got %d", tt.tokens, job.Tokens)
			}
		})
	}
}

func TestJob_InputsValidation(t *testing.T) {
	tests := []struct {
		name   string
		inputs *llmManagement.Inputs
	}{
		{
			name: "with prompt",
			inputs: &llmManagement.Inputs{
				Prompt: "test prompt",
			},
		},
		{
			name: "with system prompt",
			inputs: &llmManagement.Inputs{
				Prompt:       "test",
				SystemPrompt: "system",
			},
		},
		{
			name:   "nil inputs",
			inputs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Result: make(chan *domain.JobResult, 1),
				Error:  make(chan error, 1),
				Inputs: tt.inputs,
			}

			if job.Inputs != tt.inputs {
				t.Error("Expected inputs to match")
			}
		})
	}
}

func TestJob_ChannelBuffering(t *testing.T) {
	// Test unbuffered channels
	job1 := &Job{
		Result: make(chan *domain.JobResult),
		Error:  make(chan error),
	}

	if cap(job1.Result) != 0 {
		t.Errorf("Expected unbuffered result channel, got capacity %d", cap(job1.Result))
	}
	if cap(job1.Error) != 0 {
		t.Errorf("Expected unbuffered error channel, got capacity %d", cap(job1.Error))
	}

	// Test buffered channels
	job2 := &Job{
		Result: make(chan *domain.JobResult, 10),
		Error:  make(chan error, 5),
	}

	if cap(job2.Result) != 10 {
		t.Errorf("Expected result channel capacity 10, got %d", cap(job2.Result))
	}
	if cap(job2.Error) != 5 {
		t.Errorf("Expected error channel capacity 5, got %d", cap(job2.Error))
	}
}

func TestJob_ConcurrentAccess(t *testing.T) {
	job := &Job{
		Result:  make(chan *domain.JobResult, 10),
		Error:   make(chan error, 10),
		Tokens:  100,
		Retries: 0,
	}

	done := make(chan bool)

	// Simulate concurrent reads
	go func() {
		for i := 0; i < 5; i++ {
			_ = job.Tokens
			_ = job.Retries
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			_ = job.Tokens
			_ = job.Retries
		}
		done <- true
	}()

	// Wait for goroutines
	<-done
	<-done
}

func TestJob_NilInputs(t *testing.T) {
	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Error:  make(chan error, 1),
		Inputs: nil,
	}

	if job.Inputs != nil {
		t.Error("Expected nil inputs")
	}
}

func TestJob_MultipleResults(t *testing.T) {
	resultChan := make(chan *domain.JobResult, 3)

	job := &Job{
		Result: resultChan,
		Error:  make(chan error, 1),
	}

	// Send multiple results
	results := []*openai.ChatCompletionResponse{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
	}

	for _, chatRes := range results {
		result := domain.CreateJobResult(chatRes, nil)
		resultChan <- result
	}

	// Receive all results
	for i := 0; i < len(results); i++ {
		select {
		case result := <-job.Result:
			if result == nil {
				t.Errorf("Expected non-nil result at index %d", i)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timeout waiting for result %d", i)
		}
	}
}

func TestJob_CloseChannels(t *testing.T) {
	resultChan := make(chan *domain.JobResult)
	errorChan := make(chan error)

	job := &Job{
		Result: resultChan,
		Error:  errorChan,
	}

	// Close channels
	close(resultChan)
	close(errorChan)

	// Verify channels are closed
	_, ok := <-job.Result
	if ok {
		t.Error("Expected result channel to be closed")
	}

	_, ok = <-job.Error
	if ok {
		t.Error("Expected error channel to be closed")
	}
}
