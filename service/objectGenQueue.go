package service

import (
	"context"
	"encoding/json"
	"net/http"
	"objectweaver/cache"
	"objectweaver/jsonSchema"
	"objectweaver/logger"
	"os"

	"github.com/google/uuid"
)

type QueuedResponse struct {
	ID string `json:"id"`
}

func (s *Server) ObjectGenQueded(w http.ResponseWriter, r *http.Request) {
	// Ensure method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if !cache.IsActive() {
		http.Error(w, "Cache is disabled", http.StatusServiceUnavailable)
		return
	}

	if os.Getenv("ENVIRONMENT") == "development" {
		logger.Printf("Request received: %s %s", r.Method, r.URL.Path)
	}

	body := &jsonSchema.RequestBody{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(body); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	//id for the redis cache
	id := generateUUID()

	// Detach from request context so the queued job isn't cancelled when the HTTP request finishes.
	queuedReq := r.Clone(context.Background())

	go func(body *jsonSchema.RequestBody, id string, req *http.Request) {
		response, err := processObjectGenRequest(*body, req)
		if err != nil {
			logger.Printf("Queued generation error for id %s: %v", id, err)
			return
		}

		c := cache.GetCache()
		payload, err := json.Marshal(response)
		if err != nil {
			logger.Printf("Queued generation marshal error for id %s: %v", id, err)
			return
		}
		if err := c.Set(id, payload); err != nil {
			logger.Printf("Queued generation cache set error for id %s: %v", id, err)
		}
	}(body, id, queuedReq)

	res := QueuedResponse{
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(res)
}

func generateUUID() string {
	uuid := uuid.New()
	return uuid.String()
}
