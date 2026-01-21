package main

import (
	"net"
	"testing"
	"time"
)

func TestNewHTTPManager(t *testing.T) {
	// Create a test listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ready := make(chan bool, 1)
	manager := NewHTTPManager(listener, ready)

	if manager == nil {
		t.Fatal("Expected non-nil HTTPManager")
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

func TestHTTPManager_Start(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	ready := make(chan bool, 1)
	manager := NewHTTPManager(listener, ready)

	// Start server in goroutine
	go func() {
		manager.Start()
	}()

	// Wait for ready signal
	select {
	case <-ready:
		// Success - ready signal received
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for ready signal")
	}

	// Clean up
	listener.Close()
}

func TestHTTPManager_ReadyChannel(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ready := make(chan bool, 1)
	manager := NewHTTPManager(listener, ready)

	// Verify ready channel is not blocked
	go func() {
		manager.Start()
	}()

	select {
	case val := <-ready:
		if !val {
			t.Error("Expected ready signal to be true")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for ready signal")
	}
}
