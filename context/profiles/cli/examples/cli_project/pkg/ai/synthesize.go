package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"

	goaiprovider "github.com/zendev-sh/goai/provider"
	"golang.org/x/sync/errgroup"
)

// InvokeResult holds the result of a single tool invocation.
type InvokeResult struct {
	ToolName string
	Output   string
	Error    error
}

// InvokeAll runs multiple tool handlers concurrently and collects results.
// Goroutine spawning is bounded by runtime.NumCPU() via errgroup. Each
// handler receives its own input from the inputs map. Errors are captured
// per-tool rather than failing the entire invocation. Panics within
// handlers are recovered and recorded as errors.
func InvokeAll[In, Out any](ctx context.Context, handlers map[string]Handler[In, Out], inputs map[string]In) []InvokeResult {
	results := make([]InvokeResult, 0, len(handlers))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	type invokeEntry struct {
		name string
		in   In
		h    Handler[In, Out]
	}

	var entries []invokeEntry
	for name, handler := range handlers {
		input, ok := inputs[name]
		if !ok {
			results = append(results, InvokeResult{
				ToolName: name,
				Error:    fmt.Errorf("no input provided for tool %q", name),
			})
			continue
		}
		entries = append(entries, invokeEntry{name: name, in: input, h: handler})
	}

	for _, e := range entries {
		g.Go(func() error {
			var out Out
			var err error

			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("tool %q panicked: %v", e.name, r)
					}
				}()
				out, err = e.h(gctx, e.in)
			}()

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results = append(results, InvokeResult{
					ToolName: e.name,
					Error:    err,
				})
				return nil
			}

			data, marshalErr := json.Marshal(out)
			if marshalErr != nil {
				results = append(results, InvokeResult{
					ToolName: e.name,
					Error:    fmt.Errorf("marshal output for %q: %w", e.name, marshalErr),
				})
				return nil
			}

			results = append(results, InvokeResult{
				ToolName: e.name,
				Output:   string(data),
			})
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		results = append(results, InvokeResult{
			ToolName: "invoke-all",
			Error:    err,
		})
	}
	return results
}

const synthesisDensityInstruction = `

Synthesis target: produce 200-400 words. Prioritize information density over completeness. Every sentence should carry distinct meaning -- no filler, no hedging, no restatement of the obvious.`

// BuildSynthesisPrompt constructs the full prompt for synthesis by combining
// a system prefix, user prompt, and tool invocation results. The returned
// string is suitable for use as user-message content in a Generate call.
func BuildSynthesisPrompt(system, prompt string, results []InvokeResult) string {
	var sb strings.Builder
	if system != "" {
		sb.WriteString(system)
		sb.WriteByte('\n')
	}
	sb.WriteString(prompt)
	sb.WriteString(synthesisDensityInstruction)
	sb.WriteString("\n\nTool results:\n")
	for _, r := range results {
		fmt.Fprintf(&sb, "\n## %s\n", r.ToolName)
		if r.Error != nil {
			fmt.Fprintf(&sb, "Error: %v\n", r.Error)
		} else {
			sb.WriteString(r.Output)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// RunSynthesis executes a single completion to synthesize multiple tool
// outputs into a coherent response. No tool use -- just text transformation
// with a single generation step. Routes through ai.Generate for rate
// limiting and retry.
//
// The req provides provider/model routing. req.Messages is ignored;
// RunSynthesis constructs the user message from prompt and results via
// BuildSynthesisPrompt. req.System is used as a prefix within the
// assembled message, not as a separate system field.
func RunSynthesis(ctx context.Context, deps *GenerateDeps, req GenerateRequest, prompt string, results []InvokeResult) (string, error) {
	if deps == nil || deps.Registry == nil {
		return "", fmt.Errorf("synthesize: deps with registry required")
	}

	text := BuildSynthesisPrompt(req.System, prompt, results)

	req.Messages = []goaiprovider.Message{{
		Role:    goaiprovider.RoleUser,
		Content: []goaiprovider.Part{{Type: goaiprovider.PartText, Text: text}},
	}}

	genResult, err := Generate(ctx, deps, req)
	if err != nil {
		return "", fmt.Errorf("synthesize: %w", err)
	}
	return genResult.Text, nil
}
