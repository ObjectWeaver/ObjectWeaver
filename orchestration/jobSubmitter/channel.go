package jobSubmitter

import (
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/LLM"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

type ChannelJobSubmitter struct{}

func (js *ChannelJobSubmitter) SubmitJob(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (string, *openai.Usage, error) {
	input := llmManagement.Inputs{
		Prompt:       newPrompt,
		SystemPrompt: systemPrompt,
		Def:          def,
		OutStream:    outStream,
	}

	job := &LLM.Job{
		Inputs: &input,
		Result: make(chan *openai.ChatCompletionResponse),
		Tokens: 0,
	}

	if input.Def.SendImage == nil {
		input.Def.SendImage = &jsonSchema.SendImage{}
		input.Def.SendImage.ImagesData = nil
	}

	submitter := LLM.JobSubmitterFactory(LLM.DefaultSubmitter)
	return submitter.SubmitJob(job, LLM.WorkerChannel)
}
