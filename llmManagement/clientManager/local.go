package clientManager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"objectGeneration/llmManagement"
	"objectGeneration/llmManagement/requestManagement"

	"github.com/sashabaranov/go-openai"
)

// LocalClientAdapter orchestrates the conversion and processing of requests
// to a standard HTTP client, mimicking a specific API like OpenAI's.
type LocalClientAdapter struct {
	client           *http.Client
	requestBuilder   requestManagement.RequestBuilder
	requestConverter requestManagement.RequestConverter
	targetURL        string
	authToken        string
}

// NewLocalClientAdapter creates a new adapter with necessary dependencies.
func NewLocalClientAdapter(
	url, token string,
	builder requestManagement.RequestBuilder,
	converter requestManagement.RequestConverter,
	httpClient *http.Client,
) *LocalClientAdapter {
	return &LocalClientAdapter{
		client:           httpClient,
		requestBuilder:   builder,
		requestConverter: converter,
		targetURL:        url,
		authToken:        token,
	}
}

// Process handles the end-to-end flow: builds a request from inputs,
// converts it to a standard HTTP request, sends it, and converts the
// response back to a typed struct.
func (h *LocalClientAdapter) Process(inputs *llmManagement.Inputs) (*openai.ChatCompletionResponse, error) {
	// 1. Build the API-specific request object (e.g., openai.ChatCompletionRequest).
	openAIReq, err := h.requestBuilder.BuildRequest(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// 2. Marshal the request object into a JSON byte slice for the HTTP body.
	reqBytes, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Debug: Log the outgoing request

	// 3. Create a standard *http.Request that the http.Client can send.
	httpReq, err := http.NewRequest(http.MethodPost, h.targetURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create new http request: %w", err)
	}

	// 4. Set necessary headers for the target API.
	httpReq.Header.Set("Content-Type", "application/json")
	if h.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.authToken)
	}

	// 5. Use the standard http.Client to send the request.
	response, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http client failed to process request: %w", err)
	}

	// 6. Convert the standard *http.Response back into the API-specific response type.
	chatResponse, err := h.requestConverter.ToChatCompletionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to convert http response: %w", err)
	}

	return &chatResponse, nil
}
