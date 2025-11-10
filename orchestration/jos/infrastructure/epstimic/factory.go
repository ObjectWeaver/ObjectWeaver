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
package epstimic

import (
	"log"
	"objectweaver/orchestration/jos/domain"
	"os"
	"strconv"
)

func GetEpstimicEngine(generator domain.Generator) EpstimicEngine {
	llmAsJudge, err := strconv.ParseBool(os.Getenv("LLM_AS_JUDGE"))
	kMeanAsJudge, err2 := strconv.ParseBool(os.Getenv("KMEAN_AS_JUDGE"))

	// Default to false if parsing fails
	if err != nil {
		llmAsJudge = false
	}
	if err2 != nil {
		kMeanAsJudge = false
	}

	// Check K-Mean first (more specific), then LLM as judge
	if kMeanAsJudge {
		model := os.Getenv("KMEAN_EMBEDDING_MODEL")
		if model == "" {
			model = "text-embedding-3-small" // Default embedding model
		}
		log.Printf("[EpstimicFactory] Using K-Mean engine with model: %s", model)
		return NewKMeanEngine(model, generator)
	}

	if llmAsJudge {
		model := os.Getenv("LLM_JUDGE_MODEL")
		if model == "" {
			model = "gpt-4o-mini" // Default judge model
		}
		log.Printf("[EpstimicFactory] Using LLM as Judge engine with model: %s", model)
		return NewLLMAsJudge(model, generator)
	}

	// Default fallback to LLM as judge
	model := os.Getenv("LLM_JUDGE_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	log.Printf("[EpstimicFactory] Using default LLM as Judge engine with model: %s", model)
	return NewLLMAsJudge(model, generator)
}

func GetEpstimicOrchestrator(generator domain.Generator) *Orchestrator {
	workerCountEnv := os.Getenv("EPSTIMIC_WORKER_COUNT")
	workerCount, err := strconv.Atoi(workerCountEnv)
	if err != nil || workerCount <= 0 {
		workerCount = 3 // default worker count
	}

	engine := GetEpstimicEngine(generator)
	return &Orchestrator{
		epstimicEngine: engine,
		workerCount:    workerCount,
	}
}
