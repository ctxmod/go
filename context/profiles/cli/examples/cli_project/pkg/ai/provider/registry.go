package provider

import (
	"fmt"
	"sort"
	"sync"

	goaiprovider "github.com/zendev-sh/goai/provider"
)

// ProviderFactory constructs a LanguageModel for a given provider config and
// model identifier. Factories are invoked lazily on first use and the result
// is cached by the registry.
type ProviderFactory func(cfg ProviderConfig, model string) (goaiprovider.LanguageModel, error)

// providerEntry holds a registered provider's configuration alongside its
// factory.
type providerEntry struct {
	config  ProviderConfig
	factory ProviderFactory
}

// ProviderRegistry maps provider identifiers to their configuration and
// factory, and caches LanguageModel instances keyed by "provider:model".
type ProviderRegistry struct {
	mu      sync.RWMutex
	entries map[string]*providerEntry
	cache   map[string]goaiprovider.LanguageModel
}

// NewProviderRegistry returns an empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		entries: make(map[string]*providerEntry),
		cache:   make(map[string]goaiprovider.LanguageModel),
	}
}

// NewDefaultRegistry returns a registry pre-loaded with the four built-in
// providers (google, anthropic, glm5, glm47). The Google factory uses the
// supplied apiKey instead of reading from the environment.
func NewDefaultRegistry(apiKey string) *ProviderRegistry {
	reg := NewProviderRegistry()
	reg.Register("google", GoogleProviderConfig(), GoogleFactoryWithKey(apiKey))
	reg.Register("anthropic", AnthropicProviderConfig(), AnthropicFactory())
	reg.Register("glm5", GLM5ProviderConfig(), UnimplementedFactory("glm5"))
	reg.Register("glm47", GLM47ProviderConfig(), UnimplementedFactory("glm47"))
	return reg
}

// Register adds a provider to the registry. Re-registering a provider with
// the same name replaces the previous entry and invalidates any cached
// LanguageModel instances for that provider.
func (r *ProviderRegistry) Register(name string, cfg ProviderConfig, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[name] = &providerEntry{config: cfg, factory: factory}
	for key := range r.cache {
		if prefix := name + ":"; len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.cache, key)
		}
	}
}

// GetConfig returns the configuration registered for providerName.
func (r *ProviderRegistry) GetConfig(providerName string) (ProviderConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[providerName]
	if !ok {
		return ProviderConfig{}, fmt.Errorf("provider %q not registered", providerName)
	}
	return entry.config, nil
}

// GetModel returns the LanguageModel for the given provider and model,
// constructing it via the registered factory on first access and caching the
// instance for subsequent calls. Returns an error when the provider is
// unknown or the factory fails.
func (r *ProviderRegistry) GetModel(providerName, modelName string) (goaiprovider.LanguageModel, error) {
	key := providerName + ":" + modelName

	r.mu.RLock()
	if model, ok := r.cache[key]; ok {
		r.mu.RUnlock()
		return model, nil
	}
	entry, ok := r.entries[providerName]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", providerName)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if model, ok := r.cache[key]; ok {
		return model, nil
	}
	model, err := entry.factory(entry.config, modelName)
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", providerName, err)
	}
	if model == nil {
		return nil, fmt.Errorf("provider %q: factory returned nil model", providerName)
	}
	r.cache[key] = model
	return model, nil
}

// Providers returns the sorted list of registered provider identifiers.
func (r *ProviderRegistry) Providers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.entries))
	for name := range r.entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
