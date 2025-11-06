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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"objectweaver/checks"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
	"os"

	"github.com/objectweaver/go-sdk/client"
)

// Create a response struct
type Response struct {
	Data    map[string]any `json:"data"`
	UsdCost float64        `json:"usdCost"`
}

func ObjectGen(w http.ResponseWriter, r *http.Request) {
	// Ensure method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if os.Getenv("ENVIRONMENT") == "development" {
		log.Printf("Request received: %s %s", r.Method, r.URL.Path)
	}

	// Check if context is already cancelled before doing any work
	if r.Context().Err() != nil {
		log.Printf("Request context already cancelled at entry: %v", r.Context().Err())
		// Depending on which middleware cancelled it, a response might have been sent.
		// It's often safer to just return here if you expect a middleware to handle the response.
		// If you are sure no response has been sent, you could write one, e.g.:
		switch r.Context().Err() {
		case context.Canceled:
			http.Error(w, "Request cancelled by client", http.StatusRequestTimeout)
		case context.DeadlineExceeded:
			http.Error(w, "Request timed out", http.StatusRequestTimeout)
		default:
			http.Error(w, "Request context error", http.StatusInternalServerError)
		}
		return
	}

	// Parse the request body
	body := &client.RequestBody{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		log.Printf("Error decoding request body: %v", err)
		// Check context before writing, in case of timeout during body read/decode.
		if r.Context().Err() == nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
		}
		return
	}
	defer r.Body.Close()

	// Check for circular definitions
	if checks.CheckCircularDefinitions(body.Definition) {
		if r.Context().Err() == nil {
			http.Error(w, "Circular object definition", http.StatusUnprocessableEntity)
		}
		return
	}

	// Create factory
	factory := factory.NewGeneratorFactory(&factory.GeneratorConfig{
		Mode:           factory.ModeParallel, // Uses new recursive architecture by default
		MaxConcurrency: 10,
	})

	// Create generator
	generator, err := factory.Create()
	if err != nil {
		log.Printf("Error creating generator: %v", err)
		if r.Context().Err() == nil {
			http.Error(w, "Failed to create generator: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Generate
	request := domain.NewGenerationRequest(body.Prompt, body.Definition)
	result, err := generator.Generate(request)
	if err != nil {
		log.Printf("Error during generation: %v", err)
		if r.Context().Err() == nil {
			http.Error(w, "Generation failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	data := result.Data()
	cost := result.Metadata().Cost

	// Log the result data for debugging
	log.Printf("[ObjectGen] Result data: %+v", data)
	log.Printf("[ObjectGen] Result cost: %f", cost)
	log.Printf("[ObjectGen] Result metadata: %+v", result.Metadata())

	// Marshal and send the successful response
	response := Response{
		Data:    data,
		UsdCost: cost,
	}

	// Log the final JSON response being sent to client
	if responseJSON, err := json.Marshal(response); err == nil {
		log.Printf("[ObjectGen] Final response JSON: %s", string(responseJSON))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// This error might occur if the client disconnected or if (less likely by now)
		// another component wrote a response.
		log.Printf("Error encoding response (context error: %v): %v", r.Context().Err(), err)
		// Don't try http.Error here as headers might have been partially written.
	}
}

// PrettyPrintJSON (no changes needed, but ensure it's used safely if it involves I/O that could fail on closed connections)
func PrettyPrintJSON(jsonBytes []byte) {
	var jsonObj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
		log.Printf("Error unmarshalling JSON for pretty print: %v", err) // Use log.Printf, not Fatalf for a lib func
		return
	}
	prettyJSON, err := json.MarshalIndent(jsonObj, "", "    ")
	if err != nil {
		log.Printf("Error marshalling JSON for pretty print: %v", err)
		return
	}
	fmt.Println(string(prettyJSON))
}
