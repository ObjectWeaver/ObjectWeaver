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
// <https://objectweaver.dev/licensing/server-side-public-license>.
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

func (rh *RetryHandler) HandleTransientError(job *Job, queue IJobQueueManager, workerID int, err error) {
	if job.Retries < rh.MaxTransientRetries {
		job.Retries++
		if rh.Verbose {
			logger.Output.Println(fmt.Sprintf("WARN: Transient error, retrying (%d/%d). Worker: %d, Error: %v", job.Retries, rh.MaxTransientRetries, workerID, err))
		}
		// Apply a simple incremental delay before requeueing
		time.Sleep(100 * time.Millisecond * time.Duration(job.Retries))
		queue.Enqueue(job)
	} else {
		logger.Output.Println(fmt.Sprintf("ERROR: Job failed after max retries for transient error. Worker: %d, Error: %v", workerID, err))
		job.Result <- nil
	}
}

func (rh *RetryHandler) HandlePermanentError(job *Job, workerID int, err error) {
	logger.Output.Println(fmt.Sprintf("ERROR: Permanent error, dropping job. Worker: %d, Error: %v", workerID, err))
	job.Result <- nil
}
