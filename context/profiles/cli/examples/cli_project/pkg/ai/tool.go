// Package ai provides the canonical tool abstraction and adapters for
// multiple consumption contexts: goai (LLM agent loop) and go-sdk (MCP server).
//
// Tools are defined once as ConcreteTool[In, Out], then adapted via
// ToGoaiToolTyped or RegisterMCPTool for their respective runtime contexts.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goai "github.com/zendev-sh/goai"
)

// ToolConfig carries optional per-tool rate limit overrides. When set on a
// ConcreteTool, these compose with the provider-level limits as
// min(tool, provider) for concurrency and RPM dimensions.
//
// Zero-value semantics: a zero field means "no override, use the provider
// default." Specifically:
//   - MaxConcurrent: 0 uses the provider's ConcurrentRequests limit.
//   - RPM: 0 uses the provider's RequestsPerMinute limiter.
//   - MaxWait: 0 uses the provider's MaxWaitSecs value.
//   - Priority: 0 uses default queue ordering (no preference).
type ToolConfig struct {
	MaxConcurrent int
	RPM           int
	MaxWait       time.Duration
	// Priority controls concurrency queue ordering. Lower values are served
	// first when multiple requests contend for a slot. Zero is the default.
	// Set on ConcreteTool and threaded through to GenerateRequest.Priority.
	Priority int
}

// Tool defines the metadata interface for all tools.
// Both ConcreteTool and external tool implementations satisfy this interface.
type Tool interface {
	Name() string
	Description() string
	InputSchema() json.RawMessage
}

// Handler is the canonical function signature for tool execution.
// The handler receives typed input via context and returns typed output.
// Both In and Out must be JSON-serializable.
type Handler[In, Out any] func(ctx context.Context, input In) (Out, error)

// ConcreteTool implements Tool and carries a typed execution handler.
// Define tools as ConcreteTool[In, Out], then adapt them to goai or MCP
// contexts using the adapter functions in this package.
type ConcreteTool[In, Out any] struct {
	name        string
	description string
	schema      json.RawMessage
	handler     Handler[In, Out]
	Config      ToolConfig
}

// NewConcreteTool creates a tool with the given metadata and handler.
// The input schema is generated automatically from In using goai.SchemaFrom.
func NewConcreteTool[In, Out any](name, description string, handler Handler[In, Out]) *ConcreteTool[In, Out] {
	return &ConcreteTool[In, Out]{
		name:        name,
		description: description,
		schema:      goai.SchemaFrom[In](),
		handler:     handler,
	}
}

// NewConcreteToolWithSchema creates a tool with an explicit input schema.
// Use this when automatic schema generation is insufficient or when the
// schema must differ from the Go type structure.
func NewConcreteToolWithSchema[In, Out any](name, description string, schema json.RawMessage, handler Handler[In, Out]) *ConcreteTool[In, Out] {
	return &ConcreteTool[In, Out]{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	}
}

// Name returns the tool identifier.
func (t *ConcreteTool[In, Out]) Name() string { return t.name }

// Description returns the tool description.
func (t *ConcreteTool[In, Out]) Description() string { return t.description }

// InputSchema returns the JSON Schema for the tool's input parameters.
func (t *ConcreteTool[In, Out]) InputSchema() json.RawMessage { return t.schema }

// Handler returns the tool's execution handler.
func (t *ConcreteTool[In, Out]) Handler() Handler[In, Out] { return t.handler }

// GoaiTool adapts this tool to a goai.Tool by delegating to the typed
// adapter. Registries and consumers should call this method rather than
// switching on concrete types.
func (t *ConcreteTool[In, Out]) GoaiTool() goai.Tool {
	return ToGoaiToolTyped(t)
}

// Invoke deserializes raw JSON input, calls the typed handler, and returns
// marshaled output. Suitable for CLI invocation where input and output are
// both JSON. An empty or "null" raw message unmarshals into the zero value
// of In, matching the goai adapter's behavior for no-argument tools.
func (t *ConcreteTool[In, Out]) Invoke(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var in In
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, fmt.Errorf("unmarshal input: %w", err)
		}
	}
	out, err := t.handler(ctx, in)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal output: %w", err)
	}
	return json.RawMessage(data), nil
}
