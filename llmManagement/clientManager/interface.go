package clientManager

import (
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/domain"

	"github.com/sashabaranov/go-openai"
)

type ClientAdapter interface {
	Process(inputs *llmManagement.Inputs) (*domain.JobResult, error)
	//will need some proper structure etc
	ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error)
}
