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
	_ = startTime

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

	// Route based on priority:
	// - Priority < 0: Send to batch processing queue (if enabled)
	// - Priority >= 0: Use direct HTTP client for fastest response
	var submitter LLM.JobSumitter
	if priority < 0 && strings.ToLower(os.Getenv("LLM_ENABLE_BATCH")) == "true" {
		// Low priority job - send to batch queue (uses orchestrator)
		submitter = LLM.JobSubmitterFactory(LLM.DefaultSubmitter)
	} else {
		// Normal/high priority - use DirectSubmitter for immediate processing
		submitter = LLM.JobSubmitterFactory(LLM.DirectSubmitter)
	}

	completion, usage, err := submitter.SubmitJob(job, LLM.WorkerChannel)

	if err != nil {
		logger.Printf("[JobEntryPoint ERROR] Job submission failed after %v: %v", time.Since(startTime), err)
		return "", nil, err
	}

	return completion, usage, err
}
