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
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
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
