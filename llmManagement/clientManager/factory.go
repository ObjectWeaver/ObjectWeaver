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
	"objectweaver/llmManagement/client"
	"objectweaver/llmManagement/modelConverter"
	"objectweaver/llmManagement/requestManagement"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ProviderType defines the type of LLM provider to use.
type ProviderType string

const (
	// ProviderLocal uses a local or custom HTTP endpoint (OpenAI-compatible)
	ProviderLocal ProviderType = "local"
	// ProviderOpenAI uses the official OpenAI API with native SDK
	ProviderOpenAI ProviderType = "openai"
	// ProviderGemini uses Google's Gemini API with format conversion
	ProviderGemini ProviderType = "gemini"
)

// AdapterConfig holds configuration for creating a ClientAdapter.
type AdapterConfig struct {
	Provider   ProviderType
	URL        string // For local/custom providers
	APIKey     string
	UseGzip    bool
	HTTPClient *http.Client // Optional: provide your own HTTP client
}

// NewClientAdapterFromEnv creates a ClientAdapter based on environment variables.
// This is the recommended way to initialize the LLM client in production.
//
// Environment variables:
//   - LLM_PROVIDER: "local", "openai", or "gemini" (auto-detected if not set)
//   - LLM_API_URL: URL for local/custom endpoints (required for local provider)
//   - LLM_API_KEY: API key for the provider
//   - LLM_USE_GZIP: "true" to enable gzip compression (default: "false")
//
// If LLM_PROVIDER is not set, the provider is auto-detected based on available configuration:
//   - If LLM_API_URL is set → local provider
//   - Otherwise, requires LLM_PROVIDER to be explicitly set
func NewClientAdapterFromEnv() (ClientAdapter, error) {
	provider := os.Getenv("LLM_PROVIDER")
	apiURL := os.Getenv("LLM_API_URL")
	apiKey := os.Getenv("LLM_API_KEY")

	// Auto-detect provider if not explicitly set
	if provider == "" {
		if apiURL != "" {
			// If API URL is set, assume local provider
			provider = "local"
		} else {
			// No provider specified and no URL - cannot auto-detect
			return nil, fmt.Errorf("LLM_PROVIDER must be set (options: local, openai, gemini), or provide LLM_API_URL for local provider")
		}
	}

	config := AdapterConfig{
		Provider: ProviderType(strings.ToLower(provider)),
		URL:      apiURL,
		APIKey:   apiKey,
		UseGzip:  strings.ToLower(os.Getenv("LLM_USE_GZIP")) == "true",
	}

	return NewClientAdapter(config)
}

// NewClientAdapter creates a ClientAdapter based on the provided configuration.
// This factory method ensures all adapters are created with consistent dependencies
// and follow the SOLID principles.
func NewClientAdapter(config AdapterConfig) (ClientAdapter, error) {
	// Setup common dependencies that all adapters need
	modelConv := modelConverter.NewModelConverter()
	reqBuilder := requestManagement.NewDefaultOpenAIReqBuilder(modelConv)

	// Setup HTTP client if not provided
	httpClient := config.HTTPClient
	if httpClient == nil {
		if config.UseGzip {
			baseClient := client.NewStandardClient()
			httpClient = client.NewGenericGzipClient(baseClient)
		} else {
			httpClient = client.NewStandardClient()
		}
	}

	// Create the appropriate adapter based on provider type
	switch config.Provider {
	case ProviderOpenAI:
		if config.APIKey == "" {
			return nil, fmt.Errorf("openAI API key is required (set LLM_API_KEY)")
		}
		return NewOpenAIClientAdapter(config.APIKey, reqBuilder), nil

	case ProviderGemini:
		if config.APIKey == "" {
			return nil, fmt.Errorf("gemini API key is required (set LLM_API_KEY)")
		}
		return NewGeminiClientAdapter(config.APIKey, reqBuilder, modelConv, httpClient), nil

	case ProviderLocal:
		if config.URL == "" {
			return nil, fmt.Errorf("local API URL is required (set LLM_API_URL)")
		}
		converter := requestManagement.NewLocalConverter()
		return NewLocalClientAdapter(config.URL, config.APIKey, reqBuilder, converter, httpClient), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: local, openai, gemini)", config.Provider)
	}
}

// NewClientAdapterWithDefaults creates a ClientAdapter with sensible defaults.
// This is useful for testing or simple setups.
func NewClientAdapterWithDefaults(provider ProviderType, apiKey string) (ClientAdapter, error) {
	config := AdapterConfig{
		Provider: provider,
		APIKey:   apiKey,
		UseGzip:  false,
	}

	// Set default URLs for known providers
	switch provider {
	case ProviderLocal:
		config.URL = "http://localhost:11434/v1/chat/completions" // Default Ollama endpoint
	}

	return NewClientAdapter(config)
}
