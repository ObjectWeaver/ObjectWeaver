
package epstimic

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

type KMeanEngine struct {
	model     string // set the model globally or use the model that was used to generate the results ie from the metadata
	generator domain.Generator
}

type KMeanResult struct {
	Completion string
	Embedding  []float32
	Metadata   domain.ProviderMetadata
	Task       *domain.FieldTask
}

func NewKMeanEngine(model string, generator domain.Generator) EpstimicEngine {
	return &KMeanEngine{
		model:     model,
		generator: generator,
	}
}

func (k *KMeanEngine) Validate(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
	log.Printf("[KMeanEngine] Starting validation with %d results", len(results))

	kResults := make(chan *KMeanResult, len(results))

	for _, res := range results {
		go func(r TempResult) {
			if r.Error != nil {
				log.Printf("[KMeanEngine] Skipping result due to error: %v", r.Error)
				kResults <- nil
				return
			}

			kResults <- k.getEmbeddingForResult(r)
		}(res)
	}

	kMeanResults := []KMeanResult{}
	for i := 0; i < len(results); i++ {
		kRes := <-kResults
		if kRes != nil {
			kMeanResults = append(kMeanResults, *kRes)
		}
	}

	log.Printf("[KMeanEngine] Successfully generated embeddings for %d out of %d results", len(kMeanResults), len(results))

	bestResult, metadata, err := k.calculateKMean(kMeanResults)
	if err != nil {
		log.Printf("[KMeanEngine ERROR] Failed to calculate k-mean: %v", err)
		return TempResult{}, domain.ProviderMetadata{}, err
	}

	return bestResult, metadata, nil
}

func (k *KMeanEngine) createEmeddingDefinition() *jsonSchema.Definition {
	return &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"embedding": {
				Type:  jsonSchema.Vector,
				Model: k.model,
			},
		},
	}
}

func (k *KMeanEngine) getEmbeddingForResult(result TempResult) *KMeanResult {
	embeddingDef := k.createEmeddingDefinition()

	generationRequest := domain.NewGenerationRequest(result.Value.(string), embeddingDef)
	genRes, err := k.generator.Generate(generationRequest)
	if err != nil {
		log.Printf("[KMeanEngine ERROR] Failed to generate embedding: %v", err)
		return nil
	}

	type EmbeddingResponse struct {
		Embedding []float32 `json:"embedding"`
	}

	jsonBytes, err := json.Marshal(genRes.Data())
	if err != nil {
		log.Printf("[KMeanEngine ERROR] Failed to marshal embedding response: %v", err)
		return nil
	}

	var res EmbeddingResponse
	err = json.Unmarshal(jsonBytes, &res)
	if err != nil {
		log.Printf("[KMeanEngine ERROR] Failed to unmarshal embedding response: %v", err)
		return nil
	}

	log.Printf("[KMeanEngine] Generated embedding vector of size %d", len(res.Embedding))

	return &KMeanResult{
		Completion: result.Value.(string),
		Embedding:  res.Embedding,
		Metadata:   *result.Metadata,
		Task:       result.Task,
	}
}

