package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	goai "github.com/zendev-sh/goai"
	goaiprovider "github.com/zendev-sh/goai/provider"
)

// goaiAdapter is implemented by tools that can produce a goai.Tool directly.
// ConcreteTool[In, Out] satisfies this interface via its GoaiTool method.
type goaiAdapter interface {
	GoaiTool() goai.Tool
}

// CLIInvokable is the interface implemented by tools that accept JSON input
// and return JSON output directly, bypassing the goai agent loop. The CLI
// tool command uses this to invoke a registered tool by name.
type CLIInvokable interface {
	Tool
	Invoke(ctx context.Context, raw json.RawMessage) (json.RawMessage, error)
}

// Registry holds all tools and provides methods to adapt them for different
// consumption contexts (goai agent loop, MCP server). Registry is safe for
// concurrent use.
type Registry struct {
	mu    sync.RWMutex
	tools []Tool
	model goaiprovider.LanguageModel
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Add registers a tool.
func (r *Registry) Add(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = append(r.tools, t)
}

// SetModel sets the LLM model used by agent tools.
func (r *Registry) SetModel(m goaiprovider.LanguageModel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.model = m
}

// Model returns the configured LLM model.
func (r *Registry) Model() goaiprovider.LanguageModel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.model
}

// GoaiTools returns all registered tools adapted as goai tools.
// Tools that do not implement the goaiAdapter interface are skipped.
func (r *Registry) GoaiTools() []goai.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]goai.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		if a, ok := t.(goaiAdapter); ok {
			result = append(result, a.GoaiTool())
		}
	}
	return result
}

// Names returns all registered tool names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, len(r.tools))
	for i, t := range r.tools {
		names[i] = t.Name()
	}
	return names
}

// Tools returns all registered tools as a copy of the internal slice.
// The returned slice is safe for the caller to iterate without holding
// the registry lock.
func (r *Registry) Tools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, len(r.tools))
	copy(result, r.tools)
	return result
}

// CLITools returns all registered tools that implement CLIInvokable.
// Tools that lack an Invoke method are skipped.
func (r *Registry) CLITools() []CLIInvokable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]CLIInvokable, 0, len(r.tools))
	for _, t := range r.tools {
		if ci, ok := t.(CLIInvokable); ok {
			result = append(result, ci)
		}
	}
	return result
}

// RegisterMCPTools registers all tools that implement mcpAdapter with the
// given MCP server. Tools that do not implement mcpAdapter are skipped.
func (r *Registry) RegisterMCPTools(server *mcpsdk.Server) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.tools {
		if a, ok := t.(mcpAdapter); ok {
			a.RegisterMCP(server)
		}
	}
}

// Lookup returns the tool registered under name. Returns an error when no
// tool matches.
func (r *Registry) Lookup(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.tools {
		if t.Name() == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("tool %q not registered", name)
}

// LookupCLI returns the CLIInvokable tool registered under name. Returns an
// error when no tool matches or the matching tool does not implement
// CLIInvokable.
func (r *Registry) LookupCLI(name string) (CLIInvokable, error) {
	t, err := r.Lookup(name)
	if err != nil {
		return nil, err
	}
	ci, ok := t.(CLIInvokable)
	if !ok {
		return nil, fmt.Errorf("tool %q is not CLI-invokable", name)
	}
	return ci, nil
}
