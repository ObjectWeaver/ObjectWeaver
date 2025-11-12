package cors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		check   func(*testing.T, *Cors)
	}{
		{
			name: "default options with no origins",
			options: Options{
				AllowOriginFunc: nil,
			},
			check: func(t *testing.T, c *Cors) {
				if !c.allowedOriginsAll {
					t.Error("Expected allowedOriginsAll to be true with empty origins")
				}
			},
		},
		{
			name: "wildcard origin",
			options: Options{
				AllowedOrigins: []string{"*"},
			},
			check: func(t *testing.T, c *Cors) {
				if !c.allowedOriginsAll {
					t.Error("Expected allowedOriginsAll to be true with * origin")
				}
			},
		},
		{
			name: "specific origins",
			options: Options{
				AllowedOrigins: []string{"http://example.com", "https://test.com"},
			},
			check: func(t *testing.T, c *Cors) {
				if c.allowedOriginsAll {
					t.Error("Expected allowedOriginsAll to be false")
				}
				if len(c.allowedOrigins) != 2 {
					t.Errorf("Expected 2 allowed origins, got %d", len(c.allowedOrigins))
				}
			},
		},
		{
			name: "wildcard in origin",
			options: Options{
				AllowedOrigins: []string{"http://*.example.com"},
			},
			check: func(t *testing.T, c *Cors) {
				if len(c.allowedWOrigins) != 1 {
					t.Errorf("Expected 1 wildcard origin, got %d", len(c.allowedWOrigins))
				}
			},
		},
		{
			name: "default headers",
			options: Options{
				AllowedOrigins: []string{"*"},
			},
			check: func(t *testing.T, c *Cors) {
				if len(c.allowedHeaders) != 3 {
					t.Errorf("Expected 3 default headers, got %d", len(c.allowedHeaders))
				}
			},
		},
		{
			name: "wildcard headers",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedHeaders: []string{"*"},
			},
			check: func(t *testing.T, c *Cors) {
				if !c.allowedHeadersAll {
					t.Error("Expected allowedHeadersAll to be true with * header")
				}
			},
		},
		{
			name: "default methods",
			options: Options{
				AllowedOrigins: []string{"*"},
			},
			check: func(t *testing.T, c *Cors) {
				if len(c.allowedMethods) != 3 {
					t.Errorf("Expected 3 default methods, got %d", len(c.allowedMethods))
				}
			},
		},
		{
			name: "custom methods",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
			},
			check: func(t *testing.T, c *Cors) {
				if len(c.allowedMethods) != 4 {
					t.Errorf("Expected 4 methods, got %d", len(c.allowedMethods))
				}
			},
		},
		{
			name: "with credentials",
			options: Options{
				AllowedOrigins:   []string{"http://example.com"},
				AllowCredentials: true,
			},
			check: func(t *testing.T, c *Cors) {
				if !c.allowCredentials {
					t.Error("Expected allowCredentials to be true")
				}
			},
		},
		{
			name: "with max age",
			options: Options{
				AllowedOrigins: []string{"*"},
				MaxAge:         3600,
			},
			check: func(t *testing.T, c *Cors) {
				if c.maxAge != 3600 {
					t.Errorf("Expected maxAge to be 3600, got %d", c.maxAge)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.options)
			if c == nil {
				t.Fatal("Expected non-nil Cors instance")
			}
			tt.check(t, c)
		})
	}
}

func TestAllowAll(t *testing.T) {
	c := AllowAll()
	if c == nil {
		t.Fatal("Expected non-nil Cors instance")
	}
	if !c.allowedOriginsAll {
		t.Error("Expected allowedOriginsAll to be true")
	}
	if !c.allowedHeadersAll {
		t.Error("Expected allowedHeadersAll to be true")
	}
	if len(c.allowedMethods) != 6 {
		t.Errorf("Expected 6 methods, got %d", len(c.allowedMethods))
	}
}

