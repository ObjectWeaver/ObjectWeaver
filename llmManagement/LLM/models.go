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
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/domain"
	"time"

	"github.com/sashabaranov/go-openai"
)

// --- Custom Errors and Types ---

// --- Interfaces ---

// ClientAdapter defines the component that actually processes the job's data.
type JobSumitter interface {
	SubmitJob(job *Job, workerChannel chan *Job) (any, *openai.Usage, error)
}

// BackoffManager defines the contract for different backoff strategies.
type BackoffManager interface {
	ApplyBackoff(workerID int)
	ActivateBackoff(workerID int, retryAfter time.Duration)
	ResetBackoff(workerID int)
}

type Job struct {
	Result  chan *domain.JobResult
	Tokens  int
	Inputs  *llmManagement.Inputs
	Error   chan error
	Retries int // Tracks the number of retry attempts for transient errors.
}

