package provider

import (
	"fmt"

	goaiprovider "github.com/zendev-sh/goai/provider"
)

// DefaultProviderName is the provider used when no provider is specified by
// CLI flag or runtime configuration.
const DefaultProviderName = "anthropic"

// ResolveRequest carries the inputs to Resolve. Caller-supplied flag values
// (typically from CLI flags) take precedence over the runtime values, which in
// turn take precedence over the built-in defaults.
type ResolveRequest struct {
	// Provider is the provider identifier (e.g. "google", "anthropic"). When
	// empty, DefaultProviderName is used.
	Provider string
	// Model is the explicit model identifier. When empty and ModelClass is
	// also empty, the provider's DefaultModel is used.
	Model string
	// ModelClass selects a model by quality tier. When non-empty it overrides
	// Provider and Model; the first available entry in the class is returned.
	ModelClass string
	// APIKey is supplied to the provider registry for providers that read
	// their key from the runtime rather than the environment (currently
	// Google).
	APIKey string
}

// Resolve constructs a LanguageModel from req using the registered providers.
//
// Resolution precedence:
//
//  1. When ModelClass is set, GetModelByClass selects the first available
//     entry in the class; Provider and Model are ignored.
//  2. Otherwise the provider's configured DefaultModel is used when Model is
//     empty, and reg.GetModel constructs the model.
//
// The resolved provider name is returned alongside the model so callers can
// route subsequent requests (e.g. synthesis) through the same provider.
func Resolve(reg *ProviderRegistry, req ResolveRequest) (goaiprovider.LanguageModel, string, error) {
	providerName := req.Provider
	if providerName == "" {
		providerName = DefaultProviderName
	}

	if req.ModelClass != "" {
		model, err := GetModelByClass(reg, ModelClass(req.ModelClass))
		if err != nil {
			return nil, "", fmt.Errorf("resolve model class %q: %w", req.ModelClass, err)
		}
		return model, providerName, nil
	}

	modelName := req.Model
	if modelName == "" {
		cfg, err := reg.GetConfig(providerName)
		if err != nil {
			return nil, "", fmt.Errorf("provider %q: %w", providerName, err)
		}
		modelName = cfg.DefaultModel
	}
	if modelName == "" {
		return nil, "", fmt.Errorf("provider %q: model is required", providerName)
	}
	if providerName == "google" && req.APIKey == "" {
		return nil, "", fmt.Errorf("AI_API_KEY is required for google provider")
	}

	model, err := reg.GetModel(providerName, modelName)
	if err != nil {
		return nil, "", fmt.Errorf("get model: %w", err)
	}
	return model, providerName, nil
}
