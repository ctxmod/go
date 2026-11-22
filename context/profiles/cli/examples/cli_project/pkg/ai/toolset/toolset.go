// Package toolset assembles the built-in tool registry used by the agent
// and CLI tool commands. Keeping the assembly in a separate package avoids
// an import cycle between the ai package and its tools subpackage.
package toolset

import (
	goaiprovider "github.com/zendev-sh/goai/provider"

	"example.com/cliproj/pkg/ai"
	"example.com/cliproj/pkg/ai/tools"
)

// Config holds the dependencies needed to construct tools that require LLM
// access (analyze for synthesis, research for agent loops).
type Config struct {
	// Model is the language model used by the research tool's agent loop.
	Model goaiprovider.LanguageModel
	// Deps provides rate-limit and retry infrastructure for the analyze tool's
	// synthesis step. When nil, the analyze tool returns raw JSON instead of
	// synthesized text.
	Deps *ai.GenerateDeps
	// SynthReq routes the analyze tool's synthesis call through a specific
	// provider/model. Ignored when Deps is nil.
	SynthReq ai.GenerateRequest
}

// New returns a tool registry populated with the built-in tool set
// (echo, analyze, research) configured for the provided language model.
// Both the agent command and the serve command call this function so
// they operate against the same tool set.
//
// All tools are wrapped with ai.WrapForMCP so that
// Registry.RegisterMCPTools can register them with an MCP server without
// knowing their concrete type parameters.
//
// The research tool receives the currently-registered goai tool list at
// construction time so its internal agent loop can delegate to echo and
// analyze during multi-step reasoning. Tools added to the returned registry
// after this call will not retroactively become visible to research.
func New(cfg Config) *ai.Registry {
	reg := ai.NewRegistry()
	reg.SetModel(cfg.Model)

	reg.Add(ai.WrapForMCP[tools.EchoInput, tools.EchoOutput](tools.NewEchoTool()))
	reg.Add(ai.WrapForMCP[tools.AnalyzeInput, tools.AnalyzeOutput](tools.NewAnalyzeTool(cfg.Deps, cfg.SynthReq)))

	research := tools.NewResearchTool(cfg.Model, reg.GoaiTools()...)
	reg.Add(ai.WrapForMCP[tools.ResearchInput, tools.ResearchOutput](research))

	return reg
}
