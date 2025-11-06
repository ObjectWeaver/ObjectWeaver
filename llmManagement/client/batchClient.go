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
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// BatchStatus represents the possible states of a batch
type BatchStatus string

const (
	StatusValidating  BatchStatus = "validating"
	StatusFailed      BatchStatus = "failed"
	StatusInProgress  BatchStatus = "in_progress"
	StatusFinalizing  BatchStatus = "finalizing"
	StatusCompleted   BatchStatus = "completed"
	StatusExpired     BatchStatus = "expired"
	StatusCancelling  BatchStatus = "cancelling"
	StatusCancelled   BatchStatus = "cancelled"
)

// BatchEndpoint represents supported batch endpoints
type BatchEndpoint string

const (
	EndpointChatCompletions BatchEndpoint = "/v1/chat/completions"
	EndpointEmbeddings      BatchEndpoint = "/v1/embeddings"
	EndpointCompletions     BatchEndpoint = "/v1/completions"
	EndpointModerations     BatchEndpoint = "/v1/moderations"
	EndpointResponses       BatchEndpoint = "/v1/responses"
)

// BatchRequest represents a single request in the batch input file
type BatchRequest struct {
	CustomID string                 `json:"custom_id"`
	Method   string                 `json:"method"`
	URL      string                 `json:"url"`
	Body     map[string]interface{} `json:"body"`
}

// BatchResponse represents a single response in the batch output file
type BatchResponse struct {
	ID         string                 `json:"id"`
	CustomID   string                 `json:"custom_id"`
	Response   *ResponseData          `json:"response,omitempty"`
	Error      *BatchError            `json:"error,omitempty"`
}

// ResponseData contains the HTTP response details
type ResponseData struct {
	StatusCode int                    `json:"status_code"`
	RequestID  string                 `json:"request_id"`
	Body       map[string]interface{} `json:"body"`
}

// BatchError represents an error in batch processing
type BatchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RequestCounts tracks the progress of batch processing
type RequestCounts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// Batch represents a batch processing job
type Batch struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Endpoint          string                 `json:"endpoint"`
	Errors            interface{}            `json:"errors"`
	InputFileID       string                 `json:"input_file_id"`
	CompletionWindow  string                 `json:"completion_window"`
	Status            BatchStatus            `json:"status"`
	OutputFileID      *string                `json:"output_file_id"`
	ErrorFileID       *string                `json:"error_file_id"`
	CreatedAt         int64                  `json:"created_at"`
	InProgressAt      *int64                 `json:"in_progress_at"`
	ExpiresAt         int64                  `json:"expires_at"`
	CompletedAt       *int64                 `json:"completed_at"`
	FailedAt          *int64                 `json:"failed_at"`
	ExpiredAt         *int64                 `json:"expired_at"`
	CancellingAt      *int64                 `json:"cancelling_at"`
	CancelledAt       *int64                 `json:"cancelled_at"`
	RequestCounts     RequestCounts          `json:"request_counts"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// File represents an uploaded file
type File struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Bytes     int    `json:"bytes"`
	CreatedAt int64  `json:"created_at"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
}

