package domain

import "github.com/sashabaranov/go-openai"

type JobResult struct {
	ChatRes      *openai.ChatCompletionResponse
	EmbeddingRes *openai.EmbeddingResponse
}

func CreateJobResult(chatRes *openai.ChatCompletionResponse, embeddingRes *openai.EmbeddingResponse) *JobResult {
	return &JobResult{
		ChatRes:      chatRes,
		EmbeddingRes: embeddingRes,
	}
}