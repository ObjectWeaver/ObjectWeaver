package clientManager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/domain"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/requestManagement"

	"github.com/sashabaranov/go-openai"
)

// LocalClientAdapter orchestrates the conversion and processing of requests
// to a standard HTTP client, mimicking a specific API like OpenAI's.
type LocalClientAdapter struct {
	client                  *http.Client
	requestBuilder          requestManagement.RequestBuilder
	embeddingRequestBuilder requestManagement.EmbeddingRequestBuilder
	targetURL               string
	authToken               string
}

// NewLocalClientAdapter creates a new adapter with necessary dependencies.
func NewLocalClientAdapter(
	url, token string,
	builder requestManagement.RequestBuilder,
	embeddingBuilder requestManagement.EmbeddingRequestBuilder,
	httpClient *http.Client,
) *LocalClientAdapter {
	return &LocalClientAdapter{
		client:                  httpClient,
		requestBuilder:          builder,
		embeddingRequestBuilder: embeddingBuilder,
		targetURL:               url,
		authToken:               token,
	}
}

// Process handles the end-to-end flow: builds a request from inputs,
// converts it to a standard HTTP request, sends it, and converts the
// response back to a typed struct.
func (h *LocalClientAdapter) Process(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	// Check if this is an embedding request
	if inputs.Def != nil && inputs.Def.Type == "vector" {
		return h.processEmbedding(inputs)
	}
	return h.processChat(inputs)
}

// processChat handles chat completion requests
func (h *LocalClientAdapter) processChat(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
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

	// 3. Create a standard *http.Request that the http.Client can send.
	// Use context from inputs if available, otherwise use background
	ctx := inputs.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.targetURL, bytes.NewBuffer(reqBytes))
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
	defer func() {
		if response.Body != nil {
			// Drain any remaining data to enable connection reuse
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
	}()

	// 6. Read response body once
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 7. Check HTTP status code before attempting to unmarshal
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat api error (status %d): %s", response.StatusCode, string(body))
	}

	// 8. Directly unmarshal the OpenAI-formatted response
	var chatResponse openai.ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat response: %w. Response body: %s", err, string(body))
	}

	return domain.CreateJobResult(&chatResponse, nil), nil
}

// processEmbedding handles embedding requests
func (h *LocalClientAdapter) processEmbedding(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	// 1. Build the embedding request
	embeddingReq, err := h.embeddingRequestBuilder.BuildRequest(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to build embedding request: %w", err)
	}

	// 2. Marshal the request object into a JSON byte slice for the HTTP body.
	reqBytes, err := json.Marshal(embeddingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request body: %w", err)
	}

	// 3. Create a standard *http.Request that the http.Client can send.
	// For local/OpenAI-compatible APIs, embeddings typically use /v1/embeddings endpoint
	embeddingURL := h.targetURL
	// If the target URL is for chat completions, we need to adjust it for embeddings
	// This assumes the local API follows OpenAI's URL structure
	if len(embeddingURL) > 0 && (embeddingURL[len(embeddingURL)-len("/chat/completions"):] == "/chat/completions" ||
		embeddingURL[len(embeddingURL)-len("/completions"):] == "/completions") {
		// Replace the endpoint with embeddings
		embeddingURL = embeddingURL[:len(embeddingURL)-len("/chat/completions")] + "/embeddings"
	}

	// Use context from inputs if available, otherwise use background
	ctx := inputs.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, embeddingURL, bytes.NewBuffer(reqBytes))
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
		return nil, fmt.Errorf("http client failed to process embedding request: %w", err)
	}
	// CRITICAL: Ensure body is ALWAYS closed and drained for connection reuse
	defer func() {
		if response.Body != nil {
			// Drain any remaining data to enable connection reuse
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
	}()

	// 6. Handle non-200 status codes
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("embedding api error (status %d): %s", response.StatusCode, string(body))
	}

	// 7. Parse the embedding response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response: %w", err)
	}

	var embeddingResp openai.EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding response: %w", err)
	}

	return domain.CreateJobResult(nil, &embeddingResp), nil
}

func (a *LocalClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	return nil, errors.New("doesn't exist")
}
