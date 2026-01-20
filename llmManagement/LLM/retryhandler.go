package LLM

import (
	"fmt"
	"objectweaver/logger"
	"time"
)

const (
	// baseBackoffDelay is the starting delay for exponential backoff
	baseBackoffDelay = 200 * time.Millisecond
	// maxBackoffDelay caps the maximum delay between retries
	maxBackoffDelay = 5 * time.Second
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

func (rh *RetryHandler) HandleTransientError(job *Job, queue IJobQueueManager, workerID int, err error) {
	if job.Retries < rh.MaxTransientRetries {
		job.Retries++
		if rh.Verbose {
			logger.Output.Println(fmt.Sprintf("WARN: Transient error, retrying (%d/%d). Worker: %d, Error: %v", job.Retries, rh.MaxTransientRetries, workerID, err))
		}
		// Apply exponential backoff with cap before requeueing
		backoff := rh.calculateBackoff(job.Retries)
		logger.Output.Println(fmt.Sprintf("INFO: Worker %d backing off for %v before retry", workerID, backoff))
		time.Sleep(backoff)
		queue.Enqueue(job)
	} else {
		logger.Output.Println(fmt.Sprintf("ERROR: Job failed after max retries (%d) for transient error. Worker: %d, Error: %v", rh.MaxTransientRetries, workerID, err))
		job.Result <- nil
	}
}

func (rh *RetryHandler) HandlePermanentError(job *Job, workerID int, err error) {
	logger.Output.Println(fmt.Sprintf("ERROR: Permanent error, dropping job. Worker: %d, Error: %v", workerID, err))
	job.Result <- nil
}

// calculateBackoff returns exponential backoff duration capped at maxBackoffDelay
func (rh *RetryHandler) calculateBackoff(retryCount int) time.Duration {
	// exponential backoff: 200ms, 400ms, 800ms, 1.6s, 3.2s, capped at 5s
	backoff := baseBackoffDelay * time.Duration(1<<(retryCount-1))
	if backoff > maxBackoffDelay {
		backoff = maxBackoffDelay
	}
	return backoff
}
