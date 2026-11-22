package tools

import (
	"context"
	"fmt"
	"strings"

	"example.com/cliproj/pkg/ai"
)

// AnalyzeInput is the input for the analyze tool.
type AnalyzeInput struct {
	Text string `json:"text" jsonschema:"The text to analyze"`
}

// AnalyzeOutput is the output from the analyze tool.
type AnalyzeOutput struct {
	Summary string `json:"summary"`
}

// countInput is the shared input type for all sub-tools in the analyze tool.
type countInput struct {
	Text string `json:"text"`
}

// countOutput is the shared output type for all sub-tools in the analyze tool.
type countOutput struct {
	Count int `json:"count"`
}

func charCount(_ context.Context, in countInput) (countOutput, error) {
	return countOutput{Count: len(in.Text)}, nil
}

func wordCount(_ context.Context, in countInput) (countOutput, error) {
	return countOutput{Count: len(strings.Fields(in.Text))}, nil
}

func lineCount(_ context.Context, in countInput) (countOutput, error) {
	return countOutput{Count: len(strings.Split(in.Text, "\n"))}, nil
}

const analyzePrompt = "Synthesize these text analysis results into a coherent summary. Report character, word, and line counts with brief commentary on proportions and density."

// NewAnalyzeTool creates the analyze synthesizing tool.
// This demonstrates the synthesizing category: fan-out to parallel sub-tools
// via InvokeAll, then synthesize the results into natural language via
// RunSynthesis.
func NewAnalyzeTool(deps *ai.GenerateDeps, req ai.GenerateRequest) *ai.ConcreteTool[AnalyzeInput, AnalyzeOutput] {
	return ai.NewConcreteTool(
		"analyze",
		"Analyzes text by computing character, word, and line counts in parallel, then synthesizes the results.",
		func(ctx context.Context, input AnalyzeInput) (AnalyzeOutput, error) {
			in := countInput(input)

			handlers := map[string]ai.Handler[countInput, countOutput]{
				"char_count": charCount,
				"word_count": wordCount,
				"line_count": lineCount,
			}

			inputs := map[string]countInput{
				"char_count": in,
				"word_count": in,
				"line_count": in,
			}

			results := ai.InvokeAll(ctx, handlers, inputs)

			var errs int
			for _, r := range results {
				if r.Error != nil {
					errs++
				}
			}

			if errs == len(results) {
				return AnalyzeOutput{}, fmt.Errorf("analyze: all sub-tools failed")
			}

			// When no LLM deps are available (e.g. metadata-only commands),
			// fall back to raw JSON concatenation.
			if deps == nil {
				var sb strings.Builder
				for _, r := range results {
					if r.Error != nil {
						continue
					}
					fmt.Fprintf(&sb, "%s: %s\n", r.ToolName, r.Output)
				}
				return AnalyzeOutput{Summary: sb.String()}, nil
			}

			summary, err := ai.RunSynthesis(ctx, deps, req, analyzePrompt, results)
			if err != nil {
				return AnalyzeOutput{}, fmt.Errorf("analyze: synthesis failed: %w", err)
			}

			return AnalyzeOutput{Summary: summary}, nil
		},
	)
}
