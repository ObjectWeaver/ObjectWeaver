package clientManager

import (
	"github.com/sashabaranov/go-openai"
	"objectGeneration/llmManagement"
)

type ClientAdapter interface {
	Process(inputs *llmManagement.Inputs) (*openai.ChatCompletionResponse, error)
}
