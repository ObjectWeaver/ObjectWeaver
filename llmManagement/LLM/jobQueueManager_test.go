package LLM

import (
	"sync"
	"testing"
	"time"
)

// MockJobQueue is a mock implementation of IJobQueue for testing
type MockJobQueue struct {
	mu       sync.Mutex
	jobs     []*Job
	enqueued []*Job
	dequeued []*Job
}

func NewMockJobQueue() *MockJobQueue {
	return &MockJobQueue{
		jobs:     make([]*Job, 0),
		enqueued: make([]*Job, 0),
		dequeued: make([]*Job, 0),
	}
}

func (m *MockJobQueue) Enqueue(job *Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs = append(m.jobs, job)
	m.enqueued = append(m.enqueued, job)
}

func (m *MockJobQueue) Dequeue() *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.jobs) == 0 {
		return nil
	}
	job := m.jobs[0]
	m.jobs = m.jobs[1:]
	m.dequeued = append(m.dequeued, job)
	return job
}

func (m *MockJobQueue) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.jobs)
}

func (m *MockJobQueue) EnqueueCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.enqueued)
}

func (m *MockJobQueue) DequeueCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.dequeued)
}

func TestNewJobQueueManager(t *testing.T) {
	concurrency := 5
	maxQueueSize := 10
	mockQueue := NewMockJobQueue()

	manager := NewJobQueueManager(concurrency, maxQueueSize, mockQueue)

	if manager == nil {
		t.Fatal("NewJobQueueManager returned nil")
	}
	if manager.jobChan == nil {
		t.Error("jobChan should not be nil")
	}
	if cap(manager.jobChan) != concurrency {
		t.Errorf("Expected jobChan capacity %d, got %d", concurrency, cap(manager.jobChan))
	}
	if manager.stopChan == nil {
		t.Error("stopChan should not be nil")
	}
	if manager.jobQueue != mockQueue {
		t.Error("jobQueue should match the provided mock queue")
	}
}

func TestJobQueueManager_Enqueue(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	job1 := &Job{}
	job2 := &Job{}

	manager.Enqueue(job1)
	manager.Enqueue(job2)

	if mockQueue.EnqueueCount() != 2 {
		t.Errorf("Expected 2 enqueued jobs, got %d", mockQueue.EnqueueCount())
	}
	if mockQueue.Size() != 2 {
		t.Errorf("Expected queue size 2, got %d", mockQueue.Size())
	}
}

func TestJobQueueManager_Dequeue(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	job := &Job{}
	mockQueue.Enqueue(job)

	dequeued := manager.Dequeue()
	if dequeued != job {
		t.Error("Dequeued job does not match")
	}
	if mockQueue.DequeueCount() != 1 {
		t.Errorf("Expected 1 dequeued job, got %d", mockQueue.DequeueCount())
	}
}

func TestJobQueueManager_Dequeue_EmptyQueue(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	dequeued := manager.Dequeue()
	if dequeued != nil {
		t.Error("Dequeue from empty queue should return nil")
	}
}

func TestJobQueueManager_Jobs(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	jobsChan := manager.Jobs()
	if jobsChan == nil {
		t.Error("Jobs() should return a valid channel")
	}

	// Verify it's the same channel
	if jobsChan != manager.jobChan {
		t.Error("Jobs() should return the internal jobChan")
	}
}

func TestJobQueueManager_StartManager(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	job := &Job{}
	mockQueue.Enqueue(job)

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Wait for job to be sent to channel
	select {
	case received := <-manager.Jobs():
		if received != job {
			t.Error("Received job does not match enqueued job")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for job from manager")
	}

	manager.StopManager()
	wg.Wait()
}

func TestJobQueueManager_StartManager_MultipleJobs(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(5, 10, mockQueue)

	jobs := []*Job{&Job{}, &Job{}, &Job{}}
	for _, job := range jobs {
		mockQueue.Enqueue(job)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Receive all jobs
	receivedJobs := make([]*Job, 0, len(jobs))
	for i := 0; i < len(jobs); i++ {
		select {
		case job := <-manager.Jobs():
			receivedJobs = append(receivedJobs, job)
		case <-time.After(200 * time.Millisecond):
			t.Errorf("Timeout waiting for job %d", i)
		}
	}

	if len(receivedJobs) != len(jobs) {
		t.Errorf("Expected %d jobs, received %d", len(jobs), len(receivedJobs))
	}

	manager.StopManager()
	wg.Wait()
}

func TestJobQueueManager_StartManager_EmptyQueue(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Give it some time to process (should sleep and not send anything)
	select {
	case job := <-manager.Jobs():
		t.Errorf("Should not receive job from empty queue, got: %v", job)
	case <-time.After(50 * time.Millisecond):
		// Expected behavior - no job received
	}

	manager.StopManager()
	wg.Wait()
}

func TestJobQueueManager_StopManager(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Stop the manager
	manager.StopManager()

	// Wait for manager to stop
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Manager stopped successfully
	case <-time.After(200 * time.Millisecond):
		t.Error("Manager did not stop in time")
	}

	// Verify channel is closed
	select {
	case _, ok := <-manager.Jobs():
		if ok {
			t.Error("Channel should be closed after StopManager")
		}
	default:
		t.Error("Channel should be closed immediately after StopManager")
	}
}

func TestJobQueueManager_StartManager_AddJobsWhileRunning(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(5, 10, mockQueue)

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Add jobs after manager has started
	job1 := &Job{}
	job2 := &Job{}
	manager.Enqueue(job1)
	manager.Enqueue(job2)

	// Receive jobs
	receivedJobs := make([]*Job, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case job := <-manager.Jobs():
			receivedJobs = append(receivedJobs, job)
		case <-time.After(200 * time.Millisecond):
			t.Errorf("Timeout waiting for job %d", i)
		}
	}

	if len(receivedJobs) != 2 {
		t.Errorf("Expected 2 jobs, received %d", len(receivedJobs))
	}

	manager.StopManager()
	wg.Wait()
}

