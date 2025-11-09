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
package clientManager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/domain"
	"objectweaver/llmManagement/modelConverter"
	"objectweaver/llmManagement/requestManagement"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GeminiClientAdapter converts OpenAI-format requests to Gemini API format.
// All requests are built using OpenAI format and then converted for Gemini.
// Responses are converted back to OpenAI format for consistency.
type GeminiClientAdapter struct {
	httpClient              *http.Client
	apiKey                  string
	baseURL                 string
	requestBuilder          requestManagement.RequestBuilder
	embeddingRequestBuilder requestManagement.EmbeddingRequestBuilder
	modelConverter          modelConverter.ModelConverter
}

// NewGeminiClientAdapter creates a new Gemini adapter that converts between
// OpenAI format (internal standard) and Gemini API format.
func NewGeminiClientAdapter(
	apiKey string,
	builder requestManagement.RequestBuilder,
	embeddingBuilder requestManagement.EmbeddingRequestBuilder,
	converter modelConverter.ModelConverter,
	httpClient *http.Client,
) *GeminiClientAdapter {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &GeminiClientAdapter{
		httpClient:              httpClient,
		apiKey:                  apiKey,
		baseURL:                 "https://generativelanguage.googleapis.com/v1beta",
		requestBuilder:          builder,
		embeddingRequestBuilder: embeddingBuilder,
		modelConverter:          converter,
	}
}

// Process implements the ClientAdapter interface with OpenAI-to-Gemini conversion.
func (a *GeminiClientAdapter) Process(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	// Check if this is an embedding request
	if inputs.Def != nil && inputs.Def.Type == jsonSchema.Vector {
		return a.processEmbedding(inputs)
	}
	return a.processChat(inputs)
}

// processChat handles chat completion requests
func (a *GeminiClientAdapter) processChat(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	// 1. Build request in OpenAI format (our standard)
	openaiReq, err := a.requestBuilder.BuildRequest(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// 2. Convert OpenAI format to Gemini format
	geminiReq := a.convertToGeminiFormat(openaiReq)

	// 3. Send to Gemini API
	reqBytes, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	// Use the model from the built request (already converted by request builder)
	// openaiReq.Model is already in Gemini format (e.g., "gemini-2.0-flash-lite")
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		a.baseURL, openaiReq.Model, a.apiKey)

	log.Printf("[Gemini DEBUG] Using model: %s", openaiReq.Model)
	log.Printf("[Gemini DEBUG] API URL: %s", url)

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini api request failed: %w", err)
	}
	defer resp.Body.Close()

	// 4. Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, string(body))
	}

	// 5. Convert Gemini response back to OpenAI format
	res, err := a.convertFromGeminiFormat(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert gemini response: %w", err)
	}

	return domain.CreateJobResult(res, nil), nil
}

// processEmbedding handles embedding requests
func (a *GeminiClientAdapter) processEmbedding(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	// 1. Build embedding request in OpenAI format
	openaiReq, err := a.embeddingRequestBuilder.BuildRequest(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to build embedding request: %w", err)
	}

	// 2. Convert OpenAI format to Gemini embedding format
	geminiReq := a.convertToGeminiEmbeddingFormat(openaiReq)

	// 3. Send to Gemini API
	reqBytes, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini embedding request: %w", err)
	}

	// Gemini uses embedContent endpoint for embeddings
	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s",
		a.baseURL, openaiReq.Model, a.apiKey)

	log.Printf("[Gemini DEBUG] Using embedding model: %s", openaiReq.Model)
	log.Printf("[Gemini DEBUG] Embedding API URL: %s", url)

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini embedding api request failed: %w", err)
	}
	defer resp.Body.Close()

	// 4. Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini embedding api error (status %d): %s", resp.StatusCode, string(body))
	}

	// 5. Convert Gemini embedding response back to OpenAI format
	res, err := a.convertFromGeminiEmbeddingFormat(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert gemini embedding response: %w", err)
	}

	return domain.CreateJobResult(nil, res), nil
}

