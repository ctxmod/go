package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	goaiprovider "github.com/zendev-sh/goai/provider"

	ai "example.com/cliproj/pkg/ai"
	"example.com/cliproj/pkg/ai/provider"
	"example.com/cliproj/pkg/ai/toolset"
	"example.com/cliproj/pkg/runtime"
	"example.com/cliproj/pkg/runtime/aisimple"
)

func resolveOptionalModel(rt aisimple.AISimpleRuntime) (goaiprovider.LanguageModel, error) {
	if rt.AIProvider() == "" {
		return nil, nil
	}
	reg := provider.NewDefaultRegistry(rt.AIAPIKey())
	model, _, err := provider.Resolve(reg, provider.ResolveRequest{
		Provider: rt.AIProvider(),
		Model:    rt.AIModel(),
		APIKey:   rt.AIAPIKey(),
	})
	return model, err
}

// Cmd groups the tool subcommands.
type Cmd struct {
	List   ListCmd   `cmd:"" help:"List registered tools"`
	Schema SchemaCmd `cmd:"" help:"Print the JSON schema for a tool"`
	Run    RunCmd    `cmd:"" help:"Invoke a tool with JSON input"`
}

// ListCmd enumerates registered tools and their descriptions.
type ListCmd struct{}

// SchemaCmd prints a tool's input schema.
type SchemaCmd struct {
	Name string `arg:"" help:"Tool name"`
}

// RunCmd invokes a tool by name with JSON input.
type RunCmd struct {
	Name   string `arg:"" help:"Tool name"`
	Input  string `help:"JSON input (defaults to stdin when omitted)" short:"i"`
	Format string `help:"Output format" enum:"json,text,table" default:"json" short:"f"`
}

func (c *ListCmd) Run(ctx context.Context) error {
	art := aisimple.MustGetAISimpleRuntime(ctx)

	reg, err := buildRegistry(art)
	if err != nil {
		return err
	}

	tools := reg.Tools()
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name() < tools[j].Name() })

	entries := make([]toolListEntry, 0, len(tools))
	for _, t := range tools {
		_, cliOK := t.(ai.CLIInvokable)
		entries = append(entries, toolListEntry{
			Name:        t.Name(),
			Description: t.Description(),
			CLI:         cliOK,
		})
	}

	art.Result(entries)
	art.Success("listed %d tools", len(entries))
	return nil
}

func (c *SchemaCmd) Run(ctx context.Context) error {
	art := aisimple.MustGetAISimpleRuntime(ctx)

	reg, err := buildRegistry(art)
	if err != nil {
		return err
	}

	t, err := reg.Lookup(c.Name)
	if err != nil {
		return runtime.Exit(runtime.ExitUsage, err)
	}

	schema := t.InputSchema()
	if len(schema) == 0 {
		art.Result(map[string]any{})
		return nil
	}
	var decoded any
	if err := json.Unmarshal(schema, &decoded); err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "schema: decode: %v", err)
	}
	art.Result(decoded)
	return nil
}

func (c *RunCmd) Run(ctx context.Context) error {
	art := aisimple.MustGetAISimpleRuntime(ctx)

	reg, err := buildRegistry(art)
	if err != nil {
		return err
	}

	tool, err := reg.LookupCLI(c.Name)
	if err != nil {
		return runtime.Exit(runtime.ExitUsage, err)
	}

	raw, err := resolveInput(c.Input, c.Name, os.Stdin)
	if err != nil {
		return runtime.Exit(runtime.ExitUsage, err)
	}

	out, err := tool.Invoke(ctx, raw)
	if err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "invoke %s: %v", c.Name, err)
	}

	return writeOutput(art.Runtime, out, outputFormat(c.Format))
}

type toolListEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CLI         bool   `json:"cli_invokable"`
}

func buildRegistry(rt aisimple.AISimpleRuntime) (*ai.Registry, error) {
	model, err := resolveOptionalModel(rt)
	if err != nil {
		return nil, err
	}
	return toolset.New(toolset.Config{Model: model}), nil
}

func resolveInput(flag, name string, stdin io.Reader) (json.RawMessage, error) {
	if flag != "" {
		if !json.Valid([]byte(flag)) {
			return nil, fmt.Errorf("--input: invalid JSON; run `cliproj tool schema %s` for the expected format", name)
		}
		return json.RawMessage(flag), nil
	}
	if isStdinTTY(stdin) {
		return nil, fmt.Errorf("--input is required when stdin is a terminal; use --input '<json>' or pipe input")
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		return json.RawMessage("{}"), nil
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("stdin: invalid JSON")
	}
	return json.RawMessage(data), nil
}

func isStdinTTY(r io.Reader) bool {
	type statter interface{ Stat() (os.FileInfo, error) }
	s, ok := r.(statter)
	if !ok {
		return false
	}
	fi, err := s.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
