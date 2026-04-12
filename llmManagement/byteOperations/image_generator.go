package byteoperations

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"github.com/ObjectWeaver/ObjectWeaver/logger"

	"github.com/sashabaranov/go-openai"
)

// ImageGenerator handles image generation using OpenAI DALL-E models
type ImageGenerator struct {
	client *openai.Client
}

// NewImageGenerator creates a new image generator
func NewImageGenerator(client *openai.Client) *ImageGenerator {
	return &ImageGenerator{
		client: client,
	}
}

// GenerateImage creates an image using DALL-E and returns the raw bytes
func (g *ImageGenerator) GenerateImage(prompt string, model string, size string) ([]byte, error) {
	logger.Printf("[ImageGenerator] Generating image with model: %s, size: %s", model, size)
	logger.Printf("[ImageGenerator] Prompt: %s", prompt)

	// Build the image request
	req := openai.ImageRequest{
		Prompt:         prompt,
		Size:           size,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON, // Get base64 response
		N:              1,
	}

	// Set model based on configuration
	// Convert ModelType to string and check
	modelStr := string(model)
	if modelStr == "dall-e-2" {
		req.Model = openai.CreateImageModelDallE2
	} else if modelStr == "dall-e-3" {
		req.Model = openai.CreateImageModelDallE3
	} else {
		req.Model = openai.CreateImageModelDallE2 // Default fallback
	}

	logger.Printf("[ImageGenerator] Calling OpenAI CreateImage API...")

	// Call OpenAI API
	resp, err := g.client.CreateImage(context.Background(), req)
	if err != nil {
		logger.Printf("[ImageGenerator ERROR] Failed to create image: %v", err)
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	// Check if we got a response
	if len(resp.Data) == 0 {
		logger.Printf("[ImageGenerator ERROR] No image data in response")
		return nil, fmt.Errorf("no image data returned from API")
	}

	// The response can be either URL or base64
	imageData := resp.Data[0]

	var imgBytes []byte

	// If we got base64 data - DALL-E returns it already as base64
	// We should return the raw image bytes, NOT the base64 string
	// The ByteProcessor will handle base64 encoding for JSON
	if imageData.B64JSON != "" {
		logger.Printf("[ImageGenerator] Received base64 image, decoding to raw bytes...")
		imgBytes, err = base64.StdEncoding.DecodeString(imageData.B64JSON)
		if err != nil {
			logger.Printf("[ImageGenerator ERROR] Failed to decode base64: %v", err)
			return nil, fmt.Errorf("failed to decode base64 image: %w", err)
		}
		logger.Printf("[ImageGenerator] Successfully decoded image: %d bytes", len(imgBytes))
		return imgBytes, nil
	}

	// If we got a URL instead (shouldn't happen with B64JSON format, but handle it)
	if imageData.URL != "" {
		logger.Printf("[ImageGenerator] Received image URL, downloading: %s", imageData.URL)
		httpResp, err := http.Get(imageData.URL)
		if err != nil {
			logger.Printf("[ImageGenerator ERROR] Failed to download image: %v", err)
			return nil, fmt.Errorf("failed to download image from URL: %w", err)
		}
		defer httpResp.Body.Close()

		imgBytes, err = io.ReadAll(httpResp.Body)
		if err != nil {
			logger.Printf("[ImageGenerator ERROR] Failed to read image data: %v", err)
			return nil, fmt.Errorf("failed to read image data: %w", err)
		}
		logger.Printf("[ImageGenerator] Downloaded image: %d bytes", len(imgBytes))
		return imgBytes, nil
	}

	logger.Printf("[ImageGenerator ERROR] No image data (neither base64 nor URL)")
	return nil, fmt.Errorf("no image data in response")
}
