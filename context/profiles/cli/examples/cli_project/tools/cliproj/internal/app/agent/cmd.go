package agent

import (
	"context"
	"fmt"

	ai "example.com/cliproj/pkg/ai"
	"example.com/cliproj/pkg/ai/provider"
	"example.com/cliproj/pkg/ai/retry"
	"example.com/cliproj/pkg/ai/toolset"
	"example.com/cliproj/pkg/runtime/aisimple"
)

// Cmd provides AI agent commands.
type Cmd struct {
	Query      string `arg:"" help:"Query to send to the agent"`
	Provider   string `help:"LLM provider name" default:"" env:"AI_PROVIDER"`
	Model      string `help:"LLM model" default:"" env:"AI_MODEL"`
	ModelClass string `help:"Model class (genius|pro|flash|nano)" default:"" env:"AI_MODEL_CLASS"`
	MaxSteps   int    `help:"Maximum agent loop steps" default:"8" env:"AI_MAX_STEPS"`
}

func (c *Cmd) Run(ctx context.Context) error {
	art := aisimple.MustGetAISimpleRuntime(ctx)

	if c.Query == "" {
		return fmt.Errorf("query is required")
	}

	maxSteps := c.MaxSteps
	if maxSteps == 0 {
		maxSteps = art.AIMaxSteps()
	}
	if maxSteps == 0 {
		maxSteps = 8
	}

	pick := func(flag, rtVal string) string {
		if flag != "" {
			return flag
		}
		return rtVal
	}

	reg := provider.NewDefaultRegistry(art.AIAPIKey())
	model, providerName, err := provider.Resolve(reg, provider.ResolveRequest{
		Provider:   pick(c.Provider, art.AIProvider()),
		Model:      pick(c.Model, art.AIModel()),
		ModelClass: pick(c.ModelClass, art.AIModelClass()),
		APIKey:     art.AIAPIKey(),
	})
	if err != nil {
		return err
	}

	deps := ai.NewGenerateDeps(reg, nil, retry.SPEEDY)
	toolReg := toolset.New(toolset.Config{
		Model:    model,
		Deps:     deps,
		SynthReq: ai.GenerateRequest{ProviderName: providerName, ModelName: model.ModelID()},
	})

	goaiTools := toolReg.GoaiTools()

	art.Logger().Info(
		"agent invocation",
		"query", c.Query,
		"provider", providerName,
		"model", model.ModelID(),
		"model_class", pick(c.ModelClass, art.AIModelClass()),
		"max_steps", maxSteps,
		"tools", toolReg.Names(),
	)

	answer, err := ai.RunAgent(
		ctx, c.Query,
		ai.WithModel(model),
		ai.WithSystemPrompt("You are a helpful assistant with access to tools for analysis and research."),
		ai.WithTools(goaiTools...),
		ai.WithMaxSteps(maxSteps),
	)
	if err != nil {
		return fmt.Errorf("agent: %w", err)
	}

	art.Success("%s", answer)
	return nil
}
