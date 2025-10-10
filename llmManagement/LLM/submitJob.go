package LLM

import (
	"errors"

	gogpt "github.com/sashabaranov/go-openai"
)

const blank = ""

type VariedJobSubmitter struct{}

func NewVariedJobSubmitter() *VariedJobSubmitter {
	return &VariedJobSubmitter{}
}

func (v *VariedJobSubmitter) SubmitJob(job *Job, workerChannel chan *Job) (string, *gogpt.Usage, error) {
	select {
	case WorkerChannel <- job:
	default:
		return v.SubmitJob(job, workerChannel)
	}

	result := <-job.Result
	close(job.Result)

	return validateResult(result)
}

type DefaultJobSubmitter struct{}

func NewDefaultJobSubmitter() *DefaultJobSubmitter {
	return &DefaultJobSubmitter{}
}

func (d *DefaultJobSubmitter) SubmitJob(job *Job, workerChannel chan *Job) (string, *gogpt.Usage, error) {
	if job == nil {
		return blank, nil, errors.New("error, the job is nil")
	}

	workerChannel <- job

	result := <-job.Result
	close(job.Result)

	return validateResult(result)
}

func validateResult(result *gogpt.ChatCompletionResponse) (string, *gogpt.Usage, error) {
	if result == nil {
		return blank, nil, errors.New("error, the returned result is nil")
	}
	if len(result.Choices) < 1 {
		return blank, nil, errors.New("error, the returned result is empty")
	}

	return result.Choices[0].Message.Content, &result.Usage, nil
}
