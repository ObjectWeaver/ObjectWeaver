package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"objectweaver/checks"
	"objectweaver/logger"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
	"os"
	"sync"

	"github.com/objectweaver/go-sdk/client"
)

// DetailedField contains both the value and metadata for a field
type DetailedField struct {
	Value    interface{}            `json:"value"`
	Metadata *domain.ResultMetadata `json:"metadata"`
}

// Response struct with both simple data and detailed metadata
type Response struct {
	Data         map[string]any            `json:"data"`
	DetailedData map[string]*DetailedField `json:"detailedData,omitempty"`
	UsdCost      float64                   `json:"usdCost"`
}

// In objectGen.go
var generatorCache sync.Pool

func getGenerator() domain.Generator {
	if g := generatorCache.Get(); g != nil {
		return g.(domain.Generator)
	}
	// Create new generator
	generatorFactory := factory.NewGeneratorFactory(&factory.GeneratorConfig{
		Mode: factory.ModeParallel,
	})
	generator, err := generatorFactory.Create()
	if err != nil {
		panic(err)
	}
	generatorCache.Put(generator)
	return generator
}

func returnGenerator(g domain.Generator) {
	generatorCache.Put(g)
}

// ObjectGenHandler is a method on Server that uses the singleton generator
func (s *Server) ObjectGenHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if os.Getenv("ENVIRONMENT") == "development" {
		logger.Printf("Request received: %s %s", r.Method, r.URL.Path)
	}

	// Check if context is already cancelled before doing any work
	if r.Context().Err() != nil {
		logger.Printf("Request context already cancelled at entry: %v", r.Context().Err())
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
		logger.Printf("Error decoding request body: %v", err)
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

	response, err := processObjectGenRequest(*body, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if responseJSON, err := json.Marshal(response); err == nil {
		logger.Printf("[ObjectGen] Final response JSON: %s", string(responseJSON))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Printf("Error encoding response (context error: %v): %v", r.Context().Err(), err)
	}
}

func PrettyPrintJSON(jsonBytes []byte) {
	var jsonObj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
		logger.Printf("Error unmarshalling JSON for pretty print: %v", err) // Use logger.Printf, not Fatalf for a lib func
		return
	}
	prettyJSON, err := json.MarshalIndent(jsonObj, "", "    ")
	if err != nil {
		logger.Printf("Error marshalling JSON for pretty print: %v", err)
		return
	}
	fmt.Println(string(prettyJSON))
}

func processObjectGenRequest(body client.RequestBody, r *http.Request) (*Response, error) {
	generator := getGenerator()
	defer returnGenerator(generator)

	// Generate
	request := domain.NewGenerationRequest(body.Prompt, body.Definition).WithContext(r.Context())
	result, err := generator.Generate(request)
	if err != nil {
		logger.Printf("Error during generation: %v", err)
		if r.Context().Err() == nil {
			return nil, errors.New("context ran out")
		}
		return nil, err
	}

	data := result.Data()
	cost := result.Metadata().Cost

	// Log the result data for debugging
	logger.Printf("[ObjectGen] Result data: %+v", data)
	logger.Printf("[ObjectGen] Result cost: %f", cost)
	logger.Printf("[ObjectGen] Result metadata: %+v", result.Metadata())

	// Build detailed data structure if available
	var detailedData map[string]*DetailedField
	if result.HasDetailedData() {
		detailedData = make(map[string]*DetailedField)
		for key, fieldResult := range result.DetailedData() {
			detailedData[key] = &DetailedField{
				Value:    fieldResult.Value,
				Metadata: fieldResult.Metadata,
			}
		}
	}

	// Marshal and send the successful response
	return &Response{
		Data:         data,
		DetailedData: detailedData,
		UsdCost:      cost,
	}, nil
}