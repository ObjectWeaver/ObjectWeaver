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
	"os"
	"testing"
)

// setEnv sets environment variables and returns a function to restore them
func setEnv(t *testing.T, env map[string]string) func() {
	original := make(map[string]string)
	for key := range env {
		original[key] = os.Getenv(key)
		os.Setenv(key, env[key])
	}
	return func() {
		for key, value := range original {
			os.Setenv(key, value)
		}
	}
}

func TestNewClientAdapterFromEnv_OpenAI(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "openai",
		"LLM_API_KEY":  "test-key",
		"LLM_USE_GZIP": "false",
	})()

	adapter, err := NewClientAdapterFromEnv()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*OpenAIClientAdapter); !ok {
		t.Fatalf("Expected OpenAIClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapterFromEnv_Gemini(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "gemini",
		"LLM_API_KEY":  "test-key",
		"LLM_USE_GZIP": "true",
	})()

	adapter, err := NewClientAdapterFromEnv()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*GeminiClientAdapter); !ok {
		t.Fatalf("Expected GeminiClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapterFromEnv_Local(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "local",
		"LLM_API_URL":  "http://localhost:8080",
		"LLM_API_KEY":  "test-key",
		"LLM_USE_GZIP": "false",
	})()

	adapter, err := NewClientAdapterFromEnv()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*LocalClientAdapter); !ok {
		t.Fatalf("Expected LocalClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapterFromEnv_AutoDetectLocal(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_API_URL":  "http://localhost:8080",
		"LLM_API_KEY":  "test-key",
		"LLM_USE_GZIP": "false",
	})()

	adapter, err := NewClientAdapterFromEnv()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*LocalClientAdapter); !ok {
		t.Fatalf("Expected LocalClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapterFromEnv_InvalidProvider(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "invalid",
		"LLM_API_KEY":  "test-key",
	})()

	_, err := NewClientAdapterFromEnv()
	if err == nil {
		t.Fatal("Expected error for invalid provider")
	}
	expected := "unknown provider: invalid (supported: local, openai, gemini)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapterFromEnv_MissingProviderAndURL(t *testing.T) {
	defer setEnv(t, map[string]string{})()

	_, err := NewClientAdapterFromEnv()
	if err == nil {
		t.Fatal("Expected error for missing provider and URL")
	}
	expected := "LLM_PROVIDER must be set (options: local, openai, gemini), or provide LLM_API_URL for local provider"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapterFromEnv_OpenAI_MissingKey(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "openai",
	})()

	_, err := NewClientAdapterFromEnv()
	if err == nil {
		t.Fatal("Expected error for missing API key")
	}
	expected := "openAI API key is required (set LLM_API_KEY)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapterFromEnv_Gemini_MissingKey(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "gemini",
	})()

	_, err := NewClientAdapterFromEnv()
	if err == nil {
		t.Fatal("Expected error for missing API key")
	}
	expected := "gemini API key is required (set LLM_API_KEY)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapterFromEnv_Local_MissingURL(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "local",
		"LLM_API_KEY":  "test-key",
	})()

	_, err := NewClientAdapterFromEnv()
	if err == nil {
		t.Fatal("Expected error for missing URL")
	}
	expected := "local API URL is required (set LLM_API_URL)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapter_OpenAI(t *testing.T) {
	config := AdapterConfig{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		UseGzip:  false,
	}

	adapter, err := NewClientAdapter(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*OpenAIClientAdapter); !ok {
		t.Fatalf("Expected OpenAIClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapter_Gemini(t *testing.T) {
	config := AdapterConfig{
		Provider: ProviderGemini,
		APIKey:   "test-key",
		UseGzip:  true,
	}

	adapter, err := NewClientAdapter(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*GeminiClientAdapter); !ok {
		t.Fatalf("Expected GeminiClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapter_Local(t *testing.T) {
	config := AdapterConfig{
		Provider: ProviderLocal,
		URL:      "http://localhost:8080",
		APIKey:   "test-key",
		UseGzip:  false,
	}

	adapter, err := NewClientAdapter(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*LocalClientAdapter); !ok {
		t.Fatalf("Expected LocalClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapter_InvalidProvider(t *testing.T) {
	config := AdapterConfig{
		Provider: "invalid",
		APIKey:   "test-key",
	}

	_, err := NewClientAdapter(config)
	if err == nil {
		t.Fatal("Expected error for invalid provider")
	}
	expected := "unknown provider: invalid (supported: local, openai, gemini)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapter_OpenAI_MissingKey(t *testing.T) {
	config := AdapterConfig{
		Provider: ProviderOpenAI,
	}

	_, err := NewClientAdapter(config)
	if err == nil {
		t.Fatal("Expected error for missing API key")
	}
	expected := "openAI API key is required (set LLM_API_KEY)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapter_Gemini_MissingKey(t *testing.T) {
	config := AdapterConfig{
		Provider: ProviderGemini,
	}

	_, err := NewClientAdapter(config)
	if err == nil {
		t.Fatal("Expected error for missing API key")
	}
	expected := "gemini API key is required (set LLM_API_KEY)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapter_Local_MissingURL(t *testing.T) {
	config := AdapterConfig{
		Provider: ProviderLocal,
		APIKey:   "test-key",
	}

	_, err := NewClientAdapter(config)
	if err == nil {
		t.Fatal("Expected error for missing URL")
	}
	expected := "local API URL is required (set LLM_API_URL)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewClientAdapterWithDefaults_OpenAI(t *testing.T) {
	adapter, err := NewClientAdapterWithDefaults(ProviderOpenAI, "test-key")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*OpenAIClientAdapter); !ok {
		t.Fatalf("Expected OpenAIClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapterWithDefaults_Gemini(t *testing.T) {
	adapter, err := NewClientAdapterWithDefaults(ProviderGemini, "test-key")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*GeminiClientAdapter); !ok {
		t.Fatalf("Expected GeminiClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapterWithDefaults_Local(t *testing.T) {
	adapter, err := NewClientAdapterWithDefaults(ProviderLocal, "test-key")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
	}
	if _, ok := adapter.(*LocalClientAdapter); !ok {
		t.Fatalf("Expected LocalClientAdapter, got %T", adapter)
	}
}

func TestNewClientAdapterWithDefaults_InvalidProvider(t *testing.T) {
	_, err := NewClientAdapterWithDefaults("invalid", "test-key")
	if err == nil {
		t.Fatal("Expected error for invalid provider")
	}
	expected := "unknown provider: invalid (supported: local, openai, gemini)"
	if err.Error() != expected {
		t.Fatalf("Expected error %q, got %q", expected, err.Error())
	}
}
