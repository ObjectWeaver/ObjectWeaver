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
