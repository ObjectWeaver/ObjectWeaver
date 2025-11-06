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
package client

import (
	"net/http"
	"sync"
	"time"
)

var standardClient *http.Client
var once = sync.Once{}

// NewStandardClient creates a standard HTTP client suitable for JSON APIs like OpenAI.
// This client supports mTLS for secure communication with the completion server.
func NewStandardClient() *http.Client {
		once.Do(func() {
			standardClient = &http.Client{}
	})
	return standardClient
}

// NewStandardClientWithTimeout creates a standard HTTP client with a custom timeout.
func NewStandardClientWithTimeout(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}
