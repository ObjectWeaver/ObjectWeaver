package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "GET request",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST request",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "PUT request",
			method:     http.MethodPut,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			HealthCheck(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HealthCheck() status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", contentType)
			}

			// Parse and validate response
			var response HealthCheckResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "healthy" {
				t.Errorf("Status = %v, want healthy", response.Status)
			}

			if response.Service != "objectweaver" {
				t.Errorf("Service = %v, want objectweaver", response.Service)
			}

			// Check timestamp is recent (within last 5 seconds)
			timeDiff := time.Since(response.Timestamp)
			if timeDiff > 5*time.Second {
				t.Errorf("Timestamp is too old: %v", response.Timestamp)
			}
			if timeDiff < 0 {
				t.Errorf("Timestamp is in the future: %v", response.Timestamp)
			}
		})
	}
}

func TestHealthCheckResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthCheck(w, req)

	var response HealthCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate all fields are populated
	if response.Status == "" {
		t.Error("Status field is empty")
	}
	if response.Service == "" {
		t.Error("Service field is empty")
	}
	if response.Timestamp.IsZero() {
		t.Error("Timestamp field is zero")
	}

	// Validate timestamp is in UTC
	if response.Timestamp.Location() != time.UTC {
		t.Errorf("Timestamp should be in UTC, got %v", response.Timestamp.Location())
	}
}

func TestHealthCheckConcurrent(t *testing.T) {
	// Test concurrent health check requests
	const numRequests = 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			HealthCheck(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Concurrent request failed with status %d", w.Code)
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

func TestHealthCheckResponseJSONFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthCheck(w, req)

	// Ensure the response is valid JSON
	var jsonData map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&jsonData); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	// Check expected fields exist
	expectedFields := []string{"status", "timestamp", "service"}
	for _, field := range expectedFields {
		if _, ok := jsonData[field]; !ok {
			t.Errorf("Response missing field: %s", field)
		}
	}
}
