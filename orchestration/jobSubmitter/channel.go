package jobSubmitter

import (
	"context"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/LLM"
	"objectweaver/llmManagement/domain"

	"objectweaver/jsonSchema"

	"github.com/sashabaranov/go-openai"
)

type ChannelJobSubmitter struct{}

func (js *ChannelJobSubmitter) SubmitJob(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *openai.Usage, error) {
	input := llmManagement.Inputs{
		Ctx:          ctx,
		Prompt:       newPrompt,
		SystemPrompt: systemPrompt,
		Def:          def,
		OutStream:    outStream,
	}

	job := &LLM.Job{
		Inputs: &input,
		Result: make(chan *domain.JobResult, 1),
		Tokens: 0,
	}

	if input.Def.SendImage == nil {
		input.Def.SendImage = &jsonSchema.SendImage{}
		input.Def.SendImage.ImagesData = nil
	}

	submitter := LLM.JobSubmitterFactory(LLM.DefaultSubmitter)
	return submitter.SubmitJob(job, LLM.WorkerChannel)
}
