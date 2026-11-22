package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	goaiprovider "github.com/zendev-sh/goai/provider"

	"example.com/cliproj/pkg/ai/provider"
	"example.com/cliproj/pkg/ai/retry"
)

// GenerateRequest is the input to Generate. It mirrors goaiprovider.GenerateParams
// enough for the common agent-loop path while adding provider/model routing,
// speedy-mode opt-in, priority-based concurrency ordering, and tool-scoped
// rate limit keys.
type GenerateRequest struct {
	ProviderName string
	ModelName    string
	Messages     []goaiprovider.Message
	System       string
	Tools        []goaiprovider.ToolDefinition
	MaxSteps     int
	Speedy       bool
	// Priority controls concurrency queue ordering. Lower values are served
	// first when multiple requests contend for a slot. Zero is the default.
	Priority int
	// ToolName, when non-empty, scopes the concurrency pool key to
	// "provider:model:toolname" instead of "provider:model".
	ToolName string
	// ToolRateLimitOverrides, when non-zero, compose with the provider-level
	// limits via min(tool, provider) for concurrency and RPM dimensions.
	ToolRateLimitOverrides ToolConfig
}

// GenerateResult returns the model's response plus enough metadata for
// downstream agent-loop steps to reason about continuation and cost.
type GenerateResult struct {
	Text         string
	ToolCalls    []goaiprovider.ToolCall
	FinishReason goaiprovider.FinishReason
	Usage        goaiprovider.Usage
}

// keepaliveInterval is the period between OnWait callbacks while a request
// is blocked waiting for a concurrency slot.
const keepaliveInterval = 5 * time.Second

// SlotLimiter reserves concurrency slots per pool key. Generate is its only
// consumer, so the interface lives here and stays as narrow as the calls
// Generate makes; any priority-queue limiter that satisfies it plugs in.
// A nil SlotLimiter disables slot reservation.
type SlotLimiter interface {
	// SetBackendCapacity registers (or updates) the concurrent-request
	// capacity of one pool key.
	SetBackendCapacity(key string, capacity int)
	// Acquire blocks until a slot of key is free, the deadline passes, or
	// ctx is done. Higher priority values are served first. The returned
	// release function returns the slot and must be called exactly once.
	Acquire(ctx context.Context, key string, priority int64, deadline time.Time) (release func(), err error)
}

// GenerateDeps wires the rate-limit and retry infrastructure behind
// Generate. Consumers build one GenerateDeps per process and reuse it for
// every call; the internal limiter and registry guard all shared state.
// Use NewGenerateDeps to construct instances.
type GenerateDeps struct {
	Registry *provider.ProviderRegistry
	Limiter  SlotLimiter
	// RetryConfig applied to provider calls. Zero value is treated as
	// retry.DEFAULT.
	RetryConfig retry.RetryConfig
	// OnWait is called periodically while a request is waiting for a
	// concurrency slot. The message parameter describes the wait state.
	// When nil, no keepalive callback is invoked.
	OnWait func(msg string)
}

// NewGenerateDeps constructs a GenerateDeps with the required fields.
// Callers may set OnWait on the returned value when keepalive callbacks
// are needed.
func NewGenerateDeps(registry *provider.ProviderRegistry, lim SlotLimiter, retryConfig retry.RetryConfig) *GenerateDeps {
	return &GenerateDeps{
		Registry:    registry,
		Limiter:     lim,
		RetryConfig: retryConfig,
	}
}

// Generate routes a request through the rate-limit + retry pipeline:
//  1. Resolve the model and provider config from the registry.
//  2. Acquire a concurrency slot (combined concurrency + rate limiting).
//  3. Execute DoGenerate inside the retry loop.
//  4. On provider error (not context cancellation), walk fallback providers.
func Generate(ctx context.Context, deps *GenerateDeps, req GenerateRequest) (GenerateResult, error) {
	if deps == nil || deps.Registry == nil {
		return GenerateResult{}, fmt.Errorf("generate: deps with registry required")
	}
	if req.ProviderName == "" || req.ModelName == "" {
		return GenerateResult{}, errors.New("generate: provider and model required")
	}

	cfg, err := deps.Registry.GetConfig(req.ProviderName)
	if err != nil {
		return GenerateResult{}, err
	}

	result, err := generateForProvider(ctx, deps, req, cfg)
	if err == nil {
		return result, nil
	}

	// Walk fallbacks on any non-cancellation error. Provider 5xx, network
	// errors, persistent 429 after retry exhaustion all trigger fallback.
	// Context cancellation and deadline exceeded propagate immediately.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return result, err
	}

	chain := buildFallbackChain(cfg, req.ModelName)
	if len(chain) == 0 {
		return result, err
	}
	slog.Warn(
		"primary provider failed, walking fallback chain",
		"primary", req.ProviderName,
		"primary_model", req.ModelName,
		"error", err,
	)
	return retry.TryEach(ctx, chain, func(ctx context.Context, fallback fallbackTarget) (GenerateResult, error) {
		slog.Warn(
			"primary provider unavailable, trying fallback",
			"primary", req.ProviderName,
			"primary_model", req.ModelName,
			"fallback_provider", fallback.Provider,
			"fallback_model", fallback.Model,
			"error", err,
		)
		fallbackCfg, cfgErr := deps.Registry.GetConfig(fallback.Provider)
		if cfgErr != nil {
			return GenerateResult{}, cfgErr
		}
		fallbackReq := req
		fallbackReq.ProviderName = fallback.Provider
		fallbackReq.ModelName = fallback.Model
		return generateForProvider(ctx, deps, fallbackReq, fallbackCfg)
	})
}

