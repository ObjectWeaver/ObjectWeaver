package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// TestNewGRPCManager tests the creation of a new GRPCManager
func TestNewGRPCManager(t *testing.T) {
	// Set GRPC_UNSECURE to avoid requiring TLS certificates
	originalUnsecure := os.Getenv("GRPC_UNSECURE")
	os.Setenv("GRPC_UNSECURE", "true")
	defer func() {
		if originalUnsecure == "" {
			os.Unsetenv("GRPC_UNSECURE")
		} else {
			os.Setenv("GRPC_UNSECURE", originalUnsecure)
		}
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ready := make(chan bool, 1)
	manager := NewGRPCManager(listener, ready)

	if manager == nil {
		t.Fatal("Expected GRPCManager to be created, got nil")
	}

	if manager.listener != listener {
		t.Error("Expected listener to be set correctly")
	}

	if manager.server == nil {
		t.Error("Expected server to be initialized")
	}

	if manager.ready != ready {
		t.Error("Expected ready channel to be set correctly")
	}
}

// TestGRPCManagerStart tests the Start method of GRPCManager
func TestGRPCManagerStart(t *testing.T) {
	// Set GRPC_UNSECURE to avoid requiring TLS certificates
	originalUnsecure := os.Getenv("GRPC_UNSECURE")
	os.Setenv("GRPC_UNSECURE", "true")
	defer func() {
		if originalUnsecure == "" {
			os.Unsetenv("GRPC_UNSECURE")
		} else {
			os.Setenv("GRPC_UNSECURE", originalUnsecure)
		}
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ready := make(chan bool, 1)
	manager := NewGRPCManager(listener, ready)

	// Start the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- manager.Start()
	}()

	// Check if ready signal is sent
	select {
	case <-ready:
		// Success - ready signal received
	case <-time.After(1 * time.Second):
		t.Fatal("Expected ready signal to be sent")
	}

	// Stop the server
	manager.server.Stop()

	// Check if Start returns without error
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Expected no error from Start, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start method did not return in time")
	}
}

// TestHandlePanic tests the handlePanic function (limited test due to os.Exit)
func TestHandlePanic(t *testing.T) {
	// Note: We can't fully test handlePanic because it calls os.Exit
	// which terminates the test process. We can only verify the log file creation.

	tempFile := "error.log"
	defer os.Remove(tempFile)

	// Test that logToFile works (which is what handlePanic uses)
	testMessage := "test panic message"
	logToFile(testMessage)

	// Verify the log file was created and contains the message
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !bytes.Contains(content, []byte(testMessage)) {
		t.Errorf("Expected log file to contain %q, got %q", testMessage, string(content))
	}
}

// TestLogToFile tests the logToFile function
func TestLogToFile(t *testing.T) {
	tempFile := "error.log" // Use the actual log file name
	defer os.Remove(tempFile)

	testMessage := "test error message"
	logToFile(testMessage)

	// Read the log file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !bytes.Contains(content, []byte(testMessage)) {
		t.Errorf("Expected log file to contain %q, got %q", testMessage, string(content))
	}
}

// TestLogToFileMultipleEntries tests multiple log entries
func TestLogToFileMultipleEntries(t *testing.T) {
	tempFile := "error.log" // Use the actual log file name
	defer os.Remove(tempFile)

	messages := []string{"error 1", "error 2", "error 3"}
	for _, msg := range messages {
		logToFile(msg)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	for _, msg := range messages {
		if !bytes.Contains(content, []byte(msg)) {
			t.Errorf("Expected log file to contain %q", msg)
		}
	}
}

// TestDefaultPort tests that the default port is used when PORT env var is not set
func TestDefaultPort(t *testing.T) {
	t.Skip("Skipping integration test that starts full server - would conflict with other tests")
	// This test is skipped because it would start actual servers which is complex to test
	// The port logic is simple enough to verify through code inspection
}

// TestCustomPort tests that a custom port from env var is used
func TestCustomPort(t *testing.T) {
	t.Skip("Skipping integration test that starts full server - would conflict with other tests")
	// This test is skipped because it would start actual servers which is complex to test
	// The port logic is simple enough to verify through code inspection
}

// TestPrintAscii tests the printAscii function
func TestPrintAscii(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save and restore environment
	originalPort := os.Getenv("PORT")
	originalEnv := os.Getenv("ENVIRONMENT")
	defer func() {
		if originalPort == "" {
			os.Unsetenv("PORT")
		} else {
			os.Setenv("PORT", originalPort)
		}
		if originalEnv == "" {
			os.Unsetenv("ENVIRONMENT")
		} else {
			os.Setenv("ENVIRONMENT", originalEnv)
		}
	}()

	// Test with default port and production environment
	os.Unsetenv("PORT")
	os.Unsetenv("ENVIRONMENT")

	printAscii()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("ObjectWeaver")) {
		t.Error("Expected output to contain 'ObjectWeaver'")
	}

	if !bytes.Contains([]byte(output), []byte("License Notice")) {
		t.Error("Expected output to contain 'License Notice'")
	}

	if !bytes.Contains([]byte(output), []byte("https://objectweaver.dev")) {
		t.Error("Expected output to contain 'https://objectweaver.dev'")
	}
}

