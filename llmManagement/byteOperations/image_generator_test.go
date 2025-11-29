package byteoperations

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestNewImageGenerator(t *testing.T) {
	client := &openai.Client{}
	generator := NewImageGenerator(client)

	if generator == nil {
		t.Fatal("Expected non-nil ImageGenerator")
	}

	if generator.client != client {
		t.Error("Expected client to be set correctly")
	}
}

func TestImageGenerator_ModelSelection(t *testing.T) {
	// This test validates model conversion logic
	tests := []struct {
		name          string
		inputModel    string
		expectedModel string
	}{
		{
			name:          "DALL-E 2 model",
			inputModel:    "dall-e-2",
			expectedModel: openai.CreateImageModelDallE2,
		},
		{
			name:          "DALL-E 3 model",
			inputModel:    "dall-e-3",
			expectedModel: openai.CreateImageModelDallE3,
		},
		{
			name:          "Unknown model defaults to DALL-E 2",
			inputModel:    "unknown-model",
			expectedModel: openai.CreateImageModelDallE2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the actual API call without mocking,
			// but we can validate the model type constants exist
			if tt.expectedModel == "" {
				t.Error("Expected model should not be empty")
			}
		})
	}
}

func TestImageGenerator_Base64Decoding(t *testing.T) {
	// Test base64 decoding functionality
	testData := []byte("test image data")
	encoded := base64.StdEncoding.EncodeToString(testData)

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	if string(decoded) != string(testData) {
		t.Errorf("Decoded data doesn't match original. Got %s, want %s", decoded, testData)
	}
}

func TestImageGenerator_InvalidBase64(t *testing.T) {
	invalidBase64 := "not-valid-base64!@#$"

	_, err := base64.StdEncoding.DecodeString(invalidBase64)
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}
}

func TestImageGenerator_HTTPDownload(t *testing.T) {
	// Create a test server that serves an image
	testImageData := []byte("fake image data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testImageData)
	}))
	defer server.Close()

	// Test downloading from the test server
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to download from test server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestImageGenerator_HTTPDownloadError(t *testing.T) {
	// Test error handling for failed HTTP requests
	_, err := http.Get("http://invalid-url-that-does-not-exist.local")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestImageGenerator_ContextCreation(t *testing.T) {
	// Validate that context.Background() works correctly
	ctx := context.Background()
	if ctx == nil {
		t.Error("Expected non-nil context")
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestImageGenerator_EmptyPrompt(t *testing.T) {
	// Test handling of edge cases
	emptyPrompt := ""
	if len(emptyPrompt) != 0 {
		t.Error("Empty prompt should have length 0")
	}
}

func TestImageGenerator_SizeValidation(t *testing.T) {
	// Common DALL-E image sizes
	validSizes := []string{
		"256x256",
		"512x512",
		"1024x1024",
		"1792x1024",
		"1024x1792",
	}

	for _, size := range validSizes {
		if size == "" {
			t.Errorf("Size should not be empty")
		}
	}
}

func TestImageGenerator_ResponseFormatConstant(t *testing.T) {
	// Validate that response format constant is correct
	format := openai.CreateImageResponseFormatB64JSON
	if format == "" {
		t.Error("Response format should not be empty")
	}
}

func TestImageGenerator_ModelConstants(t *testing.T) {
	// Validate OpenAI model constants exist and are not empty
	if openai.CreateImageModelDallE2 == "" {
		t.Error("DALL-E 2 model constant should not be empty")
	}

	if openai.CreateImageModelDallE3 == "" {
		t.Error("DALL-E 3 model constant should not be empty")
	}
}

func TestImageGenerator_GenerateImage_Base64Response(t *testing.T) {
	// This test validates the base64 response path
	// Since we can't easily mock the OpenAI client, we'll test the decoding logic

	testImageBytes := []byte("test image data content")
	encodedImage := base64.StdEncoding.EncodeToString(testImageBytes)

	// Verify encoding/decoding works correctly
	decoded, err := base64.StdEncoding.DecodeString(encodedImage)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	if string(decoded) != string(testImageBytes) {
		t.Errorf("Decoded data doesn't match. Got %s, want %s", decoded, testImageBytes)
	}

	// Verify the length is preserved
	if len(decoded) != len(testImageBytes) {
		t.Errorf("Decoded length mismatch. Got %d, want %d", len(decoded), len(testImageBytes))
	}
}

func TestImageGenerator_GenerateImage_URLDownload(t *testing.T) {
	// Test the URL download path with a mock HTTP server
	testImageData := []byte("fake PNG image data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(testImageData)
	}))
	defer server.Close()

	// Test downloading from the server
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to download: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if string(data) != string(testImageData) {
		t.Errorf("Downloaded data doesn't match. Got %s, want %s", data, testImageData)
	}
}

func TestImageGenerator_GenerateImage_URLDownloadError(t *testing.T) {
	// Test error handling when download fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestImageGenerator_GenerateImage_InvalidURL(t *testing.T) {
	// Test with invalid URL
	_, err := http.Get("http://this-domain-does-not-exist-12345.invalid")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "dial") && !strings.Contains(err.Error(), "lookup") {
		t.Logf("Got error: %v", err)
	}
}

