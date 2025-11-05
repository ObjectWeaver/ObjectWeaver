package epstimic

import (
	"encoding/json"
	"fmt"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

//this will process the list of temp results and return the best one along with the updated metadata containing the information from all results

type LLMasJudge struct {
	model     string // set the model globally or use the model that was used to generate the results ie from the metadata
	generator domain.Generator
}

func NewLLMAsJudge(model string, generator domain.Generator) EpstimicEngine {
	return &LLMasJudge{
		model:     model,
		generator: generator,
	}
}

type ScoreResult struct {
	Completeness int `json:"completeness"` //from the llm judge
	Correctness  int `json:"correctness"`  // from the llm judge
	Score        int `json:"score"`        // the average of the two scores
	Result       TempResult
}

type ScoredResults []ScoreResult

func (s ScoredResults) Best() (TempResult, int) {
	if len(s) == 0 {
		return TempResult{}, 0
	}

	best := s[0]
	for _, score := range s[1:] {
		if score.Score > best.Score {
			best = score
		}
	}
	return best.Result, best.Score
}

func (j *LLMasJudge) Validate(results []TempResult) (TempResult, domain.ProviderMetadata, error) {
	scoredResults := make(chan *ScoreResult, len(results))

	for _, result := range results {
		go func(result TempResult) {
			if result.Error != nil {
				scoredResults <- nil
				return
			}

			scoreResult := j.assessCompletion(result)
			if scoreResult == nil {
				scoredResults <- nil
				return
			}

			scoredResults <- &ScoreResult{
				Completeness: scoreResult.Completeness,
				Correctness:  scoreResult.Correctness,
				Score:        scoreResult.Score,
				Result:       result,
			}
		}(result)
	}

	scores := ScoredResults{}

	// Collect exactly len(results) items from the channel
	for i := 0; i < len(results); i++ {
		score := <-scoredResults
		if score != nil {
			scores = append(scores, *score)
		}
	}

	best, _ := scores.Best()

	// Check if we have a valid result with metadata
	if best.Metadata == nil {
		return TempResult{}, domain.ProviderMetadata{}, fmt.Errorf("no valid results to validate")
	}

	best.Metadata.Choices = convertToChoice(scores)

	return best, *best.Metadata, nil
}

func (j *LLMasJudge) assessCompletion(result TempResult) *ScoreResult {
	// assess the result value and return a score - this could be based on length, presence of certain keywords, etc.
	prompt := j.createJudgePrompt(result.Metadata.Prompt, result.Value)
	schema := j.getJudgeDefinition(result)

	// Use the Generator interface's Generate method
	req := domain.NewGenerationRequest(prompt, schema)
	genRes, err := j.generator.Generate(req)
	if err != nil {
		return nil
	}

	jsonBytes, err := json.Marshal(genRes.Data())
	if err != nil {
		return nil
	}

	var res ScoreResult
	err = json.Unmarshal(jsonBytes, &res)
	if err != nil {
		return nil
	}

	res.Score = (res.Completeness + res.Correctness) / 2

	// for now, return a dummy score
	return &res
}

func (j *LLMasJudge) getJudgeDefinition(result TempResult) *jsonSchema.Definition {
	var model string
	if j.model != "" {
		model = j.model
	} else {
		model = result.Metadata.Model
	}

	return &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: fmt.Sprintf("You are an expert evaluator. Assess the quality of the completion based on the original prompt. Use model: %s", model),
		Model:       model,
		Epistemic: jsonSchema.EpistemicValidation{
			Active: false,
			Judges: 3,
		},
		Properties: map[string]jsonSchema.Definition{
			"completeness": {
				Type:        jsonSchema.Integer,
				Instruction: "Evaluate how completely the response addresses all aspects of the prompt. Score from 0 (incomplete) to 100 (fully complete).",
				Seed: infrastructure.GenerateSeed(),
			},
			"correctness": {
				Type:        jsonSchema.Integer,
				Instruction: "Evaluate the accuracy and correctness of the information in the response. Score from 0 (incorrect) to 100 (fully correct).",
				Seed: infrastructure.GenerateSeed(),
			},
		},
	}
}

func (j *LLMasJudge) createJudgePrompt(prompt string, completion any) string {
	// create the prompt for the judge LLM based on the original prompt and the completion
	return fmt.Sprintf("Prompt:\n%s\n\nCompletion:\n%v", prompt, completion)
}

func convertToChoice(results ScoredResults) []domain.Choice {
	// convert TempResult to Choice
	var choices []domain.Choice
	for _, res := range results {
		// Skip results with nil metadata or task
		if res.Result.Metadata == nil || res.Result.Task == nil {
			continue
		}

		choice := domain.Choice{
			Prompt:     res.Result.Metadata.Prompt,
			Completion: res.Result.Value,
			FieldTask:  *res.Result.Task,
			Model:      res.Result.Metadata.Model,
			Score:      res.Score,
			Confidence: float64(res.Score) / 100.0,
		}
		choices = append(choices, choice)
	}

	return choices
}
