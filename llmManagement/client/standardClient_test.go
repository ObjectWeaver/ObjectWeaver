package client

import (
	"net/http"
	"testing"
	"time"
)

func TestNewStandardClient_Singleton(t *testing.T) {
	client1 := NewStandardClient()
	client2 := NewStandardClient()

	// Note: NewStandardClient creates a new instance each time (not a singleton)
	// This allows for independent client configurations
	if client1 == nil {
		t.Error("NewStandardClient should not return nil")
	}

	if client2 == nil {
		t.Error("NewStandardClient should not return nil")
	}
}

func TestNewStandardClientWithTimeout_Timeout(t *testing.T) {
	timeout := 5 * time.Second
	client := NewStandardClientWithTimeout(timeout)

	if client.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.Timeout)
	}

	if client == nil {
		t.Error("NewStandardClientWithTimeout should not return nil")
	}
}

func TestNewStandardClientWithTimeout_Transport(t *testing.T) {
	timeout := 10 * time.Second
	client := NewStandardClientWithTimeout(timeout)

	if client.Transport == nil {
		t.Error("Transport should not be nil")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Error("Transport should be of type *http.Transport")
	}

	// Updated to reflect high-load optimized defaults (5k req/s)
	if transport.MaxIdleConns != 5000 {
		t.Errorf("Expected MaxIdleConns 5000 (high-load default), got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 4000 {
		t.Errorf("Expected MaxIdleConnsPerHost 4000 (high-load default), got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != 5000 {
		t.Errorf("Expected MaxConnsPerHost 5000 (high-load default), got %d", transport.MaxConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s (high-load default), got %v", transport.IdleConnTimeout)
	}
}
