package provider

import (
	"fmt"
	"os"

	goaiprovider "github.com/zendev-sh/goai/provider"
	anthropic "github.com/zendev-sh/goai/provider/anthropic"
	google "github.com/zendev-sh/goai/provider/google"
)

// GoogleProviderConfig returns the default ProviderConfig for Google Gemini.
// APIKey is resolved lazily from the Auth.EnvVar at factory time.
func GoogleProviderConfig() ProviderConfig {
	return ProviderConfig{
		ID:        "google",
		Name:      "Google Gemini",
		APIFormat: APIFormatOpenAICompatible,
		Endpoint:  "https://generativelanguage.googleapis.com",
		Capabilities: []ProviderCapability{
			CapabilityCompletions,
			CapabilityStructuredOutput,
			CapabilityToolCalls,
			CapabilityStreaming,
		},
		RateLimits: RateLimitConfig{
			ConcurrentRequests: 5,
			RequestsPerMinute:  60,
			MaxWaitSecs:        30,
		},
		Auth: AuthConfig{
			Method: "api_key",
			EnvVar: "AI_API_KEY",
		},
		DefaultModel: "gemini-2.0-flash",
		Enabled:      true,
	}
}

// GoogleFactoryWithKey returns a factory that constructs the Google
// LanguageModel using a caller-supplied API key. Useful when the key is
// resolved from a source other than the process environment (e.g., the
// runtime configuration loaded via koanf).
func GoogleFactoryWithKey(apiKey string) ProviderFactory {
	return func(cfg ProviderConfig, model string) (goaiprovider.LanguageModel, error) {
		if model == "" {
			model = cfg.DefaultModel
		}
		if model == "" {
			return nil, fmt.Errorf("google: model is required")
		}
		opts := []google.Option{}
		if apiKey != "" {
			opts = append(opts, google.WithAPIKey(apiKey))
		}
		return google.Chat(model, opts...), nil
	}
}

// UnimplementedFactory returns a factory that fails with a descriptive
// error referencing providerName. Used for providers whose SDK integration
// is not yet wired into this project.
func UnimplementedFactory(providerName string) ProviderFactory {
	return func(cfg ProviderConfig, model string) (goaiprovider.LanguageModel, error) {
		return nil, fmt.Errorf("provider %q: not implemented", providerName)
	}
}

// AnthropicProviderConfig returns the default ProviderConfig for Anthropic.
func AnthropicProviderConfig() ProviderConfig {
	return ProviderConfig{
		ID:        "anthropic",
		Name:      "Anthropic",
		APIFormat: APIFormatAnthropicCompatible,
		Capabilities: []ProviderCapability{
			CapabilityCompletions,
			CapabilityStructuredOutput,
			CapabilityToolCalls,
			CapabilityStreaming,
		},
		RateLimits: RateLimitConfig{
			ConcurrentRequests: 20,
			MaxWaitSecs:        120,
		},
		DefaultModel: "claude-sonnet-4-6",
		Enabled:      true,
	}
}

// AnthropicFactory returns a factory that constructs an Anthropic
// LanguageModel routing all requests through llmsrv. The factory sets
// a placeholder API key so the goai SDK does not attempt to read
// ANTHROPIC_API_KEY from the environment — credentials are resolved
// by llmsrv, not by the caller. The base URL is read explicitly from
// ANTHROPIC_BASE_URL and passed via WithBaseURL so the SDK does not
// fall back to api.anthropic.com.
func AnthropicFactory() ProviderFactory {
	return func(cfg ProviderConfig, model string) (goaiprovider.LanguageModel, error) {
		if model == "" {
			model = cfg.DefaultModel
		}
		if model == "" {
			return nil, fmt.Errorf("anthropic: model is required")
		}
		baseURL := os.Getenv("ANTHROPIC_BASE_URL")
		if baseURL == "" {
			return nil, fmt.Errorf("anthropic: ANTHROPIC_BASE_URL is required")
		}
		return anthropic.Chat(model, anthropic.WithAPIKey("llmsrv"), anthropic.WithBaseURL(baseURL)), nil
	}
}

// GLM5ProviderConfig returns the default ProviderConfig for GLM-5 (z.ai).
func GLM5ProviderConfig() ProviderConfig {
	return ProviderConfig{
		ID:        "glm5",
		Name:      "GLM-5 (z.ai)",
		APIFormat: APIFormatAnthropicCompatible,
		Capabilities: []ProviderCapability{
			CapabilityCompletions,
			CapabilityStructuredOutput,
			CapabilityStreaming,
		},
		RateLimits: RateLimitConfig{
			ConcurrentRequests: 5,
			RequestsPerMinute:  60,
			MaxWaitSecs:        30,
		},
		Auth: AuthConfig{
			Method: "api_key",
			EnvVar: "GLM_API_KEY",
		},
		Enabled: true,
	}
}

// GLM47ProviderConfig returns the default ProviderConfig for GLM-4.7.
func GLM47ProviderConfig() ProviderConfig {
	return ProviderConfig{
		ID:        "glm47",
		Name:      "GLM-4.7 (z.ai)",
		APIFormat: APIFormatAnthropicCompatible,
		Capabilities: []ProviderCapability{
			CapabilityCompletions,
			CapabilityStructuredOutput,
			CapabilityStreaming,
		},
		Auth: AuthConfig{
			Method: "api_key",
			EnvVar: "GLM_API_KEY",
		},
		Enabled: true,
	}
}
