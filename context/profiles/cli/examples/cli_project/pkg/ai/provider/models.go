package provider

import (
	"fmt"

	goaiprovider "github.com/zendev-sh/goai/provider"
)

// ModelEntry represents a model available from a specific provider.
type ModelEntry struct {
	Provider string
	Model    string
}

// modelClasses maps each model class to an ordered list of provider:model
// pairs. Initialized once at package level; not exported to prevent
// external mutation.
var modelClasses = map[ModelClass][]ModelEntry{
	ClassGenius: {
		{Provider: "anthropic", Model: "claude-opus-4.6"},
	},
	ClassPro: {
		{Provider: "glm5", Model: "glm-5.1"},
		{Provider: "glm5", Model: "glm-5-turbo"},
		{Provider: "glm47", Model: "glm-4.7"},
		{Provider: "glm47", Model: "glm-4.7-flash"},
	},
	ClassFlash: {
		{Provider: "glm5", Model: "glm-5-turbo"},
		{Provider: "glm47", Model: "glm-4.7-flash"},
	},
	ClassNano: {
		{Provider: "glm47", Model: "glm-4.7-flash"},
	},
}

// GetModelByClass returns the first LanguageModel in the class list whose
// provider is registered and not disabled. Returns an error when the class
// is unknown or when no entry in the class is currently available.
func GetModelByClass(r *ProviderRegistry, class ModelClass) (goaiprovider.LanguageModel, error) {
	entries, ok := modelClasses[class]
	if !ok {
		return nil, fmt.Errorf("unknown model class %q", class)
	}
	var lastErr error
	for _, entry := range entries {
		if _, err := r.GetConfig(entry.Provider); err != nil {
			continue
		}
		if !IsProviderEnabled(entry.Provider) {
			continue
		}
		model, err := r.GetModel(entry.Provider, entry.Model)
		if err != nil {
			lastErr = err
			continue
		}
		return model, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("class %q: no available model (last error: %w)", class, lastErr)
	}
	return nil, fmt.Errorf("class %q: no available model", class)
}

// getActiveModels returns the subset of modelClasses[class] whose providers
// are registered and enabled. The result preserves class-list order.
func getActiveModels(r *ProviderRegistry, class ModelClass) []ModelEntry {
	entries, ok := modelClasses[class]
	if !ok {
		return nil
	}
	active := make([]ModelEntry, 0, len(entries))
	for _, entry := range entries {
		if _, err := r.GetConfig(entry.Provider); err != nil {
			continue
		}
		if !IsProviderEnabled(entry.Provider) {
			continue
		}
		active = append(active, entry)
	}
	return active
}
