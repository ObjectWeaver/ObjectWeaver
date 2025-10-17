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
	"objectweaver/llmManagement/modelConverter"
	"objectweaver/llmManagement/requestManagement"

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
	httpClient     *http.Client
	apiKey         string
	baseURL        string
	requestBuilder requestManagement.RequestBuilder
	modelConverter modelConverter.ModelConverter
}

// NewGeminiClientAdapter creates a new Gemini adapter that converts between
// OpenAI format (internal standard) and Gemini API format.
func NewGeminiClientAdapter(
	apiKey string,
	builder requestManagement.RequestBuilder,
	converter modelConverter.ModelConverter,
	httpClient *http.Client,
) *GeminiClientAdapter {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &GeminiClientAdapter{
		httpClient:     httpClient,
		apiKey:         apiKey,
		baseURL:        "https://generativelanguage.googleapis.com/v1beta",
		requestBuilder: builder,
		modelConverter: converter,
	}
}

// Process implements the ClientAdapter interface with OpenAI-to-Gemini conversion.
func (a *GeminiClientAdapter) Process(inputs *llmManagement.Inputs) (*openai.ChatCompletionResponse, error) {
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
	return a.convertFromGeminiFormat(resp)
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

func (a *GeminiClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	return nil, errors.New("Doesn't exist")
}