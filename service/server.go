package service

import (
	"compress/gzip"
	"net/http"
	"net/http/pprof"
	"objectweaver/cors"
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
	s.Router.Use(cors.Handler(cors.Options{
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Session-Id", "Content-Encoding"},
		ExposedHeaders:   []string{"Link", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s.Router.Use(middleware.RequestID)
	s.Router.Use(middleware.RealIP)
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.ThrottleWithOpts(middleware.ThrottleOpts{
		Limit:          50000,            // increased from 10000 for high-load testing
		BacklogLimit:   25000,            // increased from 5000 for high-load testing
		BacklogTimeout: 60 * time.Second, // max wait time before 503
		RetryAfterFn: func(ctxDone bool) time.Duration {
			if ctxDone {
				return 0
			}
			return 1 * time.Second
		},
	}))
	s.Router.Use(GzipDecompression)                     // handle incoming gzip compressed requests
	s.Router.Use(middleware.Compress(5))                // enable gzip compression for responses
	s.Router.Use(middleware.Timeout(120 * time.Second)) // 2 minute timeout for slow LLM responses
	s.Router.Use(middleware.URLFormat)

	s.Router.Get("/health", HealthCheck)

	s.Router.Get("/metrics", PrometheusMetricsHandler)

	s.Router.Get("/debug/pprof/", http.HandlerFunc(pprof.Index))
	s.Router.Get("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	s.Router.Get("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	s.Router.Get("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	s.Router.Get("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	s.Router.Get("/debug/pprof/{cmd}", http.HandlerFunc(pprof.Index))

	s.Router.Group(func(r chi.Router) {
		r.Use(PrometheusMiddleware)
		r.Use(ValidatePassword)
		r.Post("/api/objectGen", s.ObjectGenHandler)
	})

	// if this isn't in production then don't provide this as an option
	if os.Getenv("ENVIRONMENT") == "development" {
		// Serve static files from the absolute path /static
		fileServer := http.FileServer(http.Dir("/static"))
		s.Router.Handle("/static/*", http.StripPrefix("/static", fileServer))

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
