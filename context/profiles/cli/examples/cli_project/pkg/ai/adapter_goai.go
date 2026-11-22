package ai

import (
	"context"
	"encoding/json"
	"fmt"

	goai "github.com/zendev-sh/goai"
)

// ToGoaiToolTyped adapts a ConcreteTool[In, Out] to goai's Tool type.
// Input and output are JSON-marshaled to bridge the typed Go handler
// with goai's Execute signature (json.RawMessage -> string).
func ToGoaiToolTyped[In, Out any](ct *ConcreteTool[In, Out]) goai.Tool {
	return goai.Tool{
		Name:        ct.Name(),
		Description: ct.Description(),
		InputSchema: ct.InputSchema(),
		Execute:     adaptTypedHandler(ct.Handler()),
	}
}

// adaptTypedHandler wraps a Handler[In, Out] into goai's Execute signature.
func adaptTypedHandler[In, Out any](h Handler[In, Out]) func(context.Context, json.RawMessage) (string, error) {
	return func(ctx context.Context, input json.RawMessage) (string, error) {
		var in In
		if len(input) > 0 && string(input) != "null" {
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("unmarshal input: %w", err)
			}
		}

		out, err := h(ctx, in)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(out)
		if err != nil {
			return "", fmt.Errorf("marshal output: %w", err)
		}
		return string(data), nil
	}
}
