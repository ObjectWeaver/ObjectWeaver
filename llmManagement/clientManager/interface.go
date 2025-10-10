package clientManager

import (
	"objectweaver/llmManagement"

	"github.com/sashabaranov/go-openai"
)

type ClientAdapter interface {
	Process(inputs *llmManagement.Inputs) (*openai.ChatCompletionResponse, error)
}
