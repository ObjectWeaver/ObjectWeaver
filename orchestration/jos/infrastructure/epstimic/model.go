package epstimic

import "objectweaver/orchestration/jos/domain"

type EpstimicEngine interface {
	Validate(results []TempResult) (TempResult, domain.ProviderMetadata, error)
}