func TestHandler(t *testing.T) {
	tests := []struct {
		name          string
		options       Options
		method        string
		headers       map[string]string
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
		handlerCalled bool
	}{
		{
			name: "preflight request with allowed origin",
			options: Options{
				AllowedOrigins: []string{"http://example.com"},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Content-Type"},
			},
			method: "OPTIONS",
			headers: map[string]string{
				"Origin":                        "http://example.com",
				"Access-Control-Request-Method": "POST",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
					t.Errorf("Expected Allow-Origin header to be http://example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
				}
				if w.Header().Get("Access-Control-Allow-Methods") != "POST" {
					t.Errorf("Expected Allow-Methods header to be POST, got %s", w.Header().Get("Access-Control-Allow-Methods"))
				}
			},
			handlerCalled: false,
		},
		{
			name: "preflight request with wildcard origin",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST"},
			},
			method: "OPTIONS",
			headers: map[string]string{
				"Origin":                        "http://anywhere.com",
				"Access-Control-Request-Method": "GET",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Errorf("Expected Allow-Origin header to be *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
				}
			},
			handlerCalled: false,
		},
		{
			name: "actual request with allowed origin",
			options: Options{
				AllowedOrigins: []string{"http://example.com"},
				AllowedMethods: []string{"GET", "POST"},
			},
			method: "GET",
			headers: map[string]string{
				"Origin": "http://example.com",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
					t.Errorf("Expected Allow-Origin header to be http://example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
				}
			},
			handlerCalled: true,
		},
		{
			name: "actual request with wildcard origin",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET"},
			},
			method: "GET",
			headers: map[string]string{
				"Origin": "http://test.com",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Errorf("Expected Allow-Origin header to be *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
				}
			},
			handlerCalled: true,
		},
		{
			name: "actual request with disallowed origin",
			options: Options{
				AllowedOrigins: []string{"http://example.com"},
				AllowedMethods: []string{"GET"},
			},
			method: "GET",
			headers: map[string]string{
				"Origin": "http://evil.com",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "" {
					t.Error("Expected no Allow-Origin header for disallowed origin")
				}
			},
			handlerCalled: true,
		},
		{
			name: "request with credentials",
			options: Options{
				AllowedOrigins:   []string{"http://example.com"},
				AllowedMethods:   []string{"GET"},
				AllowCredentials: true,
			},
			method: "GET",
			headers: map[string]string{
				"Origin": "http://example.com",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
					t.Error("Expected Allow-Credentials header to be true")
				}
			},
			handlerCalled: true,
		},
		{
			name: "preflight with options passthrough",
			options: Options{
				AllowedOrigins:     []string{"*"},
				AllowedMethods:     []string{"POST"},
				OptionsPassthrough: true,
			},
			method: "OPTIONS",
			headers: map[string]string{
				"Origin":                        "http://example.com",
				"Access-Control-Request-Method": "POST",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Expected Allow-Origin header")
				}
			},
			handlerCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.options)

			handlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := c.Handler(nextHandler)

			req := httptest.NewRequest(tt.method, "http://example.com/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			tt.checkResponse(t, w)

			if handlerCalled != tt.handlerCalled {
				t.Errorf("Expected handlerCalled to be %v, got %v", tt.handlerCalled, handlerCalled)
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name     string
		options  Options
		origin   string
		expected bool
	}{
		{
			name: "allowed origin exact match",
			options: Options{
				AllowedOrigins: []string{"http://example.com"},
			},
			origin:   "http://example.com",
			expected: true,
		},
		{
			name: "disallowed origin",
			options: Options{
				AllowedOrigins: []string{"http://example.com"},
			},
			origin:   "http://evil.com",
			expected: false,
		},
		{
			name: "wildcard all origins",
			options: Options{
				AllowedOrigins: []string{"*"},
			},
			origin:   "http://anywhere.com",
			expected: true,
		},
		{
			name: "wildcard subdomain match",
			options: Options{
				AllowedOrigins: []string{"http://*.example.com"},
			},
			origin:   "http://test.example.com",
			expected: true,
		},
		{
			name: "wildcard subdomain no match",
			options: Options{
				AllowedOrigins: []string{"http://*.example.com"},
			},
			origin:   "http://example.com",
			expected: false,
		},
		{
			name: "custom function allows",
			options: Options{
				AllowOriginFunc: func(r *http.Request, origin string) bool {
					return strings.HasPrefix(origin, "http://trusted")
				},
			},
			origin:   "http://trusted.com",
			expected: true,
		},
		{
			name: "custom function denies",
			options: Options{
				AllowOriginFunc: func(r *http.Request, origin string) bool {
					return strings.HasPrefix(origin, "http://trusted")
				},
			},
			origin:   "http://evil.com",
			expected: false,
		},
		{
			name: "case insensitive origin",
			options: Options{
				AllowedOrigins: []string{"http://Example.COM"},
			},
			origin:   "http://example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.options)
			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			result := c.isOriginAllowed(req, tt.origin)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsMethodAllowed(t *testing.T) {
	tests := []struct {
		name     string
		options  Options
		method   string
		expected bool
	}{
		{
			name: "allowed method",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST"},
			},
			method:   "GET",
			expected: true,
		},
		{
			name: "disallowed method",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST"},
			},
			method:   "DELETE",
			expected: false,
		},
		{
			name: "OPTIONS always allowed",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET"},
			},
			method:   "OPTIONS",
			expected: true,
		},
		{
			name: "case insensitive method",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"get", "post"},
			},
			method:   "GET",
			expected: true,
		},
		{
			name: "OPTIONS allowed even with empty methods",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{},
			},
			method:   "OPTIONS",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.options)
			result := c.isMethodAllowed(tt.method)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for method %s", tt.expected, result, tt.method)
			}
		})
	}
}

