package LLM

import (
	"objectweaver/logger"
	"fmt"
	"time"
)

// RetryHandler manages retry logic for non-rate-limit errors.
type RetryHandler struct {
	MaxTransientRetries int
	Verbose             bool
}

func NewRetryHandler(maxRetries int, verbose bool) *RetryHandler {
	return &RetryHandler{
		MaxTransientRetries: maxRetries,
		Verbose:             verbose,
	}
}

func (rh *RetryHandler) HandleTransientError(job *Job, queue *JobQueue, workerID int, err error) {
	if job.Retries < rh.MaxTransientRetries {
		job.Retries++
		if rh.Verbose {
			logger.Output.Println(fmt.Sprintf("WARN: Transient error, retrying (%d/%d). Worker: %d, Error: %v", job.Retries, rh.MaxTransientRetries, workerID, err))
		}
		// Apply a simple incremental delay before requeueing
		time.Sleep(100 * time.Millisecond * time.Duration(job.Retries))
		queue.Prepend(job)
	} else {
		logger.Output.Println(fmt.Sprintf("ERROR: Job failed after max retries for transient error. Worker: %d, Error: %v", workerID, err))
		job.Result <- nil
	}
}

func (rh *RetryHandler) HandlePermanentError(job *Job, workerID int, err error) {
	logger.Output.Println(fmt.Sprintf("ERROR: Permanent error, dropping job. Worker: %d, Error: %v", workerID, err))
	job.Result <- nil
}
