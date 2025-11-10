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
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/httptransport"
	"google.golang.org/genai"
)

// NewGzipClient creates and returns a new http.Client that will automatically
// compress request bodies using gzip. It intelligently creates the correct
// underlying http.Client (e.g., for Vertex AI authentication) based on the
// provided ClientConfig.
func NewGzipClient(ctx context.Context, cc *genai.ClientConfig) (*http.Client, error) {
	var baseClient *http.Client

	// Replicate the client creation logic from the genai library to ensure
	// we create the correct type of client (e.g., authenticated for Vertex).
	if cc.Backend == genai.BackendVertexAI && cc.APIKey == "" {
		// This block handles the specific case for Vertex AI using Application
		// Default Credentials, which requires an authenticated transport.

		// First, ensure credentials exist. If not provided, detect them.
		if cc.Credentials == nil {
			creds, err := credentials.DetectDefault(&credentials.DetectOptions{
				Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to find default credentials for Vertex AI: %w", err)
			}
			cc.Credentials = creds
		}

		quotaProjectID, err := cc.Credentials.QuotaProjectID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get quota project ID: %w", err)
		}

		// Create the authenticated client using httptransport.
		baseClient, err = httptransport.NewClient(&httptransport.Options{
			Credentials: cc.Credentials,
			Headers: http.Header{
				"X-Goog-User-Project": []string{quotaProjectID},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated HTTP client for Vertex AI: %w", err)
		}
	} else {
		// For the Gemini API or Vertex with an API Key, a standard client is sufficient.
		baseClient = &http.Client{}
	}

	// Now that we have the correct base client (either standard or authenticated),
	// we can wrap its transport with our gzip transport.
	transport := baseClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// Return a new client with our custom gzip transport.
	return &http.Client{
		Transport: &gzipTransport{
			underlyingTransport: transport,
		},
		// Copy other fields from the base client.
		CheckRedirect: baseClient.CheckRedirect,
		Jar:           baseClient.Jar,
		Timeout:       baseClient.Timeout,
	}, nil
}
