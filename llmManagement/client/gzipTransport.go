package client // gzipTransport is a custom http.RoundTripper that compresses request bodies

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
)

// using gzip and sets the Content-Encoding header.
type gzipTransport struct {
	// underlyingTransport is the original transport to which the request will be sent
	// after the body is compressed.
	underlyingTransport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface. It compresses the
// request body, sets the appropriate headers, and then delegates the request
// to the underlying transport.
func (t *gzipTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// If there's no body or the body is empty, there's nothing to compress.
	// We also check if the Content-Encoding header is already set to avoid
	// double-compressing the body.
	if req.Body == nil || req.Header.Get("Content-Encoding") != "" {
		return t.underlyingTransport.RoundTrip(req)
	}

	// Create a new buffer to hold the gzipped data.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	// Copy the original request body into the gzip writer.
	if _, err := io.Copy(gz, req.Body); err != nil {
		req.Body.Close() // Ensure original body is closed on error
		return nil, err
	}
	if err := gz.Close(); err != nil {
		req.Body.Close()
		return nil, err
	}

	// IMPORTANT: Close the original request body.
	req.Body.Close()

	// Create a new io.ReadCloser from our gzipped buffer.
	// This will be the new request body.
	newBody := io.NopCloser(&buf)
	req.Body = newBody

	// Set the Content-Length to the size of the compressed data.
	req.ContentLength = int64(buf.Len())

	// Set the Content-Encoding header to tell the server we've sent gzipped data.
	req.Header.Set("Content-Encoding", "gzip")

	// Delegate the request to the original transport.
	return t.underlyingTransport.RoundTrip(req)
}
