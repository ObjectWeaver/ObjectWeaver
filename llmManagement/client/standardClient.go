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
