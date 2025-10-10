package client

import (
	"net/http"
	"testing"
	"time"
)

func TestNewStandardClient_Singleton(t *testing.T) {
	client1 := NewStandardClient()
	client2 := NewStandardClient()

	if client1 != client2 {
		t.Error("NewStandardClient should return the same instance (singleton)")
	}

	if client1 == nil {
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

	if transport.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("Expected MaxIdleConnsPerHost 10, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s, got %v", transport.IdleConnTimeout)
	}
}