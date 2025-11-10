// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
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
				time.Sleep(10 * time.Millisecond) // Wait if queue is empty
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
