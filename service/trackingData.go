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
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Define custom metrics
var (
	// Track the total number of HTTP requests
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status_code"},
	)

	// Track request duration
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Track the size of HTTP responses
	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6), // From 100 bytes to ~1MB
		},
		[]string{"method", "path"},
	)

	// Track the size of HTTP requests
	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP requests in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6), // From 100 bytes to ~1MB
		},
		[]string{"method", "path"},
	)

	// Track the number of active requests
	activeRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_requests",
			Help: "Number of active HTTP requests.",
		},
	)

	// Track the number of errors
	errorRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_requests_total",
			Help: "Total number of error HTTP requests.",
		},
		[]string{"method", "path", "status_code"},
	)

	// Track the number of goroutines
	goroutines = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "goroutines",
			Help: "Number of goroutines.",
		},
		func() float64 {
			return float64(runtime.NumGoroutine())
		},
	)

	// Track memory usage
	memoryUsage = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Current memory usage in bytes.",
		},
		func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.Alloc) // Alloc is the total bytes allocated
		},
	)
)

func init() {
	// Register custom metrics with Prometheus
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(httpResponseSize)
	prometheus.MustRegister(httpRequestSize)
	prometheus.MustRegister(activeRequests)
	prometheus.MustRegister(errorRequestsTotal)
	prometheus.MustRegister(goroutines)
	prometheus.MustRegister(memoryUsage) // Register the memory usage metric
}

// Custom response writer to capture response size
type responseWriter struct {
	http.ResponseWriter
	Size       int `json:"size"`
	StatusCode int `json:"status_code"`
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.Size += size
	return size, err
}

func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		activeRequests.Inc()
		defer activeRequests.Dec()

		requestSize, _ := strconv.Atoi(r.Header.Get("Content-Length"))

		rw := &responseWriter{ResponseWriter: w, StatusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		statusCode := rw.StatusCode
		if statusCode >= 400 {
			errorRequestsTotal.WithLabelValues(r.Method, r.URL.Path, http.StatusText(statusCode)).Inc()
		}

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, http.StatusText(statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		httpResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(rw.Size))
		httpRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(requestSize))
	})
}
