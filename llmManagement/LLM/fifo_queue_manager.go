package LLM

import (
	"sync"
)

// FIFOQueueManager implements JobQueue interface with FIFO ordering
type FIFOQueueManager struct {
	queue []*Job
	mu    sync.Mutex
	cond  *sync.Cond
}

func NewFIFOQueueManager() *FIFOQueueManager {
	manager := &FIFOQueueManager{
		queue: make([]*Job, 0),
	}
	manager.cond = sync.NewCond(&manager.mu)
	return manager
}

func (f *FIFOQueueManager) Enqueue(request *Job) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.queue = append(f.queue, request)
	f.cond.Signal() // Notify one waiting goroutine
}

func (f *FIFOQueueManager) Dequeue() *Job {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.queue) == 0 {
		return nil
	}

	item := f.queue[0]
	f.queue = f.queue[1:]
	return item
}

// Size returns the current queue size
func (f *FIFOQueueManager) Size() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.queue)
}
