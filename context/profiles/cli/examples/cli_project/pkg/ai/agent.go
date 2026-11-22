package ai

import (
	"context"
	"fmt"
	"log/slog"

	goai "github.com/zendev-sh/goai"
	goaiprovider "github.com/zendev-sh/goai/provider"
)

// AgentConfig holds configuration for agent tool execution.
type AgentConfig struct {
	Model           goaiprovider.LanguageModel
	System          string
	MaxSteps        int
	Tools           []goai.Tool
	OutputThreshold int
	// GenerateDeps, when non-nil, wraps Model in a rateLimitedModel so that
	// every DoGenerate call inside the agent loop passes through the
	// rate-limit + retry pipeline.
	GenerateDeps *GenerateDeps
	// ProviderName and ModelName identify the provider/model for rate
	// limiting when GenerateDeps is set.
	ProviderName string
	ModelName    string
}

// AgentOption configures AgentConfig.
type AgentOption func(*AgentConfig)

// WithModel sets the language model for agent execution.
func WithModel(m goaiprovider.LanguageModel) AgentOption {
	return func(c *AgentConfig) { c.Model = m }
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(s string) AgentOption {
	return func(c *AgentConfig) { c.System = s }
}

// WithMaxSteps sets the maximum number of tool-loop iterations.
func WithMaxSteps(n int) AgentOption {
	return func(c *AgentConfig) { c.MaxSteps = n }
}

// WithTools sets the tools available to the agent.
func WithTools(tools ...goai.Tool) AgentOption {
	return func(c *AgentConfig) { c.Tools = tools }
}

// WithOutputThreshold sets the character threshold for sub-tool output
// truncation. Output exceeding this limit is truncated before being sent
// to the LLM. Zero means no limit.
func WithOutputThreshold(n int) AgentOption {
	return func(c *AgentConfig) { c.OutputThreshold = n }
}

// WithGenerateDeps enables rate limiting for the agent loop by wrapping the
// model in a rateLimitedModel that routes every DoGenerate call through
// ai.Generate's rate-limit + retry pipeline. The provider and model names
// identify the target for concurrency control and fallback routing.
func WithGenerateDeps(deps *GenerateDeps, provider, model string) AgentOption {
	return func(c *AgentConfig) {
		c.GenerateDeps = deps
		c.ProviderName = provider
		c.ModelName = model
	}
}

// defaultAgentConfig returns sensible defaults for agent configuration.
// The model must be set via WithModel before calling RunAgent.
func defaultAgentConfig(model goaiprovider.LanguageModel) AgentConfig {
	return AgentConfig{
		Model:    model,
		MaxSteps: 10,
		System:   "You are a helpful assistant.",
	}
}

// RunAgent executes a multi-turn agent loop with the given query.
// It returns the final text result from the LLM.
//
// When WithGenerateDeps is provided, the model is wrapped in a
// rateLimitedModel so every DoGenerate call inside the loop passes through
// the rate-limit + retry pipeline. Without it, the model is used directly.
func RunAgent(ctx context.Context, query string, opts ...AgentOption) (string, error) {
	cfg := defaultAgentConfig(nil)
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Model == nil {
		return "", fmt.Errorf("agent: model is required")
	}

	model := cfg.Model
	if cfg.GenerateDeps != nil {
		model = &rateLimitedModel{
			inner:     cfg.Model,
			deps:      cfg.GenerateDeps,
			provider:  cfg.ProviderName,
			modelName: cfg.ModelName,
		}
	}

	result, err := goai.GenerateText(
		ctx, model,
		goai.WithSystem(cfg.System),
		goai.WithPrompt(query),
		goai.WithTools(cfg.Tools...),
		goai.WithMaxSteps(cfg.MaxSteps),
		goai.WithOnAfterToolExecute(truncateOutput(cfg.OutputThreshold)),
		goai.WithOnToolCall(logToolCall),
	)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// rateLimitedModel wraps a goaiprovider.LanguageModel and routes DoGenerate
// calls through ai.Generate's rate-limit + retry pipeline. This ensures that
// agent loop iterations (which call DoGenerate multiple times) are protected
// by concurrency slots, throughput limiters, and retry logic.
type rateLimitedModel struct {
	inner     goaiprovider.LanguageModel
	deps      *GenerateDeps
	provider  string
	modelName string
}

func (m *rateLimitedModel) ModelID() string { return m.inner.ModelID() }

func (m *rateLimitedModel) DoStream(ctx context.Context, params goaiprovider.GenerateParams) (*goaiprovider.StreamResult, error) {
	return m.inner.DoStream(ctx, params)
}

func (m *rateLimitedModel) DoGenerate(ctx context.Context, params goaiprovider.GenerateParams) (*goaiprovider.GenerateResult, error) {
	cfg, err := m.deps.Registry.GetConfig(m.provider)
	if err != nil {
		return nil, err
	}
	result, err := generateForProvider(ctx, m.deps, GenerateRequest{
		ProviderName: m.provider,
		ModelName:    m.modelName,
		Messages:     params.Messages,
		System:       params.System,
		Tools:        params.Tools,
	}, cfg)
	if err != nil {
		return nil, err
	}
	return &goaiprovider.GenerateResult{
		Text:         result.Text,
		ToolCalls:    result.ToolCalls,
		FinishReason: result.FinishReason,
		Usage:        result.Usage,
	}, nil
}

// truncateOutput returns an OnAfterToolExecute hook that truncates sub-tool
// output exceeding the threshold. If threshold is zero, no truncation occurs.
func truncateOutput(threshold int) func(goai.AfterToolExecuteInfo) goai.AfterToolExecuteResult {
	return func(info goai.AfterToolExecuteInfo) goai.AfterToolExecuteResult {
		if threshold <= 0 || len(info.Output) <= threshold {
			return goai.AfterToolExecuteResult{}
		}
		return goai.AfterToolExecuteResult{
			Output: info.Output[:threshold] + "\n... (truncated)",
		}
	}
}

// logToolCall is an OnToolCall observability hook that logs each tool
// invocation at debug level.
func logToolCall(info goai.ToolCallInfo) {
	slog.Debug(
		"tool call",
		"tool", info.ToolName,
		"step", info.Step,
		"duration", info.Duration,
		"error", info.Error,
	)
}
