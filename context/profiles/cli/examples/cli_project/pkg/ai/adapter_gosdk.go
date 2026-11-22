package ai

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpAdapter is implemented by tools that can register themselves with an
// MCP server. Registry.RegisterMCPTools calls this interface on each
// registered tool that satisfies it.
type mcpAdapter interface {
	RegisterMCP(server *mcpsdk.Server)
}

// RegisterMCPTool registers a ConcreteTool[In, Out] with an MCP server
// using the go-sdk generic AddTool function.
//
// The adapter wraps the ConcreteTool's Handler into a ToolHandlerFor
// that ignores the CallToolRequest and returns nil for the CallToolResult,
// letting the go-sdk handle result marshaling and content population.
func RegisterMCPTool[In, Out any](server *mcpsdk.Server, tool *ConcreteTool[In, Out]) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        tool.Name(),
		Description: tool.Description(),
		InputSchema: tool.InputSchema(),
	}, mcpHandlerFrom(tool.Handler()))
}

// mcpHandlerFrom adapts a Handler[In, Out] to go-sdk's ToolHandlerFor signature.
// The CallToolRequest is forwarded but typically unused by simple tools.
// The CallToolResult is nil, letting go-sdk populate it from the output value.
func mcpHandlerFrom[In, Out any](h Handler[In, Out]) mcpsdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest, input In) (*mcpsdk.CallToolResult, Out, error) {
		out, err := h(ctx, input)
		if err != nil {
			return nil, *new(Out), err
		}
		return nil, out, nil
	}
}

// mcpTool wraps a ConcreteTool[In, Out] to implement the mcpAdapter interface.
// This enables Registry.RegisterMCPTools to register tools without knowing
// their concrete type parameters at the call site.
type mcpTool struct {
	Tool
	register func(server *mcpsdk.Server)
}

func (t *mcpTool) RegisterMCP(server *mcpsdk.Server) { t.register(server) }

// WrapForMCP wraps a ConcreteTool so it implements mcpAdapter.
// The returned tool delegates Name, Description, and InputSchema to the
// underlying ConcreteTool and calls RegisterMCPTool when registered with
// an MCP server.
func WrapForMCP[In, Out any](tool *ConcreteTool[In, Out]) Tool {
	return &mcpTool{
		Tool:     tool,
		register: func(s *mcpsdk.Server) { RegisterMCPTool(s, tool) },
	}
}
