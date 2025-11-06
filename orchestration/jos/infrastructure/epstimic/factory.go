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
package epstimic

import (
	"objectweaver/orchestration/jos/domain"
	"os"
	"strconv"
)

func GetEpstimicEngine(generator domain.Generator) EpstimicEngine {
	llmAsJudge, err := strconv.ParseBool(os.Getenv("LLM_AS_JUDGE"))
	if err != nil {
		llmAsJudge = false
	} else if llmAsJudge {
		model := os.Getenv("LLM_JUDGE_MODEL")
		return NewLLMAsJudge(model, generator)
	}

	model := os.Getenv("LLM_JUDGE_MODEL")
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
