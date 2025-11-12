package grpcService

import (
	"context"
	"os"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		token     string
		expectErr bool
	}{
		{
			name:      "valid token",
			envValue:  "secret-password",
			token:     "secret-password",
			expectErr: false,
		},
		{
			name:      "invalid token",
			envValue:  "secret-password",
			token:     "wrong-password",
			expectErr: true,
		},
		{
			name:      "empty token",
			envValue:  "secret-password",
			token:     "",
			expectErr: true,
		},
		{
			name:      "token matches empty env",
			envValue:  "",
			token:     "",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("PASSWORD", tt.envValue)
			defer os.Unsetenv("PASSWORD")

			err := validateToken(tt.token)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestAuthInterceptor(t *testing.T) {
	// Set up environment
	os.Setenv("PASSWORD", "valid-api-key")
	defer os.Unsetenv("PASSWORD")

	tests := []struct {
		name      string
		metadata  metadata.MD
		expectErr bool
	}{
		{
			name: "valid API key",
			metadata: metadata.Pairs(
				"x-api-key", "valid-api-key",
			),
			expectErr: false,
		},
		{
			name: "invalid API key",
			metadata: metadata.Pairs(
				"x-api-key", "invalid-key",
			),
			expectErr: true,
		},
		{
			name: "missing API key header",
			metadata: metadata.Pairs(
				"other-header", "value",
			),
			expectErr: true,
		},
		{
			name:      "no metadata",
			metadata:  nil,
			expectErr: true,
		},
		{
			name: "empty API key value",
			metadata: metadata.Pairs(
				"x-api-key", "",
			),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with metadata
			var ctx context.Context
			if tt.metadata != nil {
				ctx = metadata.NewIncomingContext(context.Background(), tt.metadata)
			} else {
				ctx = context.Background()
			}

			// Mock handler that returns success
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return "success", nil
			}

			// Call the interceptor
			_, err := AuthInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestAuthInterceptor_HandlerCalled(t *testing.T) {
	os.Setenv("PASSWORD", "test-key")
	defer os.Unsetenv("PASSWORD")

	md := metadata.Pairs("x-api-key", "test-key")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "handler response", nil
	}

	resp, err := AuthInterceptor(ctx, "test-request", &grpc.UnaryServerInfo{}, handler)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	if resp != "handler response" {
		t.Errorf("Expected 'handler response', got: %v", resp)
	}
}

func TestAuthInterceptor_HandlerNotCalledOnError(t *testing.T) {
	os.Setenv("PASSWORD", "correct-key")
	defer os.Unsetenv("PASSWORD")

	md := metadata.Pairs("x-api-key", "wrong-key")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "handler response", nil
	}

	_, err := AuthInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	if err == nil {
		t.Error("Expected error for invalid API key")
	}

	if handlerCalled {
		t.Error("Handler should not be called when auth fails")
	}
}

func TestAuthInterceptor_MultipleAPIKeys(t *testing.T) {
	os.Setenv("PASSWORD", "valid-key")
	defer os.Unsetenv("PASSWORD")

	// Test with multiple values for x-api-key (should use the first one)
	md := metadata.MD{
		"x-api-key": []string{"valid-key", "another-key"},
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err := AuthInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	if err != nil {
		t.Errorf("Expected no error with valid first key, got: %v", err)
	}
}

func TestValidateToken_CaseSensitive(t *testing.T) {
	os.Setenv("PASSWORD", "SecretKey")
	defer os.Unsetenv("PASSWORD")

	tests := []struct {
		name      string
		token     string
		expectErr bool
	}{
		{
			name:      "exact match",
			token:     "SecretKey",
			expectErr: false,
		},
		{
			name:      "lowercase",
			token:     "secretkey",
			expectErr: true,
		},
		{
			name:      "uppercase",
			token:     "SECRETKEY",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToken(tt.token)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