// CreateBatchRequest represents the request to create a batch
type CreateBatchRequest struct {
	InputFileID      string                 `json:"input_file_id"`
	Endpoint         BatchEndpoint          `json:"endpoint"`
	CompletionWindow string                 `json:"completion_window"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ListBatchesResponse represents the response from listing batches
type ListBatchesResponse struct {
	Object  string  `json:"object"`
	Data    []Batch `json:"data"`
	FirstID string  `json:"first_id"`
	LastID  string  `json:"last_id"`
	HasMore bool    `json:"has_more"`
}

// BatchClient handles interactions with OpenAI's Batch API
type BatchClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	pollInterval time.Duration
}

// NewBatchClientWithHTTPClient creates a new batch API client with a custom HTTP client
func NewBatchClientWithHTTPClient(apiKey, baseURL string, pollInterval time.Duration, httpClient *http.Client) *BatchClient {
	return &BatchClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		pollInterval: pollInterval,
	}
}

// doRequest performs an HTTP request with authentication
func (c *BatchClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// UploadFile uploads a JSONL file for batch processing
func (c *BatchClient) UploadFile(ctx context.Context, filePath string) (*File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return c.UploadFileReader(ctx, file, filePath)
}

// UploadFileReader uploads a file from an io.Reader
func (c *BatchClient) UploadFileReader(ctx context.Context, reader io.Reader, filename string) (*File, error) {
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)

	// Create multipart form
	boundary := "----WebKitFormBoundary" + fmt.Sprintf("%d", time.Now().Unix())
	
	// Write purpose field
	fmt.Fprintf(writer, "--%s\r\n", boundary)
	fmt.Fprintf(writer, "Content-Disposition: form-data; name=\"purpose\"\r\n\r\n")
	fmt.Fprintf(writer, "batch\r\n")
	
	// Write file field
	fmt.Fprintf(writer, "--%s\r\n", boundary)
	fmt.Fprintf(writer, "Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n", filename)
	fmt.Fprintf(writer, "Content-Type: application/jsonl\r\n\r\n")
	writer.Flush()
	
	// Copy file content
	if _, err := io.Copy(&buffer, reader); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}
	
	// Write closing boundary
	fmt.Fprintf(writer, "\r\n--%s--\r\n", boundary)
	writer.Flush()

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/files", &buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var fileResp File
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &fileResp, nil
}

// CreateBatch creates a new batch processing job
func (c *BatchClient) CreateBatch(ctx context.Context, req CreateBatchRequest) (*Batch, error) {
	resp, err := c.doRequest(ctx, "POST", "/batches", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var batch Batch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &batch, nil
}

// GetBatch retrieves the status of a batch
func (c *BatchClient) GetBatch(ctx context.Context, batchID string) (*Batch, error) {
	resp, err := c.doRequest(ctx, "GET", "/batches/"+batchID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var batch Batch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &batch, nil
}

// CancelBatch cancels a batch that is in progress
func (c *BatchClient) CancelBatch(ctx context.Context, batchID string) (*Batch, error) {
	resp, err := c.doRequest(ctx, "POST", "/batches/"+batchID+"/cancel", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var batch Batch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &batch, nil
}

// ListBatches retrieves a list of batches
func (c *BatchClient) ListBatches(ctx context.Context, limit int, after string) (*ListBatchesResponse, error) {
	path := "/batches"
	if limit > 0 || after != "" {
		path += "?"
		if limit > 0 {
			path += fmt.Sprintf("limit=%d", limit)
		}
		if after != "" {
			if limit > 0 {
				path += "&"
			}
			path += fmt.Sprintf("after=%s", after)
		}
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listResp ListBatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &listResp, nil
}

// GetFileContent retrieves the content of a file
func (c *BatchClient) GetFileContent(ctx context.Context, fileID string) ([]byte, error) {
	resp, err := c.doRequest(ctx, "GET", "/files/"+fileID+"/content", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
}

// WaitForBatch polls the batch status until it reaches a terminal state
func (c *BatchClient) WaitForBatch(ctx context.Context, batchID string, pollIntervalOverride ...time.Duration) (*Batch, error) {
	interval := c.pollInterval
	if len(pollIntervalOverride) > 0 {
		interval = pollIntervalOverride[0]
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			batch, err := c.GetBatch(ctx, batchID)
			if err != nil {
				return nil, err
			}

			// Check for terminal states
			switch batch.Status {
			case StatusCompleted, StatusFailed, StatusExpired, StatusCancelled:
				return batch, nil
			}
		}
	}
}

// ReadBatchResponses reads and parses the batch output file
func (c *BatchClient) ReadBatchResponses(ctx context.Context, fileID string) ([]BatchResponse, error) {
	content, err := c.GetFileContent(ctx, fileID)
	if err != nil {
		return nil, err
	}

	var responses []BatchResponse
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		var resp BatchResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response line: %w", err)
		}
		responses = append(responses, resp)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading responses: %w", err)
	}

	return responses, nil
}

// CreateBatchRequestFile creates a JSONL file from batch requests
func CreateBatchRequestFile(requests []BatchRequest, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, req := range requests {
		if err := encoder.Encode(req); err != nil {
			return fmt.Errorf("failed to encode request: %w", err)
		}
	}

	return nil
}

// Helper function to create a batch request for chat completions
func NewChatCompletionBatchRequest(customID, model string, messages []map[string]interface{}, maxTokens int) BatchRequest {
	body := map[string]interface{}{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	
	return BatchRequest{
		CustomID: customID,
		Method:   "POST",
		URL:      string(EndpointChatCompletions),
		Body:     body,
	}
}

// Helper function to create a batch request for embeddings
func NewEmbeddingBatchRequest(customID, model string, input interface{}) BatchRequest {
	body := map[string]interface{}{
		"model": model,
		"input": input,
	}
	
	return BatchRequest{
		CustomID: customID,
		Method:   "POST",
		URL:      string(EndpointEmbeddings),
		Body:     body,
	}
}

// Helper function to create a batch request for moderations
func NewModerationBatchRequest(customID, model string, input interface{}) BatchRequest {
	body := map[string]interface{}{
		"model": model,
		"input": input,
	}
	
	return BatchRequest{
		CustomID: customID,
		Method:   "POST",
		URL:      string(EndpointModerations),
		Body:     body,
	}
}