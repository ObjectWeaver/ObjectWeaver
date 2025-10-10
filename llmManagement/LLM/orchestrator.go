package LLM

import (
	"context"
	"errors"
	"objectweaver/llmManagement/backoff"
	"objectweaver/llmManagement/clientManager"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Orchestrator coordinates workers, queues, rate limiting, and error handling.
type Orchestrator struct {
	config          OrchestratorConfig
	clientAdapter   clientManager.ClientAdapter
	jobQueue        *JobQueue
	backoffManager  BackoffManager
	retryHandler    *RetryHandler
	errorClassifier *backoff.ErrorClassifier
	requestLimiter  *rate.Limiter
	tokenLimiter    *rate.Limiter
	wg              *sync.WaitGroup
}

// OrchestratorConfig holds the configuration for the Orchestrator.
type OrchestratorConfig struct {
	Concurrency          int
	MaxTokensPerMinute   int
	MaxRequestsPerMinute int
	MaxQueueSize         int
	Verbose              bool
}

// NewOrchestrator creates and wires up a new Orchestrator instance.
func NewOrchestrator(
	config OrchestratorConfig,
	handler clientManager.ClientAdapter,
	queue *JobQueue,
	backoffManager BackoffManager,
	retryHandler *RetryHandler,
	classifier *backoff.ErrorClassifier,
) *Orchestrator {

	requestRps := float64(config.MaxRequestsPerMinute) / 60.0
	requestBurst := config.MaxRequestsPerMinute
	if requestRps <= 0 {
		requestRps = 1.0
		requestBurst = 1
	}

	tokenRps := float64(config.MaxTokensPerMinute) / 60.0
	tokenBurst := config.MaxTokensPerMinute

	return &Orchestrator{
		config:          config,
		clientAdapter:   handler,
		jobQueue:        queue,
		backoffManager:  backoffManager,
		retryHandler:    retryHandler,
		errorClassifier: classifier,
		requestLimiter:  rate.NewLimiter(rate.Limit(requestRps), requestBurst),
		tokenLimiter:    rate.NewLimiter(rate.Limit(tokenRps), tokenBurst),
		wg:              &sync.WaitGroup{},
	}
}

// StartProcessing begins the worker pool and the queue manager.
func (o *Orchestrator) StartProcessing() {
	o.wg.Add(1)
	go o.jobQueue.StartManager(o.wg)

	for i := 0; i < o.config.Concurrency; i++ {
		o.wg.Add(1)
		go func(workerID int) {
			defer o.wg.Done()
			for job := range o.jobQueue.Jobs() {
				o.orchestrationJob(job, workerID)

			}
		}(i)
	}
}

// orchestrationJob is the core logic for processing a single job.
// It replaces the old processJob method.
func (o *Orchestrator) orchestrationJob(job *Job, workerID int) {
	// 1. Honor any active backoff period.
	o.backoffManager.ApplyBackoff(workerID)

	// 2. Wait for token-bucket limiters.
	ctx := context.Background()
	if err := o.requestLimiter.Wait(ctx); err != nil {
		o.jobQueue.Prepend(job)
		return
	}
	if err := o.tokenLimiter.WaitN(ctx, job.Tokens); err != nil {
		log.Printf("CRITICAL: Job dropped. It requires %d tokens, but the burst limit is %d.", job.Tokens, o.tokenLimiter.Burst())
		job.Result <- nil
		return
	}

	// 3. Process the job.
	resp, err := o.clientAdapter.Process(job.Inputs)

	// 4. Handle the result (error or success).
	if err != nil {
		errorType := o.errorClassifier.Classify(err)
		switch errorType {
		case backoff.ErrorTypeRateLimit:
			var rateLimitErr *backoff.RateLimitError
			var retryAfter time.Duration
			if errors.As(err, &rateLimitErr) {
				retryAfter = rateLimitErr.RetryAfter
			}
			o.backoffManager.ActivateBackoff(workerID, retryAfter)
			o.jobQueue.Prepend(job)

		case backoff.ErrorTypeTransient:
			o.retryHandler.HandleTransientError(job, o.jobQueue, workerID, err)

		case backoff.ErrorTypePermanent:
			o.retryHandler.HandlePermanentError(job, workerID, err)
		}
		return
	}

	// 5. On success, reset backoff and send result.
	o.backoffManager.ResetBackoff(workerID)
	job.Result <- resp
}

// Enqueue adds a job for processing.
func (o *Orchestrator) Enqueue(job *Job) {
	o.jobQueue.Enqueue(job)
}

// Stop gracefully shuts down the orchestrator and waits for workers to finish.
func (o *Orchestrator) Stop() {
	o.jobQueue.StopManager()
	o.wg.Wait()
}
