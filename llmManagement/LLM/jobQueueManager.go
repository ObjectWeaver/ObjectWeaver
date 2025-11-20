package LLM

import (
	"sync"
	"time"
)

type IJobQueueManager interface {
	StartManager(wg *sync.WaitGroup)
	Jobs() <-chan *Job
	StopManager()
	Enqueue(request *Job)
	Dequeue() *Job
}

// JobQueue manages the flow of jobs from ingestion to workers.
type JobQueueManager struct {
	mu       sync.Mutex
	jobChan  chan *Job
	stopChan chan struct{}
	jobQueue IJobQueue
}

func NewJobQueueManager(concurrency, maxQueueSize int, jobQueue IJobQueue) *JobQueueManager {
	return &JobQueueManager{
		jobChan:  make(chan *Job, concurrency),
		stopChan: make(chan struct{}),
		jobQueue: jobQueue,
	}
}

// StartManager begins moving jobs from the internal FIFO queue to the worker channel.
func (q *JobQueueManager) StartManager(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-q.stopChan:
			close(q.jobChan)
			return
		default:
			if job := q.jobQueue.Dequeue(); job != nil {
				q.jobChan <- job
			} else {
				// Use minimal sleep to prevent tight loop, but not block processing
				time.Sleep(100 * time.Microsecond)
			}
		}
	}
}

func (q *JobQueueManager) StopManager() {
	close(q.stopChan)
}

func (q *JobQueueManager) Jobs() <-chan *Job {
	return q.jobChan
}

func (q *JobQueueManager) Enqueue(request *Job) {
	q.jobQueue.Enqueue(request)
}

func (q *JobQueueManager) Dequeue() *Job {
	return q.jobQueue.Dequeue()
}
