package client

import (
	"context"
	"net/http"
	"testing"

	"cloud.google.com/go/auth/credentials"
	"google.golang.org/genai"
)

func TestNewGzipClient_StandardClient(t *testing.T) {
	ctx := context.Background()
	cc := &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  "somekey",
	}

	client, err := NewGzipClient(ctx, cc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("expected client, got nil")
	}

	// Check transport is gzipTransport
	gzTransport, ok := client.Transport.(*gzipTransport)
	if !ok {
		t.Fatalf("expected transport to be *gzipTransport, got %T", client.Transport)
	}

	// For standard, underlying should be http.DefaultTransport
	if gzTransport.underlyingTransport != http.DefaultTransport {
		t.Errorf("expected underlying transport to be http.DefaultTransport, got %v", gzTransport.underlyingTransport)
	}

	// Check other fields are copied (from baseClient which is &http.Client{})
	if client.CheckRedirect != nil {
		t.Errorf("expected CheckRedirect to be nil, but it was not")
	}
	if client.Jar != nil {
		t.Errorf("expected Jar to be nil, but it was not")
	}
	if client.Timeout != 0 {
		t.Errorf("expected Timeout to be 0, but it was not")
	}
}

func TestNewGzipClient_VertexAIWithAPIKey(t *testing.T) {
	ctx := context.Background()
	cc := &genai.ClientConfig{
		Backend: genai.BackendVertexAI,
		APIKey:  "somekey",
	}

	client, err := NewGzipClient(ctx, cc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("expected client, got nil")
	}

	// Same checks as above
	gzTransport, ok := client.Transport.(*gzipTransport)
	if !ok {
		t.Fatalf("expected transport to be *gzipTransport, got %T", client.Transport)
	}

	if gzTransport.underlyingTransport != http.DefaultTransport {
		t.Errorf("expected underlying transport to be http.DefaultTransport, got %v", gzTransport.underlyingTransport)
	}
}

func TestNewGzipClient_VertexAINoAPIKey(t *testing.T) {
	ctx := context.Background()
	cc := &genai.ClientConfig{
		Backend: genai.BackendVertexAI,
		APIKey:  "",
	}

	// Try to detect credentials
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		t.Skip("no default credentials available for testing Vertex AI path")
	}
	cc.Credentials = creds

	client, err := NewGzipClient(ctx, cc)
	if err != nil {
		t.Fatalf("expected no error with credentials, got %v", err)
	}

	if client == nil {
		t.Fatal("expected client, got nil")
	}

	// Check transport is gzipTransport
	gzTransport, ok := client.Transport.(*gzipTransport)
	if !ok {
		t.Fatalf("expected transport to be *gzipTransport, got %T", client.Transport)
	}

	// underlying should not be http.DefaultTransport, since it's authenticated
	if gzTransport.underlyingTransport == http.DefaultTransport {
		t.Errorf("expected underlying transport to be authenticated, but got http.DefaultTransport")
	}
}

func TestNewGzipClient_VertexAINoAPIKeyNoCredentials(t *testing.T) {
	// Note: This test may pass or fail depending on whether default credentials are available in the environment.
	// If credentials are detected, it will succeed; otherwise, it will fail with error.
	// For unit testing, we skip asserting since mocking is not set up.
	t.Skip("Skipping test due to environment-dependent credential detection")
}
