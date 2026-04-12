package service

import (
	"encoding/json"
	"net/http"
	"github.com/ObjectWeaver/ObjectWeaver/logger"
	texttoweaver "github.com/ObjectWeaver/ObjectWeaver/textToWeaver"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

type TtwRequest struct {
	Prompt string
}

type TtwResponse struct {
	Definition jsonSchema.Definition `json:"definition"`
}

func (s *Server) TextToWeaver(w http.ResponseWriter, r *http.Request) {

	//get the data out of the request
	req := &TtwRequest{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	//create the request body
	body := jsonSchema.RequestBody{
		Prompt:     req.Prompt,
		Definition: texttoweaver.DefinitionForDefinition(),
	}

	response, err := processObjectGenRequest(body, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Marshal response.Data to JSON bytes
	bytes, err := json.Marshal(response.Data)
	if err != nil {
		http.Error(w, "Failed to marshal response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal into our intermediate type
	rawOutput := &texttoweaver.TtwRawOutput{}
	if err := json.Unmarshal(bytes, rawOutput); err != nil {
		http.Error(w, "Failed to parse LLM output: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Debug logging to see what the LLM actually returned
	logger.Printf("[TextToWeaver] Raw LLM output - StructuralAnalysis: %s", rawOutput.StructuralAnalysis)
	logger.Printf("[TextToWeaver] Raw LLM output - Analysis: %s", rawOutput.Analysis)
	logger.Printf("[TextToWeaver] Raw LLM output - FieldCount: %d, ActualFields: %d", len(rawOutput.Fields), len(rawOutput.Fields))
	for i, f := range rawOutput.Fields {
		logger.Printf("[TextToWeaver] Field[%d]: name=%s, type=%s, isComplex=%v, nestedCount=%d, nestedFieldsLen=%d",
			i, f.FieldName, f.FieldType, f.IsComplex, f.NestedCount, len(f.NestedFields))
	}

	// Post-process to convert to proper Definition structure
	// Pass the original prompt for contextual instructions
	definition := texttoweaver.ConvertRawOutputToDefinition(rawOutput, req.Prompt)

	// Build the final response
	res := &TtwResponse{
		Definition: *definition,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		logger.Printf("Error encoding response (context error: %v): %v", r.Context().Err(), err)
	}
}
