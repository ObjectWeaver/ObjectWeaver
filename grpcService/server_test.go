package grpcService

import (
	"os"
	"testing"

	pb "github.com/ObjectWeaver/ObjectWeaver/grpc"
)

func TestNewGRPCServer_Unsecure(t *testing.T) {
	// Save original environment
	originalUnsecure := os.Getenv("GRPC_UNSECURE")
	defer func() {
		if originalUnsecure == "" {
			os.Unsetenv("GRPC_UNSECURE")
		} else {
			os.Setenv("GRPC_UNSECURE", originalUnsecure)
		}
	}()

	// Set unsecure mode
	os.Setenv("GRPC_UNSECURE", "true")

	server := NewGRPCServer()
	if server == nil {
		t.Fatal("Expected server to be created in unsecure mode")
	}
}

func TestNewGRPCServer_SecureWithoutCerts(t *testing.T) {
	// Save original environment
	originalUnsecure := os.Getenv("GRPC_UNSECURE")
	originalCert := os.Getenv("CERT_FILE")
	originalKey := os.Getenv("KEY_FILE")
	defer func() {
		if originalUnsecure == "" {
			os.Unsetenv("GRPC_UNSECURE")
		} else {
			os.Setenv("GRPC_UNSECURE", originalUnsecure)
		}
		if originalCert == "" {
			os.Unsetenv("CERT_FILE")
		} else {
			os.Setenv("CERT_FILE", originalCert)
		}
		if originalKey == "" {
			os.Unsetenv("KEY_FILE")
		} else {
			os.Setenv("KEY_FILE", originalKey)
		}
	}()

	// Set secure mode without certs
	os.Unsetenv("GRPC_UNSECURE")
	os.Unsetenv("CERT_FILE")
	os.Unsetenv("KEY_FILE")

	server := NewGRPCServer()
	// Should return nil when certs not provided
	if server != nil {
		t.Error("Expected nil server when CERT_FILE and KEY_FILE not set")
	}
}

func TestNewGRPCServer_SecureWithInvalidCerts(t *testing.T) {
	// Save original environment
	originalUnsecure := os.Getenv("GRPC_UNSECURE")
	originalCert := os.Getenv("CERT_FILE")
	originalKey := os.Getenv("KEY_FILE")
	defer func() {
		if originalUnsecure == "" {
			os.Unsetenv("GRPC_UNSECURE")
		} else {
			os.Setenv("GRPC_UNSECURE", originalUnsecure)
		}
		if originalCert == "" {
			os.Unsetenv("CERT_FILE")
		} else {
			os.Setenv("CERT_FILE", originalCert)
		}
		if originalKey == "" {
			os.Unsetenv("KEY_FILE")
		} else {
			os.Setenv("KEY_FILE", originalKey)
		}
	}()

	// Set secure mode with invalid cert paths
	os.Unsetenv("GRPC_UNSECURE")
	os.Setenv("CERT_FILE", "/nonexistent/cert.pem")
	os.Setenv("KEY_FILE", "/nonexistent/key.pem")

	// This will log.Fatalf, so we can't easily test it
	// Skip this test or refactor the code to return error instead of fatal
	t.Skip("NewGRPCServer calls log.Fatalf on cert load failure, cannot test")
}

func TestServer_ImplementsInterface(t *testing.T) {
	// Verify Server implements the gRPC service interface
	var _ pb.JSONSchemaServiceServer = &Server{}
}
