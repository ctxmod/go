// Package provider contains provider registry types and factories.
package provider

// APIFormat enumerates supported API endpoint formats.
type APIFormat string

const (
	APIFormatOpenAICompatible    APIFormat = "openai_compatible"
	APIFormatAnthropicCompatible APIFormat = "anthropic_compatible"
)

// ProviderCapability declares what an LLM provider supports.
type ProviderCapability string

const (
	CapabilityCompletions      ProviderCapability = "completions"
	CapabilityEmbeddings       ProviderCapability = "embeddings"
	CapabilityStructuredOutput ProviderCapability = "structured_output"
	CapabilityToolCalls        ProviderCapability = "tool_calls"
	CapabilityStreaming        ProviderCapability = "streaming"
)

// ModelClass defines the quality tier used for class-based model selection.
type ModelClass string

const (
	ClassGenius ModelClass = "genius"
	ClassPro    ModelClass = "pro"
	ClassFlash  ModelClass = "flash"
	ClassNano   ModelClass = "nano"
)

// FallbackCondition specifies when a fallback provider activates.
type FallbackCondition string

const (
	FallbackRateLimited FallbackCondition = "rate_limited"
	FallbackUnavailable FallbackCondition = "unavailable"
	FallbackError       FallbackCondition = "error"
)

// AuthConfig defines authentication for an LLM provider.
type AuthConfig struct {
	// Method is one of "api_key", "bearer_token", or "pass_through_oauth".
	Method       string
	HeaderName   string
	HeaderPrefix string
	EnvVar       string
}

// RateLimitConfig defines per-provider rate limit dimensions.
type RateLimitConfig struct {
	ConcurrentRequests int
	RequestsPerMinute  int
	MaxWaitSecs        int
}

// FallbackRule defines a single fallback entry in a provider's chain.
type FallbackRule struct {
	Condition      FallbackCondition
	TargetProvider string
	TargetModel    string
	Priority       int
}

// ProviderConfig defines an LLM provider's connection and capability details.
type ProviderConfig struct {
	ID           string
	Name         string
	APIFormat    APIFormat
	Endpoint     string
	Capabilities []ProviderCapability
	RateLimits   RateLimitConfig
	Auth         AuthConfig
	Fallbacks    []FallbackRule
	DefaultModel string
	Models       []string
	Enabled      bool
}
