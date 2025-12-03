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

	// OpenAI adapter uses OPENAI_API_KEY env var, not LLM_API_KEY
	// The adapter will be created but will fail when actually used
	adapter, err := NewClientAdapterFromEnv()
	if err != nil {
		t.Fatalf("Expected no error during adapter creation, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}
}

func TestNewClientAdapterFromEnv_Gemini_MissingKey(t *testing.T) {
	defer setEnv(t, map[string]string{
		"LLM_PROVIDER": "gemini",
	})()

	// Gemini adapter uses GEMINI_API_KEY env var, not LLM_API_KEY
	// The adapter will be created but will fail when actually used
	adapter, err := NewClientAdapterFromEnv()
	if err != nil {
		t.Fatalf("Expected no error during adapter creation, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
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
	// Ensure OPENAI_API_KEY is not set
	defer setEnv(t, map[string]string{})()

	config := AdapterConfig{
		Provider: ProviderOpenAI,
	}

	// OpenAI adapter uses OPENAI_API_KEY env var directly
	// The adapter will be created with empty key but will fail when used
	adapter, err := NewClientAdapter(config)
	if err != nil {
		t.Fatalf("Expected no error during adapter creation, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}
}

func TestNewClientAdapter_Gemini_MissingKey(t *testing.T) {
	// Ensure GEMINI_API_KEY is not set
	defer setEnv(t, map[string]string{})()

	config := AdapterConfig{
		Provider: ProviderGemini,
	}

	// Gemini adapter uses GEMINI_API_KEY env var directly
	// The adapter will be created with empty key but will fail when used
	adapter, err := NewClientAdapter(config)
	if err != nil {
		t.Fatalf("Expected no error during adapter creation, got %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
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
