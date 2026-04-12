package LLM

import (
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"sync"
	"testing"
)

func TestNewFIFOQueueManager(t *testing.T) {
	manager := NewFIFOQueueManager()

	if manager == nil {
		t.Fatal("Expected non-nil FIFOQueueManager")
	}

	if manager.queue == nil {
		t.Error("Expected queue to be initialized")
	}

	if manager.cond == nil {
		t.Error("Expected cond to be initialized")
	}

	if manager.Size() != 0 {
		t.Errorf("Expected new queue to have size 0, got %d", manager.Size())
	}
}

func TestFIFOQueueManager_Enqueue(t *testing.T) {
	manager := NewFIFOQueueManager()

	job1 := &Job{Inputs: &llmManagement.Inputs{Priority: 1}}
	job2 := &Job{Inputs: &llmManagement.Inputs{Priority: 2}}

	manager.Enqueue(job1)
	if manager.Size() != 1 {
		t.Errorf("Expected size 1 after first enqueue, got %d", manager.Size())
	}

	manager.Enqueue(job2)
	if manager.Size() != 2 {
		t.Errorf("Expected size 2 after second enqueue, got %d", manager.Size())
	}
}

func TestFIFOQueueManager_Dequeue(t *testing.T) {
	manager := NewFIFOQueueManager()

	job1 := &Job{Inputs: &llmManagement.Inputs{Priority: 1}}
	job2 := &Job{Inputs: &llmManagement.Inputs{Priority: 2}}
	job3 := &Job{Inputs: &llmManagement.Inputs{Priority: 3}}

	manager.Enqueue(job1)
	manager.Enqueue(job2)
	manager.Enqueue(job3)

	// Dequeue should return jobs in FIFO order
	dequeued1 := manager.Dequeue()
	if dequeued1 != job1 {
		t.Error("Expected first enqueued job to be dequeued first")
	}
	if manager.Size() != 2 {
		t.Errorf("Expected size 2 after first dequeue, got %d", manager.Size())
	}

	dequeued2 := manager.Dequeue()
	if dequeued2 != job2 {
		t.Error("Expected second enqueued job to be dequeued second")
	}
	if manager.Size() != 1 {
		t.Errorf("Expected size 1 after second dequeue, got %d", manager.Size())
	}

	dequeued3 := manager.Dequeue()
	if dequeued3 != job3 {
		t.Error("Expected third enqueued job to be dequeued third")
	}
	if manager.Size() != 0 {
		t.Errorf("Expected size 0 after third dequeue, got %d", manager.Size())
	}
}

func TestFIFOQueueManager_DequeueEmpty(t *testing.T) {
	manager := NewFIFOQueueManager()

	// Dequeue from empty queue should return nil
	dequeued := manager.Dequeue()
	if dequeued != nil {
		t.Error("Expected nil when dequeuing from empty queue")
	}
}

func TestFIFOQueueManager_Size(t *testing.T) {
	manager := NewFIFOQueueManager()

	sizes := []int{0, 1, 2, 3, 4, 5}

	for i, expectedSize := range sizes {
		if manager.Size() != expectedSize {
			t.Errorf("Step %d: Expected size %d, got %d", i, expectedSize, manager.Size())
		}
		if i < len(sizes)-1 {
			manager.Enqueue(&Job{Inputs: &llmManagement.Inputs{Priority: int32(i)}})
		}
	}

	// Dequeue all and verify size decreases
	for i := 4; i >= 0; i-- {
		manager.Dequeue()
		if manager.Size() != i {
			t.Errorf("Expected size %d after dequeue, got %d", i, manager.Size())
		}
	}
}

func TestFIFOQueueManager_ConcurrentEnqueue(t *testing.T) {
	manager := NewFIFOQueueManager()

	const numGoroutines = 10
	const jobsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				job := &Job{Inputs: &llmManagement.Inputs{Priority: int32(id*100 + j)}}
				manager.Enqueue(job)
			}
		}(i)
	}

	wg.Wait()

	expectedSize := numGoroutines * jobsPerGoroutine
	if manager.Size() != expectedSize {
		t.Errorf("Expected size %d after concurrent enqueues, got %d", expectedSize, manager.Size())
	}
}

func TestFIFOQueueManager_ConcurrentDequeue(t *testing.T) {
	manager := NewFIFOQueueManager()

	// Pre-fill the queue
	const numJobs = 100
	for i := 0; i < numJobs; i++ {
		manager.Enqueue(&Job{Inputs: &llmManagement.Inputs{Priority: int32(i)}})
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	dequeuedCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			count := 0
			for {
				job := manager.Dequeue()
				if job == nil {
					break
				}
				count++
			}
			dequeuedCount <- count
		}()
	}

	wg.Wait()
	close(dequeuedCount)

	total := 0
	for count := range dequeuedCount {
		total += count
	}

	if total != numJobs {
		t.Errorf("Expected to dequeue %d jobs total, got %d", numJobs, total)
	}

	if manager.Size() != 0 {
		t.Errorf("Expected empty queue after all dequeues, got size %d", manager.Size())
	}
}

func TestFIFOQueueManager_MixedOperations(t *testing.T) {
	manager := NewFIFOQueueManager()

	// Enqueue some jobs
	job1 := &Job{Inputs: &llmManagement.Inputs{Priority: 1}}
	job2 := &Job{Inputs: &llmManagement.Inputs{Priority: 2}}
	manager.Enqueue(job1)
	manager.Enqueue(job2)

	// Dequeue one
	dequeued := manager.Dequeue()
	if dequeued != job1 {
		t.Error("Expected to dequeue job1")
	}

	// Enqueue more
	job3 := &Job{Inputs: &llmManagement.Inputs{Priority: 3}}
	manager.Enqueue(job3)

	// Check size
	if manager.Size() != 2 {
		t.Errorf("Expected size 2, got %d", manager.Size())
	}

	// Dequeue remaining in FIFO order
	dequeued = manager.Dequeue()
	if dequeued != job2 {
		t.Error("Expected to dequeue job2")
	}

	dequeued = manager.Dequeue()
	if dequeued != job3 {
		t.Error("Expected to dequeue job3")
	}

	// Queue should be empty
	if manager.Size() != 0 {
		t.Errorf("Expected empty queue, got size %d", manager.Size())
	}
}
