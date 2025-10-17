package LLM

import (
	"container/heap"
	"sync"
)

// RequestQueueManager implements JobQueue interface with priority-based ordering
type RequestQueueManager struct {
	queue *PriorityQueue
	mu    sync.Mutex
	cond  *sync.Cond
}

func NewRequestQueueManager() *RequestQueueManager {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	manager := &RequestQueueManager{
		queue: &pq,
	}
	manager.cond = sync.NewCond(&manager.mu)
	return manager
}

func (r *RequestQueueManager) Enqueue(request *Job) {
	r.mu.Lock()
	defer r.mu.Unlock()

	heap.Push(r.queue, request)
	r.cond.Signal() // Notify one waiting goroutine
}

func (r *RequestQueueManager) Dequeue() *Job {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.queue.Len() == 0 {
		return nil
	}

	return heap.Pop(r.queue).(*Job)
}

// Size returns the current queue size
func (r *RequestQueueManager) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.queue.Len()
}
