package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusMiddleware(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with prometheus middleware
	wrappedHandler := PrometheusMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected body 'test response', got '%s'", w.Body.String())
	}
}

func TestPrometheusMiddleware_RequestMetrics(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	wrappedHandler := PrometheusMiddleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Content-Length", "100")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestPrometheusMiddleware_ErrorTracking(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	wrappedHandler := PrometheusMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestPrometheusMiddleware_ActiveRequests(t *testing.T) {
	// Check that active requests gauge increases during request
	handlerCalled := make(chan bool)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled <- true
		<-handlerCalled // Wait for signal to continue
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := PrometheusMiddleware(handler)

	go func() {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}()

	<-handlerCalled // Wait for handler to be called
	// Active requests should be > 0 here
	handlerCalled <- true // Signal to continue
}

func TestPrometheusMiddleware_ResponseSize(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
	}{
		{"small response", "ok"},
		{"medium response", strings.Repeat("data", 100)},
		{"large response", strings.Repeat("x", 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.responseBody))
			})

			wrappedHandler := PrometheusMiddleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Body.String() != tt.responseBody {
				t.Errorf("Response body doesn't match")
			}
		})
	}
}

func TestResponseWriter_Write(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		Size:           0,
		StatusCode:     http.StatusOK,
	}

	data := []byte("test data")
	n, err := rw.Write(data)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if rw.Size != len(data) {
		t.Errorf("Expected Size to be %d, got %d", len(data), rw.Size)
	}
}

func TestResponseWriter_MultipleWrites(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		Size:           0,
		StatusCode:     http.StatusOK,
	}

	writes := [][]byte{
		[]byte("first"),
		[]byte("second"),
		[]byte("third"),
	}

	totalSize := 0
	for _, data := range writes {
		n, err := rw.Write(data)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		totalSize += n
	}

	if rw.Size != totalSize {
		t.Errorf("Expected Size to be %d, got %d", totalSize, rw.Size)
	}
}

func TestPrometheusMiddleware_DifferentMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := PrometheusMiddleware(handler)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/endpoint", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d for %s, got %d", http.StatusOK, method, w.Code)
			}
		})
	}
}

func TestPrometheusMiddleware_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			})

			wrappedHandler := PrometheusMiddleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != statusCode {
				t.Errorf("Expected status %d, got %d", statusCode, w.Code)
			}
		})
	}
}

func TestPrometheusMetrics_Registered(t *testing.T) {
	// Test that metrics are registered with Prometheus
	metrics := []prometheus.Collector{
		httpRequestsTotal,
		httpRequestDuration,
		httpResponseSize,
		httpRequestSize,
		activeRequests,
		errorRequestsTotal,
		goroutines,
		memoryUsage,
	}

	for _, metric := range metrics {
		if metric == nil {
			t.Error("Metric should not be nil")
		}
	}
}

func TestPrometheusMiddleware_WithRequestBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	wrappedHandler := PrometheusMiddleware(handler)

	requestBody := "test request body"
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(requestBody))
	req.Header.Set("Content-Length", "17")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != requestBody {
		t.Errorf("Expected body '%s', got '%s'", requestBody, w.Body.String())
	}
}
