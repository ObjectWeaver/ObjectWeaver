// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://objectweaver.dev/licensing/server-side-public-license>.
package requestManagement

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/modelConverter"
	"os"
	"strconv"
	"strings"

	gogpt "github.com/sashabaranov/go-openai"
)

// --- Helper Functions ---

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- Interface and Struct Definitions ---

// RequestBuilder defines the interface for building a standard request
// that conforms to the OpenAI API standard.
type RequestBuilder interface {
	// BuildRequest constructs a ChatCompletionRequest from the given inputs.
	BuildRequest(inputs *llmManagement.Inputs) (gogpt.ChatCompletionRequest, error)
	//BuildBatchRequest(inputs []*llmManagement.Inputs) (gogpt.ChatCompletionRequest, error)
}

// defaultOpenAIReqBuilder is the concrete implementation of RequestBuilder
// for the standard OpenAI API.
type defaultOpenAIReqBuilder struct {
	modelConverter modelConverter.ModelConverter
}

// NewDefaultOpenAIReqBuilder initializes a new request builder that implements
// the RequestBuilder interface.
func NewDefaultOpenAIReqBuilder(modelConverter modelConverter.ModelConverter) RequestBuilder {
	return &defaultOpenAIReqBuilder{
		modelConverter: modelConverter,
	}
}

// --- Interface Implementation ---

// BuildRequest constructs the OpenAI request based on the inputs.
// It merges and adapts the logic from the original CreateOpenAIRequest and CreateOpenAIStreamRequest functions.
func (b *defaultOpenAIReqBuilder) BuildRequest(inputs *llmManagement.Inputs) (gogpt.ChatCompletionRequest, error) {
	// 1. Validate inputs
	if inputs == nil {
		return gogpt.ChatCompletionRequest{}, fmt.Errorf("inputs cannot be nil")
	}
	if inputs.Def == nil {
		return gogpt.ChatCompletionRequest{}, fmt.Errorf("inputs.Def cannot be nil")
	}

	// 2. Extract parameters from the Inputs struct
	prompt := inputs.Prompt
	systemPrompt := inputs.SystemPrompt
	model := b.modelConverter.Convert(inputs.Def.Model) // Cast ModelType to string

	// Set defaults for ModelConfig fields
	var temp float32 = 1.0 // Default temperature
	var topP float32 = 1.0
	var topLogProbs int = 0
	var presencePenalty float32 = 0.0
	var frequencyPenalty float32 = 0.0
	var maxCompletionTokens int = 0
	var logProbs bool = false
	var reasoningEffort string = ""
	var chatTemplateKwargs map[string]interface{} = nil

	if inputs.Def.ModelConfig != nil {
		temp = float32(inputs.Def.ModelConfig.Temperature)
		topP = inputs.Def.ModelConfig.TopP
		topLogProbs = inputs.Def.ModelConfig.TopLogProbs
		presencePenalty = inputs.Def.ModelConfig.PresencePenalty
		frequencyPenalty = inputs.Def.ModelConfig.FrequencyPenalty
		maxCompletionTokens = inputs.Def.ModelConfig.MaxCompletionTokens
		logProbs = inputs.Def.ModelConfig.LogProbs
		reasoningEffort = inputs.Def.ModelConfig.ReasoningEffort
		chatTemplateKwargs = inputs.Def.ModelConfig.ChatTemplateKwargs
	}

	isStream := inputs.Def.Stream

	// 2. Build the base messages
	var message gogpt.ChatCompletionMessage
	messages := []gogpt.ChatCompletionMessage{
		{
			Role:    gogpt.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// 3. Handle image data
	var imagesData [][]byte
	if inputs.Def.SendImage != nil {
		imagesData = inputs.Def.SendImage.ImagesData
		log.Printf("[CreateOpenAIReq DEBUG] Found %d images in SendImage", len(imagesData))
		for i, img := range imagesData {
			log.Printf("[CreateOpenAIReq DEBUG] Image %d: %d bytes", i, len(img))
			if len(img) > 0 {
				log.Printf("[CreateOpenAIReq DEBUG] Image %d first 20 bytes: %v", i, img[:min(20, len(img))])
			}
		}
	} else {
		log.Printf("[CreateOpenAIReq DEBUG] No SendImage data found")
	}

	var url *gogpt.ChatMessageImageURL
	var multiContent []gogpt.ChatMessagePart

	if imagesData != nil && model != "gpt-3.5-turbo-0613" && model != "gpt-4-0613" {
		messages = append(messages, promptWriter(prompt)) // Use helper
		for _, image := range imagesData {
			mimeType := detectMimeType(image)                 // Use helper
			base64DataURL := toBase64DataURL(image, mimeType) // Use helper
			url = &gogpt.ChatMessageImageURL{
				URL:    base64DataURL,
				Detail: gogpt.ImageURLDetailAuto,
			}
			multiContent = []gogpt.ChatMessagePart{
				{
					Type:     gogpt.ChatMessagePartTypeImageURL,
					ImageURL: url,
				},
			}
			message = gogpt.ChatCompletionMessage{
				Role:         gogpt.ChatMessageRoleUser,
				MultiContent: multiContent,
			}
			messages = append(messages, message)
		}
	} else {
		// Default text-only message
		message = gogpt.ChatCompletionMessage{
			Role:    gogpt.ChatMessageRoleUser,
			Content: prompt,
		}
		messages = append(messages, message)
	}

	// 4. Handle Reasoning Models logic

	// 5. Handle Standard Models logic
	var seed *int
	if inputs.Def.ModelConfig != nil && inputs.Def.ModelConfig.Seed != nil {
		seed = inputs.Def.ModelConfig.Seed
	} else {
		seedEnv := os.Getenv("LLM_SEED")

		strconvSeed, err := strconv.Atoi(seedEnv)
		if err == nil {
			seed = &strconvSeed
		} else {
			defaultSeed := 51635473
			seed = &defaultSeed
		}
	}

	req := gogpt.ChatCompletionRequest{
		Messages:            messages,
		Model:               model,
		Stream:              isStream,
		Temperature:         temp,
		TopP:                topP,
		Seed:                seed,
		TopLogProbs:         topLogProbs,
		PresencePenalty:     presencePenalty,
		FrequencyPenalty:    frequencyPenalty,
		MaxCompletionTokens: maxCompletionTokens,
		LogProbs:            logProbs,
		ReasoningEffort:     reasoningEffort,
		ChatTemplateKwargs:  chatTemplateKwargs,
	}

	return req, nil
}

// --- Private Helper Functions ---
// (These are the original functions from your prompt, kept as private helpers)

// Convert a byte array into a base64 data URL
func toBase64DataURL(imageData []byte, mimeType string) string {
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)
}

// Function to determine MIME type
func detectMimeType(imageData []byte) string {
	mimeType := http.DetectContentType(imageData)
	log.Println("the mime type found: ", mimeType)
	if !strings.HasPrefix(mimeType, "image/") {
		log.Println("returning the value of image/jpeg")
		return "image/jpeg"
	}
	return mimeType
}

// promptWriter formats a prompt as a multi-part message (for consistency with image messages)
func promptWriter(prompt string) gogpt.ChatCompletionMessage {
	multiContent := []gogpt.ChatMessagePart{
		{
			Type: gogpt.ChatMessagePartTypeText,
			Text: prompt,
		},
	}
	return gogpt.ChatCompletionMessage{
		Role:         gogpt.ChatMessageRoleUser,
		MultiContent: multiContent,
	}
}

