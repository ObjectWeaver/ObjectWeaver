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
// <https://objectweaver.dev/licensing/server-side-public-license>.
package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name           string
		env            map[string]string
		authHeader     string
		expectedStatus int
		expectedBody   string
		nextCalled     bool
	}{
		{
			name:           "development skips validation",
			env:            map[string]string{"ENVIRONMENT": "development"},
			authHeader:     "",
			expectedStatus: 200,
			expectedBody:   "next",
			nextCalled:     true,
		},
		{
			name:           "no authorization header",
			env:            map[string]string{"PASSWORD": "secret"},
			authHeader:     "",
			expectedStatus: 401,
			expectedBody:   "Unauthorized\n",
			nextCalled:     false,
		},
		{
			name:           "invalid bearer token format",
			env:            map[string]string{"PASSWORD": "secret"},
			authHeader:     "Basic token",
			expectedStatus: 401,
			expectedBody:   "Unauthorized\n",
			nextCalled:     false,
		},
		{
			name:           "password environment variable not set",
			env:            map[string]string{},
			authHeader:     "Bearer token",
			expectedStatus: 500,
			expectedBody:   "Server error: PASSWORD environment variable not set\n",
			nextCalled:     false,
		},
		{
			name:           "token does not match password",
			env:            map[string]string{"PASSWORD": "secret"},
			authHeader:     "Bearer wrong",
			expectedStatus: 401,
			expectedBody:   "Unauthorized\n",
			nextCalled:     false,
		},
		{
			name:           "correct token",
			env:            map[string]string{"PASSWORD": "secret"},
			authHeader:     "Bearer secret",
			expectedStatus: 200,
			expectedBody:   "next",
			nextCalled:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.env {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Create a new HTTP request
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Create a response recorder
			w := httptest.NewRecorder()

			// Track if the next handler was called
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("next"))
			})

			// Apply the middleware
			handler := ValidatePassword(next)
			handler.ServeHTTP(w, req)

			// Check the status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check the response body
			if w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, w.Body.String())
			}

			// Check if the next handler was called
			if nextCalled != tt.nextCalled {
				t.Errorf("expected next called %v, got %v", tt.nextCalled, nextCalled)
			}
		})
	}
}
