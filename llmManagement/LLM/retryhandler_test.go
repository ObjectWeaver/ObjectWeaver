package LLM

import (
	"firechimp/llmManagement"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

func TestNewRetryHandler(t *testing.T) {
	maxRetries := 5
	verbose := true
	rh := NewRetryHandler(maxRetries, verbose)

	if rh.MaxTransientRetries != maxRetries {
		t.Errorf("Expected MaxTransientRetries %d, got %d", maxRetries, rh.MaxTransientRetries)
	}
	if rh.Verbose != verbose {
		t.Errorf("Expected Verbose %t, got %t", verbose, rh.Verbose)
	}
}

func TestHandleTransientError_Retry(t *testing.T) {
	rh := NewRetryHandler(3, true)
	queue := NewJobQueue(1, 10)
	job := &Job{
		Result:  make(chan *openai.ChatCompletionResponse, 1),
		Inputs:  &llmManagement.Inputs{},
		Error:   make(chan error, 1),
		Retries: 0,
	}
	workerID := 1
	err := error(nil) // some error

	// Since sleep is 100ms * 1 = 100ms, but for test, we can wait or use goroutine
	go rh.HandleTransientError(job, queue, workerID, err)

	// Wait a bit for sleep
	time.Sleep(150 * time.Millisecond)

	// Check if retries incremented
	if job.Retries != 1 {
		t.Errorf("Expected retries 1, got %d", job.Retries)
	}

	// Check if job is prepended (dequeue should get it)
	dequeued := queue.dequeue()
	if dequeued != job {
		t.Error("Job not prepended to queue")
	}
}

func TestHandleTransientError_MaxRetries(t *testing.T) {
	rh := NewRetryHandler(1, true)
	queue := NewJobQueue(1, 10)
	job := &Job{
		Result:  make(chan *openai.ChatCompletionResponse, 1),
		Inputs:  &llmManagement.Inputs{},
		Error:   make(chan error, 1),
		Retries: 1, // already at max
	}
	workerID := 1
	err := error(nil)

	rh.HandleTransientError(job, queue, workerID, err)

	// Should not increment retries
	if job.Retries != 1 {
		t.Errorf("Expected retries 1, got %d", job.Retries)
	}

	// Check if nil sent to result
	select {
	case res := <-job.Result:
		if res != nil {
			t.Error("Expected nil result")
		}
	default:
		t.Error("No result sent")
	}
}

func TestHandlePermanentError(t *testing.T) {
	rh := NewRetryHandler(3, true)
	job := &Job{
		Result:  make(chan *openai.ChatCompletionResponse, 1),
		Inputs:  &llmManagement.Inputs{},
		Error:   make(chan error, 1),
		Retries: 0,
	}
	workerID := 1
	err := error(nil)

	rh.HandlePermanentError(job, workerID, err)

	// Check if nil sent to result
	select {
	case res := <-job.Result:
		if res != nil {
			t.Error("Expected nil result")
		}
	default:
		t.Error("No result sent")
	}
}
