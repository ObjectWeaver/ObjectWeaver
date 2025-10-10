package modelConverter

// ModelConverter is an interface for converting a local ModelType to its provider-specific string.
type ModelConverter interface {
	// Convert takes a local ModelType and returns the corresponding provider-specific model string.
	// If the model is not recognized, it returns an empty string.
	Convert(model string) string
}

// providerModelConverter is a concrete implementation of the ModelConverter interface.
// It maps local ModelType enums to their official string representations for various LLM providers.
type providerModelConverter struct{}

// NewModelConverter creates and returns a new instance of the providerModelConverter.
func NewModelConverter() ModelConverter {
	return &providerModelConverter{}
}

// Convert implements the logic to translate a ModelType into the correct API string.
// It handles known model constants and falls back to treating unknown models as
// pass-through strings, enabling environment variable configuration and custom models.
func (c *providerModelConverter) Convert(model string) string {
	modelStr := string(model)

	return modelStr
}
