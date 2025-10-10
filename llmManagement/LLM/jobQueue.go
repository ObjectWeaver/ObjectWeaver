package LLM

import (
	"sync"
	"time"
)

// JobQueue manages the flow of jobs from ingestion to workers.
type JobQueue struct {
	mu       sync.Mutex
	fifo     []*Job
	jobChan  chan *Job
	stopChan chan struct{}
}

func NewJobQueue(concurrency, maxQueueSize int) *JobQueue {
	return &JobQueue{
		fifo:     make([]*Job, 0, maxQueueSize),
		jobChan:  make(chan *Job, concurrency),
		stopChan: make(chan struct{}),
	}
}

// Enqueue adds a job to the end of the queue.
func (q *JobQueue) Enqueue(job *Job) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.fifo = append(q.fifo, job)
}

// Prepend adds a job to the front of the queue for immediate retry.
func (q *JobQueue) Prepend(job *Job) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.fifo = append([]*Job{job}, q.fifo...)
}

// Dequeue removes and returns a job from the front of the queue.
func (q *JobQueue) dequeue() *Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.fifo) == 0 {
		return nil
	}
	job := q.fifo[0]
	q.fifo = q.fifo[1:]
	return job
}

// StartManager begins moving jobs from the internal FIFO queue to the worker channel.
func (q *JobQueue) StartManager(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-q.stopChan:
			close(q.jobChan)
			return
		default:
			if job := q.dequeue(); job != nil {
				q.jobChan <- job
			} else {
				time.Sleep(10 * time.Millisecond) // Wait if queue is empty
			}
		}
	}
}

func (q *JobQueue) StopManager() {
	close(q.stopChan)
}

func (q *JobQueue) Jobs() <-chan *Job {
	return q.jobChan
}
