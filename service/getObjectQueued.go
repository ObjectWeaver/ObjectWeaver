package service

import (
	"net/http"
	"objectweaver/cache"
	"objectweaver/logger"
	"os"
)

func (s *Server) GetObjectQueued(w http.ResponseWriter, r *http.Request) {
	// Ensure method is GET
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if !cache.IsActive() {
		http.Error(w, "Cache is disabled", http.StatusServiceUnavailable)
		return
	}

	if os.Getenv("ENVIRONMENT") == "development" {
		logger.Printf("Request received: %s %s", r.Method, r.URL.Path)
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	c := cache.GetCache()
	result, err := c.Get(id)
	if err != nil {
		http.Error(w, "Error retrieving result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if result == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	//check if this is a good idea or not.
	if err := c.Delete(id); err != nil {
		logger.Printf("Cache delete error for id %s: %v", id, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}
