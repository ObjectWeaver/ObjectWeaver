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
package service

import (
	"log"
	"net/http"
	"os"
	"strings"
)

// ValidateDefinitionMiddleware is a middleware that extracts the Bearer token from the Authorization header
// and checks if it matches the PASSWORD environment variable. If not, it rejects the request.
//
//garble:controlflow flatten_passes=3 junk_jumps=max block_splits=max flatten_hardening=xor,delegate_table
func ValidatePassword(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if os.Getenv("ENVIRONMENT") == "development" {
			// In development mode, skip password validation
			next.ServeHTTP(w, r)
			return
		}

		// Extract the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("No Authorization header found")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if it is a Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Println("Invalid token")
			log.Println("auth header: ", authHeader)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract the token value
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Get the PASSWORD environment variable
		envPassword := os.Getenv("PASSWORD")
		if envPassword == "" {
			log.Println("Missing environment variable PASSWORD")
			http.Error(w, "Server error: PASSWORD environment variable not set", http.StatusInternalServerError)
			return
		}

		// Compare the token with the PASSWORD environment variable
		if token != envPassword {
			log.Println("The token does not match")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// If everything is OK, proceed to the next handler
		next.ServeHTTP(w, r)
	})
}