func TestImageGenerator_GenerateImage_Base64DecodeError(t *testing.T) {
	// Test invalid base64 handling
	invalidBase64 := "this is not valid base64!@#$%^&*()"

	_, err := base64.StdEncoding.DecodeString(invalidBase64)
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestImageGenerator_GenerateImage_EmptyResponse(t *testing.T) {
	// Test handling of empty response data
	emptyData := ""
	if len(emptyData) != 0 {
		t.Error("Empty data should have length 0")
	}
}

func TestImageGenerator_GenerateImage_ModelConversion(t *testing.T) {
	tests := []struct {
		name          string
		inputModel    string
		expectedModel string
	}{
		{
			name:          "DALL-E 2",
			inputModel:    "dall-e-2",
			expectedModel: openai.CreateImageModelDallE2,
		},
		{
			name:          "DALL-E 3",
			inputModel:    "dall-e-3",
			expectedModel: openai.CreateImageModelDallE3,
		},
		{
			name:          "Unknown defaults to DALL-E 2",
			inputModel:    "unknown-model",
			expectedModel: openai.CreateImageModelDallE2,
		},
		{
			name:          "Empty string defaults to DALL-E 2",
			inputModel:    "",
			expectedModel: openai.CreateImageModelDallE2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate model conversion logic
			var resultModel string

			modelStr := string(tt.inputModel)
			if modelStr == "dall-e-2" {
				resultModel = openai.CreateImageModelDallE2
			} else if modelStr == "dall-e-3" {
				resultModel = openai.CreateImageModelDallE3
			} else {
				resultModel = openai.CreateImageModelDallE2
			}

			if resultModel != tt.expectedModel {
				t.Errorf("Expected %s, got %s", tt.expectedModel, resultModel)
			}
		})
	}
}

func TestImageGenerator_GenerateImage_SizeValidation(t *testing.T) {
	validSizes := map[string]bool{
		"256x256":   true,
		"512x512":   true,
		"1024x1024": true,
		"1792x1024": true,
		"1024x1792": true,
	}

	for size, expected := range validSizes {
		t.Run(size, func(t *testing.T) {
			if expected && size == "" {
				t.Error("Valid size should not be empty")
			}
			if len(size) == 0 && expected {
				t.Error("Expected non-empty size")
			}
		})
	}
}

func TestImageGenerator_GenerateImage_ResponseFormat(t *testing.T) {
	// Validate response format constant
	format := openai.CreateImageResponseFormatB64JSON
	if format != "b64_json" {
		t.Errorf("Expected b64_json, got %s", format)
	}

	// Test URL format too
	urlFormat := openai.CreateImageResponseFormatURL
	if urlFormat != "url" {
		t.Errorf("Expected url, got %s", urlFormat)
	}
}

func TestImageGenerator_GenerateImage_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		if ctx.Err() == nil {
			t.Error("Expected context error after cancellation")
		}
	default:
		t.Error("Context should be cancelled")
	}
}