// generateForProvider runs the slot+limiter+retry pipeline for a single
// provider. The helper is shared between the primary path and the fallback
// walker so retry/rate-limit semantics stay identical across tiers.
func generateForProvider(
	ctx context.Context,
	deps *GenerateDeps,
	req GenerateRequest,
	cfg provider.ProviderConfig,
) (GenerateResult, error) {
	model, err := deps.Registry.GetModel(req.ProviderName, req.ModelName)
	if err != nil {
		return GenerateResult{}, err
	}

	release, err := deps.acquireSlot(ctx, req, cfg)
	if err != nil {
		return GenerateResult{}, err
	}
	defer release()

	config := deps.RetryConfig
	if (config == retry.RetryConfig{}) {
		config = retry.DEFAULT
	}
	if req.Speedy {
		config = retry.SPEEDY
	}

	params := goaiprovider.GenerateParams{
		Messages: req.Messages,
		System:   req.System,
		Tools:    req.Tools,
	}

	outcome := retry.Retry(ctx, config, func(ctx context.Context) (*goaiprovider.GenerateResult, error) {
		return model.DoGenerate(ctx, params)
	})
	if outcome.Err != nil {
		return GenerateResult{}, outcome.Err
	}
	if outcome.Value == nil {
		return GenerateResult{}, errors.New("generate: nil result from provider")
	}
	return GenerateResult{
		Text:         outcome.Value.Text,
		ToolCalls:    outcome.Value.ToolCalls,
		FinishReason: outcome.Value.FinishReason,
		Usage:        outcome.Value.Usage,
	}, nil
}

// acquireSlot reserves a concurrency slot from the limiter, running an
// OnWait keepalive goroutine while the caller blocks on the priority
// queue. The returned release function returns the slot when called and
// must be invoked once the request completes.
func (deps *GenerateDeps) acquireSlot(
	ctx context.Context,
	req GenerateRequest,
	cfg provider.ProviderConfig,
) (release func(), err error) {
	if deps.Limiter == nil {
		return func() {}, nil
	}

	// Build pool key: "provider:model" normally, or "provider:model:toolname"
	// when ToolName is set.
	poolKey := req.ProviderName + ":" + req.ModelName
	if req.ToolName != "" {
		poolKey = poolKey + ":" + req.ToolName
	}

	capacity := cfg.RateLimits.ConcurrentRequests
	if capacity <= 0 {
		capacity = 1
	}
	if req.ToolRateLimitOverrides.MaxConcurrent > 0 && req.ToolRateLimitOverrides.MaxConcurrent < capacity {
		capacity = req.ToolRateLimitOverrides.MaxConcurrent
	}

	// Ensure the backend capacity is registered for this model key.
	deps.Limiter.SetBackendCapacity(poolKey, capacity)

	deadline, _ := ctx.Deadline()

	// Acquire with keepalive: launch a goroutine that calls OnWait every 5s
	// while the caller blocks on the priority queue.
	acquired := make(chan struct{})
	var (
		releaseFn  func()
		acquireErr error
	)
	go func() {
		defer close(acquired)
		releaseFn, acquireErr = deps.Limiter.Acquire(ctx, poolKey, int64(req.Priority), deadline)
	}()

	if deps.OnWait != nil {
		go func() {
			ticker := time.NewTicker(keepaliveInterval)
			defer ticker.Stop()
			for {
				select {
				case <-acquired:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					deps.OnWait(fmt.Sprintf("waiting for concurrency slot on %s", poolKey))
				}
			}
		}()
	}

	<-acquired
	if acquireErr != nil {
		return nil, fmt.Errorf("acquire slot for %s: %w", poolKey, acquireErr)
	}
	return releaseFn, nil
}

// fallbackTarget is the internal shape TryEach iterates.
type fallbackTarget struct {
	Provider string
	Model    string
}

// buildFallbackChain translates the provider's declared fallback rules
// into TryEach targets. Rules with missing target fields fall back to
// requestedModel when no target model is specified.
func buildFallbackChain(cfg provider.ProviderConfig, requestedModel string) []fallbackTarget {
	chain := make([]fallbackTarget, 0, len(cfg.Fallbacks))
	for _, rule := range cfg.Fallbacks {
		if rule.TargetProvider == "" {
			continue
		}
		model := rule.TargetModel
		if model == "" {
			model = requestedModel
		}
		chain = append(chain, fallbackTarget{Provider: rule.TargetProvider, Model: model})
	}
	return chain
}
