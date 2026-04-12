package LLM

import (
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/domain"
	"sync"
	"testing"
	"time"
)

// Mock implementation of IJobQueueManager for testing
type mockJobQueueManager struct {
	jobs []*Job
	mu   sync.Mutex
}

func (m *mockJobQueueManager) StartManager(wg *sync.WaitGroup) {
	// Not needed for these tests
}

func (m *mockJobQueueManager) Jobs() <-chan *Job {
	// Not needed for these tests
	return nil
}

func (m *mockJobQueueManager) StopManager() {
	// Not needed for these tests
}

func (m *mockJobQueueManager) Enqueue(request *Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs = append(m.jobs, request)
}

func (m *mockJobQueueManager) Dequeue() *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.jobs) == 0 {
		return nil
	}
	job := m.jobs[0]
	m.jobs = m.jobs[1:]
	return job
}

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
	queue := &mockJobQueueManager{
		jobs: make([]*Job, 0),
	}
	job := &Job{
		Result:  make(chan *domain.JobResult, 1),
		Inputs:  &llmManagement.Inputs{},
		Error:   make(chan error, 1),
		Retries: 0,
	}
	workerID := 1
	err := error(nil) // some error

	// exponential backoff is 200ms for first retry, so wait a bit longer
	go rh.HandleTransientError(job, queue, workerID, err)

	// Wait for exponential backoff (200ms + buffer)
	time.Sleep(300 * time.Millisecond)

	// Check if retries incremented
	if job.Retries != 1 {
		t.Errorf("Expected retries 1, got %d", job.Retries)
	}

	// Check if job is prepended (dequeue should get it)
	dequeued := queue.Dequeue()
	if dequeued != job {
		t.Error("Job not prepended to queue")
	}
}

func TestHandleTransientError_MaxRetries(t *testing.T) {
	rh := NewRetryHandler(1, true)
	queue := &mockJobQueueManager{
		jobs: make([]*Job, 0),
	}
	job := &Job{
		Result:  make(chan *domain.JobResult, 1),
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
		Result:  make(chan *domain.JobResult, 1),
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
