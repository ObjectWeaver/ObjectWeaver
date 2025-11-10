package LLM

import (
	"errors"
	"objectweaver/llmManagement/domain"

	gogpt "github.com/sashabaranov/go-openai"
)

const blank = ""

type VariedJobSubmitter struct{}

func NewVariedJobSubmitter() *VariedJobSubmitter {
	return &VariedJobSubmitter{}
}

func (v *VariedJobSubmitter) SubmitJob(job *Job, workerChannel chan *Job) (any, *gogpt.Usage, error) {
	select {
	case WorkerChannel <- job:
	default:
		return v.SubmitJob(job, workerChannel)
	}

	result := <-job.Result
	close(job.Result)

	//TODO have an option to return a different kind of result for the vector/embeddings types
	if result.EmbeddingRes != nil {
		// Return the embedding vector ([]float32) instead of the Embedding struct
		// to avoid proto serialization issues
		return result.EmbeddingRes.Data[0].Embedding, nil, nil
	}

	return validateResult(result)
}

type DefaultJobSubmitter struct{}

func NewDefaultJobSubmitter() *DefaultJobSubmitter {
	return &DefaultJobSubmitter{}
}

func (d *DefaultJobSubmitter) SubmitJob(job *Job, workerChannel chan *Job) (any, *gogpt.Usage, error) {
	if job == nil {
		return blank, nil, errors.New("error, the job is nil")
	}

	workerChannel <- job

	result := <-job.Result
	close(job.Result)

	if result.EmbeddingRes != nil {
		// Return the embedding vector ([]float32) instead of the Embedding struct
		// to avoid proto serialization issues
		return result.EmbeddingRes.Data[0].Embedding, nil, nil
	}

	return validateResult(result)
}

func validateResult(result *domain.JobResult) (any, *gogpt.Usage, error) {
	if result == nil {
		return blank, nil, errors.New("error, the returned result is nil")
	}
	if len(result.ChatRes.Choices) < 1 {
		return blank, nil, errors.New("error, the returned result is empty")
	}

	return result.ChatRes.Choices[0].Message.Content, &result.ChatRes.Usage, nil
}
