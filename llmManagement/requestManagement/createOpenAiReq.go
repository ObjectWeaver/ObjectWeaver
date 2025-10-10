package requestManagement

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"objectGeneration/llmManagement"
	"objectGeneration/llmManagement/modelConverter"
	"strings"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
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
	// 1. Extract parameters from the Inputs struct
	prompt := inputs.Prompt
	systemPrompt := inputs.SystemPrompt
	model := b.modelConverter.Convert(inputs.Def.Model) // Cast ModelType to string

	temp := float32(inputs.Def.Temp)
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

	if imagesData != nil && model != string(jsonSchema.Gpt3) {
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
	if isReasoningModel(model) {
		req := gogpt.ChatCompletionRequest{
			Messages:        messages,
			ReasoningEffort: "medium",
			Model:           model,
			Stream:          isStream, // Set stream flag
		}

		if isStream {
			// Stream settings from original stream function
			req.Temperature = 0.0
			req.TopP = 0.0
			var streamSeed int = 0
			req.Seed = &streamSeed
		} else {
			// Non-stream settings from original non-stream function
			// Note: Original reasoning request had Temp, TopP, and Seed commented out.
			// Re-enable them here if needed.
			// req.Temperature = temp
			// req.TopP = 0.0
			// var seed int = 51635473
			// req.Seed = &seed
		}
		return req, nil
	}

	// 5. Handle Standard Models logic
	var seed int = 51635473
	req := gogpt.ChatCompletionRequest{
		Messages: messages,
		Model:    model,
		Stream:   isStream, // Set stream flag
	}

	if isStream {
		// Stream settings
		req.Temperature = 0.0
		// Note: original stream func didn't set TopP or Seed for non-reasoning
	} else {
		// Non-stream settings
		req.Temperature = temp
		req.TopP = 0.0
		req.Seed = &seed
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

// isReasoningModel checks if the model string matches known reasoning model identifiers
func isReasoningModel(model string) bool {
	switch model {
	case "o3-mini-2025-01-31":
		return true
	case "o3-min":
		return true
	case "o4-mini-2025-04-16":
		return true
	default:
		return false
	}
}