// convertToGeminiFormat transforms an OpenAI ChatCompletionRequest to Gemini's format.
func (a *GeminiClientAdapter) convertToGeminiFormat(req openai.ChatCompletionRequest) map[string]interface{} {
	contents := make([]map[string]interface{}, 0)

	// Convert OpenAI messages to Gemini contents
	for _, msg := range req.Messages {
		// Map OpenAI roles to Gemini roles
		role := "user"
		if msg.Role == openai.ChatMessageRoleAssistant || msg.Role == "model" {
			role = "model"
		}

		// Handle both simple content and multi-content messages
		parts := make([]map[string]interface{}, 0)

		if msg.Content != "" {
			parts = append(parts, map[string]interface{}{
				"text": msg.Content,
			})
		}

		// Handle multi-content (images, etc.)
		if len(msg.MultiContent) > 0 {
			for _, part := range msg.MultiContent {
				switch part.Type {
				case openai.ChatMessagePartTypeText:
					parts = append(parts, map[string]interface{}{
						"text": part.Text,
					})
				case openai.ChatMessagePartTypeImageURL:
					if part.ImageURL != nil {
						// Extract base64 data from data URL (remove "data:image/xxx;base64," prefix)
						imageData := part.ImageURL.URL
						mimeType := "image/jpeg"

						log.Printf("[Gemini DEBUG] Original imageData length: %d", len(imageData))
						log.Printf("[Gemini DEBUG] First 100 chars: %s", imageData[:min(100, len(imageData))])

						// Check if it's a data URL and extract the base64 part
						if len(imageData) > 0 {
							// Parse data URL format: "data:image/jpeg;base64,BASE64DATA"
							if idx := bytes.Index([]byte(imageData), []byte(";base64,")); idx != -1 {
								// Extract mime type
								if bytes.HasPrefix([]byte(imageData), []byte("data:")) {
									mimeType = imageData[5:idx] // Extract "image/jpeg" from "data:image/jpeg"
								}
								// Extract base64 data (skip ";base64,")
								imageData = imageData[idx+8:]
								log.Printf("[Gemini DEBUG] After extraction - mimeType: %s, data length: %d", mimeType, len(imageData))
								log.Printf("[Gemini DEBUG] First 50 chars of base64: %s", imageData[:min(50, len(imageData))])
							} else {
								log.Printf("[Gemini DEBUG] No ';base64,' found in imageData, using as-is")
							}
						}

						// Gemini supports inline data in base64 format
						parts = append(parts, map[string]interface{}{
							"inline_data": map[string]interface{}{
								"mime_type": mimeType,
								"data":      imageData, // Pure base64, no prefix
							},
						})
					}
				}
			}
		}

		if len(parts) > 0 {
			content := map[string]interface{}{
				"role":  role,
				"parts": parts,
			}
			contents = append(contents, content)
		}
	}

	// Build generation config
	generationConfig := map[string]interface{}{}

	if req.Temperature > 0 {
		generationConfig["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		generationConfig["topP"] = req.TopP
	}
	if req.MaxTokens > 0 {
		generationConfig["maxOutputTokens"] = req.MaxTokens
	}

	geminiReq := map[string]interface{}{
		"contents": contents,
	}

	if len(generationConfig) > 0 {
		geminiReq["generationConfig"] = generationConfig
	}

	return geminiReq
}

// convertFromGeminiFormat transforms a Gemini API response to OpenAI format.
func (a *GeminiClientAdapter) convertFromGeminiFormat(resp *http.Response) (*openai.ChatCompletionResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response: %w", err)
	}

	// Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	// Convert to OpenAI format
	var content string
	var finishReason string

	if len(geminiResp.Candidates) > 0 {
		candidate := geminiResp.Candidates[0]

		// Concatenate all text parts
		for _, part := range candidate.Content.Parts {
			content += part.Text
		}

		// Map Gemini finish reasons to OpenAI format
		switch candidate.FinishReason {
		case "STOP":
			finishReason = "stop"
		case "MAX_TOKENS":
			finishReason = "length"
		case "SAFETY":
			finishReason = "content_filter"
		default:
			finishReason = "stop"
		}
	}

	openaiResp := &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: content,
				},
				FinishReason: openai.FinishReason(finishReason),
			},
		},
		Usage: openai.Usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}

	return openaiResp, nil
}

// convertToGeminiEmbeddingFormat transforms an OpenAI EmbeddingRequest to Gemini's format.
func (a *GeminiClientAdapter) convertToGeminiEmbeddingFormat(req openai.EmbeddingRequest) map[string]interface{} {
	// Convert input to string if it's not already
	var text string
	switch v := req.Input.(type) {
	case string:
		text = v
	case []string:
		if len(v) > 0 {
			text = v[0] // Gemini embedContent typically processes one text at a time
		}
	default:
		text = fmt.Sprintf("%v", v)
	}

	geminiReq := map[string]interface{}{
		"content": map[string]interface{}{
			"parts": []map[string]interface{}{
				{
					"text": text,
				},
			},
		},
	}

	return geminiReq
}

// convertFromGeminiEmbeddingFormat transforms a Gemini embedding response to OpenAI format.
func (a *GeminiClientAdapter) convertFromGeminiEmbeddingFormat(resp *http.Response) (*openai.EmbeddingResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini embedding response: %w", err)
	}

	// Gemini embedding response structure
	var geminiResp struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini embedding response: %w", err)
	}

	// Convert to OpenAI format
	openaiResp := &openai.EmbeddingResponse{
		Object: "list",
		Data: []openai.Embedding{
			{
				Object:    "embedding",
				Embedding: geminiResp.Embedding.Values,
				Index:     0,
			},
		},
		Model: openai.AdaEmbeddingV2, // Default model identifier
		Usage: openai.Usage{
			PromptTokens: 0, // Gemini doesn't provide token usage in embedding response
			TotalTokens:  0,
		},
	}

	return openaiResp, nil
}

func (a *GeminiClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	return nil, errors.New("doesn't exist")
}
