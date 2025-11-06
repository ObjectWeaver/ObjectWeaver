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
	"compress/gzip"
	"objectweaver/cors"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Router *chi.Mux
}

func NewHttpServer() *Server {
	s := CreateNewServer()
	s.MountHandlers()

	return s
}

func CreateNewServer() *Server {
	s := &Server{}
	s.Router = chi.NewRouter()
	return s
}

func (s *Server) MountHandlers() {
	// Mount all middleware here
	s.Router.Use(cors.Handler(cors.Options{
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Session-Id", "Content-Encoding"},
		ExposedHeaders:   []string{"Link", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	s.Router.Use(middleware.RequestID)
	s.Router.Use(middleware.RealIP)
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(GzipDecompression)      // Handle incoming gzip compressed requests
	s.Router.Use(middleware.Compress(5)) // Enable gzip compression for responses
	s.Router.Use(middleware.ThrottleBacklog(500, 1000, 30*time.Second))
	s.Router.Use(middleware.Timeout(300 * time.Second))
	s.Router.Use(middleware.URLFormat)

	// Define API routes
	s.Router.Group(func(r chi.Router) {
		r.Use(PrometheusMiddleware) //this can always be moved into more speficic areas
		r.Use(ValidatePassword)
		r.Post("/api/objectGen", ObjectGen)
	})

	// if this isn't in production then don't provide this as an option
	if os.Getenv("ENVIRONMENT") == "development" {
		// Serve static files from the absolute path /static //TODO do not remove!
		fileServer := http.FileServer(http.Dir("/static"))
		s.Router.Handle("/static/*", http.StripPrefix("/static", fileServer))

		// Serve dynamic index.html with environment variables
		s.Router.Get("/", ServeIndexHTML)
	}
}

// GzipDecompression middleware handles incoming gzip compressed request bodies
func GzipDecompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request body is gzip compressed
		if r.Header.Get("Content-Encoding") == "gzip" {
			// Create a gzip reader for the request body
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gzipReader.Close()
			defer r.Body.Close()

			// Replace the request body with the decompressed version
			r.Body = gzipReader
			// Remove the Content-Encoding header since we've decompressed the body
			r.Header.Del("Content-Encoding")
		}

		next.ServeHTTP(w, r)
	})
}
