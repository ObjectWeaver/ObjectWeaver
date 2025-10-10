package LLM

import (
	"objectweaver/llmManagement"
	"time"

	"github.com/sashabaranov/go-openai"
	gogpt "github.com/sashabaranov/go-openai"
)

// --- Custom Errors and Types ---

// --- Interfaces ---

// ClientAdapter defines the component that actually processes the job's data.
type JobSumitter interface {
	SubmitJob(job *Job, workerChannel chan *Job) (string, *gogpt.Usage, error)
}

// BackoffManager defines the contract for different backoff strategies.
type BackoffManager interface {
	ApplyBackoff(workerID int)
	ActivateBackoff(workerID int, retryAfter time.Duration)
	ResetBackoff(workerID int)
}

type Job struct {
	Result  chan *openai.ChatCompletionResponse
	Tokens  int
	Inputs  *llmManagement.Inputs
	Error   chan error
	Retries int // Tracks the number of retry attempts for transient errors.
}
