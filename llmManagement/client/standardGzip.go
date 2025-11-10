package client

import (
	"net/http"
)

// NewClient creates and returns a new http.Client that will automatically
// compress request bodies using gzip. It wraps the transport of the provided
// base client. If baseClient is nil, it wraps http.DefaultTransport.
func NewGenericGzipClient(baseClient *http.Client) *http.Client {
	// Create a new client that will have our custom transport.
	newClient := &http.Client{}
	var transport http.RoundTripper

	if baseClient != nil {
		// If a base client is provided, copy its configuration.
		transport = baseClient.Transport
		newClient.CheckRedirect = baseClient.CheckRedirect
		newClient.Jar = baseClient.Jar
		newClient.Timeout = baseClient.Timeout
	}

	// If the transport is still nil (either because baseClient was nil or
	// baseClient.Transport was nil), use the default transport.
	if transport == nil {
		transport = http.DefaultTransport
	}

	// Set the new client's transport to our gzip transport, which wraps the
	// determined underlying transport.
	newClient.Transport = &gzipTransport{
		underlyingTransport: transport,
	}

	return newClient
}
