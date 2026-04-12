package LLM

import (
	"errors"
	"objectweaver/llmManagement/domain"
	"sync"

	"objectweaver/jsonSchema"

	gogpt "github.com/sashabaranov/go-openai"
)

const blank = ""

type DefaultJobSubmitter struct{}

var (
	defaultJobSubmitterInstance *DefaultJobSubmitter
	once                        sync.Once
)

func NewDefaultJobSubmitter() *DefaultJobSubmitter {
	once.Do(func() {
		defaultJobSubmitterInstance = &DefaultJobSubmitter{}
	})
	return defaultJobSubmitterInstance
}

func (d *DefaultJobSubmitter) SubmitJob(job *Job, workerChannel chan *Job) (any, *gogpt.Usage, error) {
	if job == nil {
		return blank, nil, errors.New("error, the job is nil")
	}

	workerChannel <- job

	result := <-job.Result
	close(job.Result)

	if job.Inputs.Def.Type == jsonSchema.Vector {
		if len(result.EmbeddingRes.Data) == 0 {
			return blank, nil, errors.New("embedding response data is empty")
		}

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
