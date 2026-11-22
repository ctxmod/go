// Package tools provides example tool implementations for the three
// tool categories: programmatic (direct execution), synthesizing
// (parallel sub-tool fan-out), and agent (LLM-driven multi-step loop).
package tools

import (
	"context"

	"example.com/cliproj/pkg/ai"
)

// EchoInput is the input for the echo tool.
type EchoInput struct {
	Message string `json:"message" jsonschema:"The message to echo back"`
}

// EchoOutput is the output from the echo tool.
type EchoOutput struct {
	Echo   string `json:"echo"`
	Length int    `json:"length"`
}

// NewEchoTool creates the echo programmatic tool.
// This demonstrates the programmatic category: no LLM, direct execution.
func NewEchoTool() *ai.ConcreteTool[EchoInput, EchoOutput] {
	return ai.NewConcreteTool(
		"echo",
		"Echoes the input message back with character count metadata.",
		func(_ context.Context, input EchoInput) (EchoOutput, error) {
			return EchoOutput{
				Echo:   input.Message,
				Length: len(input.Message),
			}, nil
		},
	)
}
