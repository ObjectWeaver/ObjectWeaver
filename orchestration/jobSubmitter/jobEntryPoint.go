package jobSubmitter

import (
	"context"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/LLM"
	"objectweaver/llmManagement/domain"
	"objectweaver/logger"
	"os"
	"strings"
	"time"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

func NewDefaultJobEntryPoint() JobEntryPoint {
	return &DefaultJobEntryPoint{}
}

// JobEntryPoint handles job submission logic.
type JobEntryPoint interface {
	SubmitJob(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *openai.Usage, error)
}

type DefaultJobEntryPoint struct{}

func (js *DefaultJobEntryPoint) SubmitJob(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *openai.Usage, error) {
	// If def is nil, create a minimal definition with the model
	if def == nil {
		def = &jsonSchema.Definition{
			Model: model,
		}
	} else {
		// This ensures SendImage requests have the correct model
		def.Model = model
	}

	startTime := time.Now()
	logger.Printf("[JobEntryPoint START] Submitting job with model: %s at %v", model, startTime)

	// CRITICAL: Allocate input on the heap, not stack, to prevent memory corruption
	// when goroutines access it asynchronously. Taking address of stack variable
	// causes nil pointer dereference under high concurrency.
	input := &llmManagement.Inputs{
		Ctx:          ctx, // Pass context from caller for cancellation support
		Prompt:       newPrompt,
		SystemPrompt: systemPrompt,
		Def:          def,
	}

	// Determine priority from definition (for batch processing)
	// Priority < 0 = batch processing, >= 0 = direct processing
	priority := def.Priority

	job := &LLM.Job{
		Inputs:   input,
		Result:   make(chan *domain.JobResult, 1),
		Tokens:   0,
		Priority: priority,
	}

	if input.Def.SendImage == nil {
		input.Def.SendImage = &jsonSchema.SendImage{}
		input.Def.SendImage.ImagesData = nil
	}

	prepDuration := time.Since(startTime)
	logger.Printf("[JobEntryPoint] Job prep took %v", prepDuration)

	submitStart := time.Now()

	// Route based on priority:
	// - Priority < 0: Send to batch processing queue (if enabled)
	// - Priority >= 0: Use direct HTTP client for fastest response
	var submitter LLM.JobSumitter
	if priority < 0 && strings.ToLower(os.Getenv("LLM_ENABLE_BATCH")) == "true" {
		// Low priority job - send to batch queue (uses orchestrator)
		logger.Printf("[JobEntryPoint] Routing low-priority job (priority=%d) to batch queue", priority)
		submitter = LLM.JobSubmitterFactory(LLM.DefaultSubmitter)
	} else {
		// Normal/high priority - use DirectSubmitter for immediate processing
		logger.Printf("[JobEntryPoint] Routing job (priority=%d) to DirectSubmitter", priority)
		submitter = LLM.JobSubmitterFactory(LLM.DirectSubmitter)
	}

	logger.Printf("[JobEntryPoint] About to call submitter.SubmitJob() at %v", time.Since(startTime))
	completion, usage, err := submitter.SubmitJob(job, LLM.WorkerChannel)
	submitDuration := time.Since(submitStart)
	logger.Printf("[JobEntryPoint] submitter.SubmitJob() returned after %v", submitDuration)

	if err != nil {
		logger.Printf("[JobEntryPoint ERROR] Job submission failed after %v: %v", time.Since(startTime), err)
		return "", nil, err
	}

	logger.Printf("[JobEntryPoint] Job completed in %v (submit took %v), response: %v", time.Since(startTime), submitDuration, completion)

	return completion, usage, err
}
