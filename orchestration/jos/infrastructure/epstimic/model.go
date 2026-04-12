package epstimic

import "github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"

type EpstimicEngine interface {
	Validate(results []TempResult) (TempResult, domain.ProviderMetadata, error)
}
