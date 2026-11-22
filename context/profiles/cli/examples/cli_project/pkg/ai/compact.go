package ai

import (
	"context"
	"fmt"

	goaiprovider "github.com/zendev-sh/goai/provider"
)

// Compact summarizes or compresses text output using a single completion
// call. No tool use -- just text transformation. Routes through
// ai.Generate for rate limiting and retry.
//
// The req provides provider/model routing. req.Messages is ignored; Compact
// constructs the user message from the text parameter. req.System defaults
// to a summarization prompt when empty.
func Compact(ctx context.Context, deps *GenerateDeps, req GenerateRequest, text string) (string, error) {
	if deps == nil || deps.Registry == nil {
		return "", fmt.Errorf("compact: deps with registry required")
	}

	system := req.System
	if system == "" {
		system = "Summarize the following output concisely, preserving key details and structure. Aim for brevity."
	}

	req.Messages = []goaiprovider.Message{{
		Role:    goaiprovider.RoleUser,
		Content: []goaiprovider.Part{{Type: goaiprovider.PartText, Text: text}},
	}}
	req.System = system

	result, err := Generate(ctx, deps, req)
	if err != nil {
		return "", fmt.Errorf("compact: %w", err)
	}
	return result.Text, nil
}