func TestAreHeadersAllowed(t *testing.T) {
	tests := []struct {
		name     string
		options  Options
		headers  []string
		expected bool
	}{
		{
			name: "allowed headers",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			},
			headers:  []string{"Content-Type"},
			expected: true,
		},
		{
			name: "disallowed header",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedHeaders: []string{"Content-Type"},
			},
			headers:  []string{"X-Custom-Header"},
			expected: false,
		},
		{
			name: "wildcard headers",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedHeaders: []string{"*"},
			},
			headers:  []string{"Any-Header"},
			expected: true,
		},
		{
			name: "empty requested headers",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedHeaders: []string{"Content-Type"},
			},
			headers:  []string{},
			expected: true,
		},
		{
			name: "multiple headers all allowed",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedHeaders: []string{"Content-Type", "Authorization", "X-Custom"},
			},
			headers:  []string{"Content-Type", "Authorization"},
			expected: true,
		},
		{
			name: "multiple headers one disallowed",
			options: Options{
				AllowedOrigins: []string{"*"},
				AllowedHeaders: []string{"Content-Type"},
			},
			headers:  []string{"Content-Type", "X-Not-Allowed"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.options)
			result := c.areHeadersAllowed(tt.headers)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for headers %v", tt.expected, result, tt.headers)
			}
		})
	}
}

func TestHandlerFunc(t *testing.T) {
	options := Options{
		AllowedOrigins: []string{"http://example.com"},
		AllowedMethods: []string{"GET", "POST"},
	}

	handler := Handler(options)
	if handler == nil {
		t.Fatal("Expected non-nil handler function")
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handler(nextHandler)
	if wrappedHandler == nil {
		t.Fatal("Expected non-nil wrapped handler")
	}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Error("Expected CORS headers to be set")
	}
}

func TestPreflightWithExposedHeaders(t *testing.T) {
	c := New(Options{
		AllowedOrigins: []string{"http://example.com"},
		AllowedMethods: []string{"GET"},
		ExposedHeaders: []string{"X-Custom-Header", "X-Another-Header"},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := c.Handler(nextHandler)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	exposedHeaders := w.Header().Get("Access-Control-Expose-Headers")
	if !strings.Contains(exposedHeaders, "X-Custom-Header") {
		t.Error("Expected X-Custom-Header in exposed headers")
	}
	if !strings.Contains(exposedHeaders, "X-Another-Header") {
		t.Error("Expected X-Another-Header in exposed headers")
	}
}

func TestPreflightWithMaxAge(t *testing.T) {
	c := New(Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST"},
		MaxAge:         3600,
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := c.Handler(nextHandler)

	req := httptest.NewRequest("OPTIONS", "http://example.com/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "3600" {
		t.Errorf("Expected Max-Age to be 3600, got %s", maxAge)
	}
}

func TestActualRequestWithoutOrigin(t *testing.T) {
	c := New(Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := c.Handler(nextHandler)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	// No Origin header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should still call next handler but not set CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no CORS headers without origin")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPreflightWithDisallowedMethod(t *testing.T) {
	c := New(Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := c.Handler(nextHandler)

	req := httptest.NewRequest("OPTIONS", "http://example.com/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "DELETE")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should not set CORS headers for disallowed method
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no CORS headers for disallowed method")
	}
}

func TestPreflightWithDisallowedHeaders(t *testing.T) {
	c := New(Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST"},
		AllowedHeaders: []string{"Content-Type"},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := c.Handler(nextHandler)

	req := httptest.NewRequest("OPTIONS", "http://example.com/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "X-Not-Allowed")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should not set CORS headers for disallowed headers
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no CORS headers for disallowed request headers")
	}
}

func TestPreflightWithAllowedHeaders(t *testing.T) {
	c := New(Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := c.Handler(nextHandler)

	req := httptest.NewRequest("OPTIONS", "http://example.com/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should set CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected CORS headers to be set")
	}
	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowHeaders, "Content-Type") || !strings.Contains(allowHeaders, "Authorization") {
		t.Errorf("Expected allowed headers in response, got %s", allowHeaders)
	}
}

func TestActualRequestWithDisallowedMethod(t *testing.T) {
	c := New(Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := c.Handler(nextHandler)

	req := httptest.NewRequest("DELETE", "http://example.com/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should not set CORS headers for disallowed method
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no CORS headers for disallowed method")
	}
}