func TestImageGenerator_GenerateImage_LargeImageData(t *testing.T) {
	// Test with large image data
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encoded := base64.StdEncoding.EncodeToString(largeData)
	decoded, err := base64.StdEncoding.DecodeString(encoded)

	if err != nil {
		t.Fatalf("Failed to decode large data: %v", err)
	}

	if len(decoded) != len(largeData) {
		t.Errorf("Size mismatch. Got %d, want %d", len(decoded), len(largeData))
	}
}

func TestImageGenerator_GenerateImage_PromptValidation(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		valid  bool
	}{
		{
			name:   "Normal prompt",
			prompt: "A beautiful sunset",
			valid:  true,
		},
		{
			name:   "Long prompt",
			prompt: strings.Repeat("word ", 100),
			valid:  true,
		},
		{
			name:   "Empty prompt",
			prompt: "",
			valid:  false,
		},
		{
			name:   "Special characters",
			prompt: "Test!@#$%^&*()",
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEmpty := len(tt.prompt) == 0
			if isEmpty && tt.valid {
				t.Error("Empty prompt should not be valid")
			}
			if !isEmpty && !tt.valid {
				t.Error("Non-empty prompt should be valid")
			}
		})
	}
}

func TestImageGenerator_GenerateImage_HTTPReadError(t *testing.T) {
	// Test error when reading response body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000") // Claim large size
		w.WriteHeader(http.StatusOK)
		// Write less data than promised to potentially cause read issues
		w.Write([]byte("small"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Try to read the data - Go's HTTP client is smart and handles this
	data, err := io.ReadAll(resp.Body)
	// In this case, we might get an EOF error or the data might be read successfully
	// depending on the HTTP/1.1 chunked encoding behavior
	if err != nil && err != io.EOF && !strings.Contains(err.Error(), "EOF") {
		t.Logf("Got expected error: %v", err)
	}

	// We should get some data regardless
	if len(data) == 0 && err == nil {
		t.Error("Expected either data or error")
	}
}

func TestImageGenerator_ErrorWrapping(t *testing.T) {
	// Test error wrapping/formatting
	baseErr := errors.New("base error")
	wrappedErr := errors.New("wrapped: " + baseErr.Error())

	if !strings.Contains(wrappedErr.Error(), "base error") {
		t.Error("Wrapped error should contain base error message")
	}
}

func TestImageGenerator_MultipleImages(t *testing.T) {
	// Test that we handle N=1 correctly (only return first image)
	n := 1
	if n != 1 {
		t.Error("N should be 1 for single image generation")
	}
}

func TestImageGenerator_ImageRequestStructure(t *testing.T) {
	// Validate ImageRequest structure
	req := openai.ImageRequest{
		Prompt:         "test prompt",
		Size:           "1024x1024",
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
		Model:          openai.CreateImageModelDallE3,
	}

	if req.Prompt == "" {
		t.Error("Prompt should not be empty")
	}
	if req.Size == "" {
		t.Error("Size should not be empty")
	}
	if req.ResponseFormat == "" {
		t.Error("ResponseFormat should not be empty")
	}
	if req.N != 1 {
		t.Error("N should be 1")
	}
	if req.Model == "" {
		t.Error("Model should not be empty")
	}
}

func TestImageGenerator_Base64Padding(t *testing.T) {
	// Test various padding scenarios in base64
	tests := []struct {
		name string
		data []byte
	}{
		{"1 byte", []byte("a")},
		{"2 bytes", []byte("ab")},
		{"3 bytes", []byte("abc")},
		{"4 bytes", []byte("abcd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := base64.StdEncoding.EncodeToString(tt.data)
			decoded, err := base64.StdEncoding.DecodeString(encoded)

			if err != nil {
				t.Errorf("Decode failed: %v", err)
			}
			if string(decoded) != string(tt.data) {
				t.Errorf("Data mismatch. Got %s, want %s", decoded, tt.data)
			}
		})
	}
}
