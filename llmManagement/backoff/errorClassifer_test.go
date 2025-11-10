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
package backoff

import (
	"errors"
	"io"
	"testing"
	"time"
)

func TestNewErrorClassifier(t *testing.T) {
	ec := NewErrorClassifier()
	if ec == nil {
		t.Errorf("NewErrorClassifier returned nil")
	}
}

func TestErrorClassifier_Classify_NilError(t *testing.T) {
	ec := NewErrorClassifier()
	result := ec.Classify(nil)
	if result != -1 {
		t.Errorf("Expected -1 for nil error, got %v", result)
	}
}

func TestErrorClassifier_Classify_EOF(t *testing.T) {
	ec := NewErrorClassifier()
	err := io.EOF
	result := ec.Classify(err)
	if result != ErrorTypeTransient {
		t.Errorf("Expected ErrorTypeTransient for io.EOF, got %v", result)
	}
}

func TestErrorClassifier_Classify_RateLimitError(t *testing.T) {
	ec := NewErrorClassifier()
	err := &RateLimitError{RetryAfter: time.Second}
	result := ec.Classify(err)
	if result != ErrorTypeRateLimit {
		t.Errorf("Expected ErrorTypeRateLimit for RateLimitError, got %v", result)
	}
}

func TestErrorClassifier_Classify_Contains429(t *testing.T) {
	ec := NewErrorClassifier()
	err := errors.New("HTTP request failed with status 429")
	result := ec.Classify(err)
	if result != ErrorTypeRateLimit {
		t.Errorf("Expected ErrorTypeRateLimit for error containing 429, got %v", result)
	}
}

func TestErrorClassifier_Classify_OpenAI_InvalidRequest(t *testing.T) {
	ec := NewErrorClassifier()
	err := errors.New("OpenAI API error: [type: invalid_request_error] bad request")
	result := ec.Classify(err)
	if result != ErrorTypePermanent {
		t.Errorf("Expected ErrorTypePermanent for OpenAI invalid_request_error, got %v", result)
	}
}

func TestErrorClassifier_Classify_OpenAI_RateLimitExceeded(t *testing.T) {
	ec := NewErrorClassifier()
	err := errors.New("OpenAI API error: [type: rate_limit_exceeded] too many requests")
	result := ec.Classify(err)
	if result != ErrorTypeRateLimit {
		t.Errorf("Expected ErrorTypeRateLimit for OpenAI rate_limit_exceeded, got %v", result)
	}
}

func TestErrorClassifier_Classify_OpenAI_ServerError(t *testing.T) {
	ec := NewErrorClassifier()
	err := errors.New("OpenAI API error: [type: server_error] internal server error")
	result := ec.Classify(err)
	if result != ErrorTypeTransient {
		t.Errorf("Expected ErrorTypeTransient for OpenAI server_error, got %v", result)
	}
}

func TestErrorClassifier_Classify_OpenAI_ServiceUnavailable(t *testing.T) {
	ec := NewErrorClassifier()
	err := errors.New("OpenAI API error: [type: service_unavailable] service down")
	result := ec.Classify(err)
	if result != ErrorTypeTransient {
		t.Errorf("Expected ErrorTypeTransient for OpenAI service_unavailable, got %v", result)
	}
}

func TestErrorClassifier_Classify_TransientErrors(t *testing.T) {
	ec := NewErrorClassifier()
	transientStrings := []string{
		"client connection lost",
		"Service Unavailable",
		"Internal Server Error",
		"context deadline exceeded",
		"connection reset by peer",
		"broken pipe",
		"unexpected EOF",
		"HTTP request failed with status 502",
		"HTTP request failed with status 503",
		"HTTP request failed with status 504",
		"failed to unmarshal json into ChatCompletionResponse",
	}

	for _, s := range transientStrings {
		err := errors.New(s)
		result := ec.Classify(err)
		if result != ErrorTypeTransient {
			t.Errorf("Expected ErrorTypeTransient for '%s', got %v", s, result)
		}
	}
}

func TestErrorClassifier_Classify_PermanentHttpErrors(t *testing.T) {
	ec := NewErrorClassifier()
	permanentStrings := []string{
		"HTTP request failed with status 400",
		"HTTP request failed with status 401",
		"HTTP request failed with status 403",
		"HTTP request failed with status 404",
	}

	for _, s := range permanentStrings {
		err := errors.New(s)
		result := ec.Classify(err)
		if result != ErrorTypePermanent {
			t.Errorf("Expected ErrorTypePermanent for '%s', got %v", s, result)
		}
	}
}

func TestErrorClassifier_Classify_DefaultPermanent(t *testing.T) {
	ec := NewErrorClassifier()
	err := errors.New("some unknown error")
	result := ec.Classify(err)
	if result != ErrorTypePermanent {
		t.Errorf("Expected ErrorTypePermanent for unknown error, got %v", result)
	}
}

func TestErrorClassifier_Classify_OpenAI_NoType(t *testing.T) {
	ec := NewErrorClassifier()
	err := errors.New("OpenAI API error: some other error")
	result := ec.Classify(err)
	if result != ErrorTypePermanent {
		t.Errorf("Expected ErrorTypePermanent for OpenAI error without specific type, got %v", result)
	}
}