// calculateKMean finds the result whose embedding is closest to the average of all embeddings
func (k *KMeanEngine) calculateKMean(kMeanResults []KMeanResult) (TempResult, domain.ProviderMetadata, error) {
	if len(kMeanResults) == 0 {
		return TempResult{}, domain.ProviderMetadata{}, fmt.Errorf("no valid results to calculate k-mean")
	}

	log.Printf("[KMeanEngine] Calculating average embedding from %d results", len(kMeanResults))

	// Calculate the average embedding vector
	avgEmbedding := k.calculateAverageEmbedding(kMeanResults)

	log.Printf("[KMeanEngine] Comparing all results to find closest to average (embedding size: %d)", len(avgEmbedding))

	// Find the result closest to the average
	bestIndex := 0
	bestDistance := k.euclideanDistance(kMeanResults[0].Embedding, avgEmbedding)

	for i := 1; i < len(kMeanResults); i++ {
		distance := k.euclideanDistance(kMeanResults[i].Embedding, avgEmbedding)
		log.Printf("[KMeanEngine] Result %d distance from average: %.6f", i, distance)
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = i
		}
	}

	log.Printf("[KMeanEngine] Best result index: %d with distance: %.6f from average", bestIndex, bestDistance)

	bestResult := kMeanResults[bestIndex]

	// Create a copy of the metadata to modify
	metadata := bestResult.Metadata

	// Store all results in metadata choices similar to LLM as judge
	metadata.Choices = k.convertToChoices(kMeanResults, avgEmbedding)

	log.Printf("[KMeanEngine] Selected completion (first 100 chars): %s...",
		truncateString(bestResult.Completion, 100))
	log.Printf("[KMeanEngine] Stored %d choices in metadata", len(metadata.Choices))

	// Create TempResult from the best KMeanResult
	tempResult := TempResult{
		Task:     bestResult.Task,
		Value:    bestResult.Completion,
		Metadata: &metadata,
		Error:    nil,
	}

	return tempResult, metadata, nil
}

// calculateAverageEmbedding computes the average embedding vector from all results
func (k *KMeanEngine) calculateAverageEmbedding(kMeanResults []KMeanResult) []float32 {
	if len(kMeanResults) == 0 {
		return nil
	}

	embeddingSize := len(kMeanResults[0].Embedding)
	avgEmbedding := make([]float32, embeddingSize)

	// Sum all embeddings
	for _, result := range kMeanResults {
		for i, val := range result.Embedding {
			avgEmbedding[i] += val
		}
	}

	// Divide by count to get average
	count := float32(len(kMeanResults))
	for i := range avgEmbedding {
		avgEmbedding[i] /= count
	}

	return avgEmbedding
}

// euclideanDistance calculates the Euclidean distance between two embedding vectors
func (k *KMeanEngine) euclideanDistance(embedding1, embedding2 []float32) float64 {
	if len(embedding1) != len(embedding2) {
		return math.MaxFloat64
	}

	var sum float64
	for i := range embedding1 {
		diff := float64(embedding1[i] - embedding2[i])
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

// convertToChoices converts KMeanResults to domain.Choice format with distance scores
func (k *KMeanEngine) convertToChoices(kMeanResults []KMeanResult, avgEmbedding []float32) []domain.Choice {
	choices := make([]domain.Choice, 0, len(kMeanResults))

	// Find max distance for normalization
	maxDistance := 0.0
	for _, r := range kMeanResults {
		d := k.euclideanDistance(r.Embedding, avgEmbedding)
		if d > maxDistance {
			maxDistance = d
		}
	}

	log.Printf("[KMeanEngine] Converting %d results to choices (max distance: %.6f)", len(kMeanResults), maxDistance)

	for i, result := range kMeanResults {
		distance := k.euclideanDistance(result.Embedding, avgEmbedding)

		// Convert distance to a confidence score (closer = higher confidence)
		// Using inverse distance, normalized to 0-1 range
		confidence := 1.0
		if maxDistance > 0 {
			confidence = 1.0 - (distance / maxDistance)
		}

		// Score as integer (0-100 scale)
		score := int(confidence * 100)

		log.Printf("[KMeanEngine] Choice %d: distance=%.6f, confidence=%.2f, score=%d",
			i, distance, confidence, score)

		// Convert float32 embeddings to float64 for domain.Choice
		embedding64 := make([]float64, len(result.Embedding))
		for j, v := range result.Embedding {
			embedding64[j] = float64(v)
		}

		choice := domain.Choice{
			Prompt:     result.Metadata.Prompt,
			Completion: result.Completion,
			FieldTask:  *result.Task,
			Model:      result.Metadata.Model,
			Score:      score,
			Confidence: confidence,
			Embedding:  embedding64,
		}
		choices = append(choices, choice)
	}

	return choices
}

// truncateString truncates a string to a maximum length, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