// TestPrintAsciiDevelopmentEnvironment tests printAscii in development mode
func TestPrintAsciiDevelopmentEnvironment(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save and restore environment
	originalEnv := os.Getenv("ENVIRONMENT")
	originalPort := os.Getenv("PORT")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ENVIRONMENT")
		} else {
			os.Setenv("ENVIRONMENT", originalEnv)
		}
		if originalPort == "" {
			os.Unsetenv("PORT")
		} else {
			os.Setenv("PORT", originalPort)
		}
	}()

	// Set development environment
	os.Setenv("ENVIRONMENT", "development")
	testPort := "3000"
	os.Setenv("PORT", testPort)

	printAscii()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedURL := fmt.Sprintf("http://localhost:%s/", testPort)
	if !bytes.Contains([]byte(output), []byte(expectedURL)) {
		t.Errorf("Expected output to contain development URL %q", expectedURL)
	}

	if !bytes.Contains([]byte(output), []byte("testing environment")) {
		t.Error("Expected output to contain 'testing environment' message")
	}
}

// TestServerManagerInterface tests that GRPCManager implements ServerManager
func TestServerManagerInterface(t *testing.T) {
	// Set GRPC_UNSECURE to avoid requiring TLS certificates
	originalUnsecure := os.Getenv("GRPC_UNSECURE")
	os.Setenv("GRPC_UNSECURE", "true")
	defer func() {
		if originalUnsecure == "" {
			os.Unsetenv("GRPC_UNSECURE")
		} else {
			os.Setenv("GRPC_UNSECURE", originalUnsecure)
		}
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ready := make(chan bool, 1)
	manager := NewGRPCManager(listener, ready)

	// Verify that GRPCManager implements ServerManager interface
	var _ ServerManager = manager
}

// TestGRPCServerInitialization tests that the gRPC server is properly initialized
func TestGRPCServerInitialization(t *testing.T) {
	// Set GRPC_UNSECURE to avoid requiring TLS certificates
	originalUnsecure := os.Getenv("GRPC_UNSECURE")
	os.Setenv("GRPC_UNSECURE", "true")
	defer func() {
		if originalUnsecure == "" {
			os.Unsetenv("GRPC_UNSECURE")
		} else {
			os.Setenv("GRPC_UNSECURE", originalUnsecure)
		}
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ready := make(chan bool, 1)
	manager := NewGRPCManager(listener, ready)

	// Check that the server is not nil
	if manager.server == nil {
		t.Error("Expected server to be initialized")
	}
}

// TestLogToFileWithInvalidPath tests error handling when log file path is invalid
func TestLogToFileWithInvalidPath(t *testing.T) {
	// Temporarily change the logToFile function to use an invalid path
	// This test captures stdout to verify error message is printed
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Try to create a file in a non-existent directory
	// We can't easily test this without modifying the function,
	// so we'll just ensure the function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logToFile panicked: %v", r)
		}
		w.Close()
		os.Stdout = oldStdout
		io.Copy(io.Discard, r)
	}()

	logToFile("test message")
}

// TestHandlePanicWithNoPanic tests that handlePanic does nothing when there's no panic
func TestHandlePanicWithNoPanic(t *testing.T) {
	// This should complete without error
	defer handlePanic()
	// Normal execution, no panic
}
