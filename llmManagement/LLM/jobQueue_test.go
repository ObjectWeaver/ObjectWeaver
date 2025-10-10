package LLM

import (
	"sync"
	"testing"
	"time"
)

func TestNewJobQueue(t *testing.T) {
	concurrency := 5
	maxQueueSize := 10
	q := NewJobQueue(concurrency, maxQueueSize)

	if q == nil {
		t.Fatal("NewJobQueue returned nil")
	}
	if cap(q.fifo) != maxQueueSize {
		t.Errorf("Expected fifo cap %d, got %d", maxQueueSize, cap(q.fifo))
	}
	if cap(q.jobChan) != concurrency {
		t.Errorf("Expected jobChan cap %d, got %d", concurrency, cap(q.jobChan))
	}
	if q.stopChan == nil {
		t.Error("stopChan should not be nil")
	}
}

func TestEnqueue(t *testing.T) {
	q := NewJobQueue(1, 10)
	job1 := &Job{}
	job2 := &Job{}

	q.Enqueue(job1)
	q.Enqueue(job2)

	q.mu.Lock()
	if len(q.fifo) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(q.fifo))
	}
	if q.fifo[0] != job1 || q.fifo[1] != job2 {
		t.Error("Jobs not enqueued in order")
	}
	q.mu.Unlock()
}

func TestPrepend(t *testing.T) {
	q := NewJobQueue(1, 10)
	job1 := &Job{}
	job2 := &Job{}

	q.Enqueue(job1)
	q.Prepend(job2)

	q.mu.Lock()
	if len(q.fifo) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(q.fifo))
	}
	if q.fifo[0] != job2 || q.fifo[1] != job1 {
		t.Error("Jobs not prepended correctly")
	}
	q.mu.Unlock()
}

func TestDequeue(t *testing.T) {
	q := NewJobQueue(1, 10)
	job := &Job{}
	q.Enqueue(job)

	dequeued := q.dequeue()
	if dequeued != job {
		t.Error("Dequeued job does not match enqueued")
	}

	// Dequeue from empty
	empty := q.dequeue()
	if empty != nil {
		t.Error("Dequeue from empty should return nil")
	}
}

func TestStartManager(t *testing.T) {
	q := NewJobQueue(1, 10)
	job := &Job{}

	var wg sync.WaitGroup
	wg.Add(1)
	go q.StartManager(&wg)

	q.Enqueue(job)

	select {
	case received := <-q.Jobs():
		if received != job {
			t.Error("Received job does not match")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for job")
	}

	q.StopManager()
	wg.Wait()
}

func TestStopManager(t *testing.T) {
	q := NewJobQueue(1, 10)

	var wg sync.WaitGroup
	wg.Add(1)
	go q.StartManager(&wg)

	q.StopManager()

	// Wait for manager to stop
	wg.Wait()

	// Channel should be closed
	select {
	case _, ok := <-q.Jobs():
		if ok {
			t.Error("Channel should be closed")
		}
	default:
		t.Error("Channel should be closed immediately")
	}
}

func TestJobs(t *testing.T) {
	q := NewJobQueue(1, 10)
	ch := q.Jobs()
	if ch == nil {
		t.Error("Jobs() returned nil")
	}
}