package tools

import (
	"context"
	"fmt"

	goai "github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"

	"example.com/cliproj/pkg/ai"
)

// ResearchInput is the input for the research agent tool.
type ResearchInput struct {
	Query string `json:"query" jsonschema:"The research query to investigate"`
}

// ResearchOutput is the output from the research agent tool.
type ResearchOutput struct {
	Answer string `json:"answer"`
}

// NewResearchTool creates the research agent tool.
// This demonstrates the agent category: a tool that internally runs a
// multi-step agent loop with access to sub-tools. The model and sub-tools
// are provided at construction time since they come from runtime config.
func NewResearchTool(model provider.LanguageModel, subTools ...goai.Tool) *ai.ConcreteTool[ResearchInput, ResearchOutput] {
	return ai.NewConcreteTool(
		"research",
		"Researches a topic using an agent loop with access to sub-tools for multi-step reasoning.",
		func(ctx context.Context, input ResearchInput) (ResearchOutput, error) {
			if model == nil {
				return ResearchOutput{}, fmt.Errorf("research: model is required")
			}

			answer, err := ai.RunAgent(
				ctx, input.Query,
				ai.WithModel(model),
				ai.WithSystemPrompt("You are a research assistant. Use the available tools to investigate the query thoroughly."),
				ai.WithTools(subTools...),
				ai.WithMaxSteps(5),
			)
			if err != nil {
				return ResearchOutput{}, fmt.Errorf("research: %w", err)
			}

			return ResearchOutput{Answer: answer}, nil
		},
	)
}