func TestJobQueueManager_ConcurrentEnqueue(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(10, 100, mockQueue)

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Enqueue jobs from multiple goroutines
	numGoroutines := 10
	jobsPerGoroutine := 5
	totalJobs := numGoroutines * jobsPerGoroutine

	var enqueueWg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		enqueueWg.Add(1)
		go func() {
			defer enqueueWg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				manager.Enqueue(&Job{})
			}
		}()
	}

	enqueueWg.Wait()

	// Receive all jobs
	receivedCount := 0
	timeout := time.After(1 * time.Second)
	for receivedCount < totalJobs {
		select {
		case <-manager.Jobs():
			receivedCount++
		case <-timeout:
			t.Errorf("Timeout: received only %d out of %d jobs", receivedCount, totalJobs)
			goto cleanup
		}
	}

cleanup:
	manager.StopManager()
	wg.Wait()

	if receivedCount != totalJobs {
		t.Errorf("Expected %d jobs, received %d", totalJobs, receivedCount)
	}
}

func TestJobQueueManager_StopManager_WithPendingJobs(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(5, 10, mockQueue)

	// Add jobs
	for i := 0; i < 5; i++ {
		mockQueue.Enqueue(&Job{})
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Start draining jobs in the background
	drainDone := make(chan struct{})
	go func() {
		for range manager.Jobs() {
			// Consume all jobs
		}
		close(drainDone)
	}()

	// Let some jobs be processed
	time.Sleep(50 * time.Millisecond)

	// Stop the manager
	manager.StopManager()

	// Manager should stop gracefully
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Manager stopped successfully
	case <-time.After(500 * time.Millisecond):
		t.Error("Manager did not stop in time with pending jobs")
	}

	// Wait for drain to complete
	select {
	case <-drainDone:
		// Drain completed
	case <-time.After(100 * time.Millisecond):
		t.Error("Channel drain did not complete")
	}
}

func TestJobQueueManager_Integration_WithRealJobQueue(t *testing.T) {
	// Test with actual FIFOQueueManager implementation
	realQueue := NewFIFOQueueManager()
	manager := NewJobQueueManager(5, 10, realQueue)

	job := &Job{}
	manager.Enqueue(job)

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	select {
	case received := <-manager.Jobs():
		if received != job {
			t.Error("Received job does not match with real FIFOQueueManager")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout with real FIFOQueueManager")
	}

	manager.StopManager()
	wg.Wait()
}

func TestJobQueueManager_ChannelCapacity(t *testing.T) {
	concurrency := 3
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(concurrency, 10, mockQueue)

	// Add more jobs than channel capacity
	for i := 0; i < concurrency+2; i++ {
		mockQueue.Enqueue(&Job{})
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go manager.StartManager(&wg)

	// Manager should handle buffering properly
	time.Sleep(50 * time.Millisecond)

	receivedCount := 0
	timeout := time.After(200 * time.Millisecond)
drainLoop:
	for {
		select {
		case <-manager.Jobs():
			receivedCount++
			if receivedCount >= concurrency+2 {
				break drainLoop
			}
		case <-timeout:
			break drainLoop
		}
	}

	manager.StopManager()
	wg.Wait()

	if receivedCount < concurrency {
		t.Errorf("Expected at least %d jobs, got %d", concurrency, receivedCount)
	}
}

func TestJobQueueManager_StartManager_WaitGroupDecrement(t *testing.T) {
	mockQueue := NewMockJobQueue()
	manager := NewJobQueueManager(1, 10, mockQueue)

	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan bool)
	go func() {
		manager.StartManager(&wg)
		done <- true
	}()

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	manager.StopManager()

	// Verify WaitGroup is decremented
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// WaitGroup properly decremented
	case <-time.After(200 * time.Millisecond):
		t.Error("WaitGroup was not decremented properly")
	}

	select {
	case <-done:
		// StartManager returned
	case <-time.After(200 * time.Millisecond):
		t.Error("StartManager did not return")
	}
}
