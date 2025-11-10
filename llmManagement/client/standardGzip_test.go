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
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewGenericGzipClient_NilBaseClient(t *testing.T) {
	client := NewGenericGzipClient(nil)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.CheckRedirect != nil {
		t.Error("Expected CheckRedirect to be nil")
	}

	if client.Jar != nil {
		t.Error("Expected Jar to be nil")
	}

	if client.Timeout != 0 {
		t.Error("Expected Timeout to be 0")
	}

	// Check transport is gzipTransport
	gzTransport, ok := client.Transport.(*gzipTransport)
	if !ok {
		t.Fatal("Expected transport to be *gzipTransport")
	}

	if gzTransport.underlyingTransport != http.DefaultTransport {
		t.Error("Expected underlying transport to be http.DefaultTransport")
	}
}

func TestNewGenericGzipClient_WithBaseClient(t *testing.T) {
	baseClient := &http.Client{
		Timeout: 10 * time.Second,
		Jar:     nil, // can set if needed
	}

	client := NewGenericGzipClient(baseClient)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.Timeout != baseClient.Timeout {
		t.Errorf("Expected Timeout %v, got %v", baseClient.Timeout, client.Timeout)
	}

	if client.Jar != baseClient.Jar {
		t.Error("Expected Jar to be copied")
	}

	// CheckRedirect is nil, so should be copied as nil

	// Check transport is gzipTransport
	gzTransport, ok := client.Transport.(*gzipTransport)
	if !ok {
		t.Fatal("Expected transport to be *gzipTransport")
	}

	// Since baseClient.Transport is nil, underlying should be DefaultTransport
	if gzTransport.underlyingTransport != http.DefaultTransport {
		t.Error("Expected underlying transport to be http.DefaultTransport")
	}
}

func TestNewGenericGzipClient_WithBaseClientCustomTransport(t *testing.T) {
	customTransport := &http.Transport{MaxIdleConns: 50}
	baseClient := &http.Client{
		Transport: customTransport,
		Timeout:   5 * time.Second,
	}

	client := NewGenericGzipClient(baseClient)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.Timeout != baseClient.Timeout {
		t.Errorf("Expected Timeout %v, got %v", baseClient.Timeout, client.Timeout)
	}

	// Check transport is gzipTransport
	gzTransport, ok := client.Transport.(*gzipTransport)
	if !ok {
		t.Fatal("Expected transport to be *gzipTransport")
	}

	if !reflect.DeepEqual(gzTransport.underlyingTransport, customTransport) {
		t.Error("Expected underlying transport to be the custom transport")
	}
}

func TestNewGenericGzipClient_CompressesRequest(t *testing.T) {
	// Create a test server that checks if the request body is gzipped
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Error("Expected Content-Encoding: gzip")
		}

		// Decompress and check body
		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gzReader.Close()

		body, err := io.ReadAll(gzReader)
		if err != nil {
			t.Fatalf("Failed to read decompressed body: %v", err)
		}

		expected := "test data for compression"
		if string(body) != expected {
			t.Errorf("Expected body '%s', got '%s'", expected, string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewGenericGzipClient(nil)

	data := "test data for compression"
	req, err := http.NewRequest("POST", server.URL, strings.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestNewGenericGzipClient_NoCompressionForEmptyBody(t *testing.T) {
	// Test that requests with no body or empty body are not compressed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "" {
			t.Error("Expected no Content-Encoding header for empty body")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		if len(body) != 0 {
			t.Errorf("Expected empty body, got '%s'", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewGenericGzipClient(nil)

	req, err := http.NewRequest("POST", server.URL, nil) // No body
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
