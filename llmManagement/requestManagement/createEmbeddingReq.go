package requestManagement

import (
	"objectweaver/llmManagement"

	"github.com/sashabaranov/go-openai"
)

type EmbeddingRequestBuilder interface {
	BuildRequest(inputs *llmManagement.Inputs) (openai.EmbeddingRequest, error)
}

type embeddingOpenAIReqBuilder struct{}

func NewEmbeddingOpenAIReqBuilder() EmbeddingRequestBuilder {
	return &embeddingOpenAIReqBuilder{}
}

func (b *embeddingOpenAIReqBuilder) BuildRequest(inputs *llmManagement.Inputs) (openai.EmbeddingRequest, error) {
	// Implementation for building embedding request goes here

	req := openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(inputs.Def.Model),
		Input: inputs.Prompt,
	}

	return req, nil
}
