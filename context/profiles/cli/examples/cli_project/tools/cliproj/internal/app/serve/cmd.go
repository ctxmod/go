package serve

import (
	"context"
	"fmt"
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	ai "example.com/cliproj/pkg/ai"
	"example.com/cliproj/pkg/ai/provider"
	"example.com/cliproj/pkg/ai/retry"
	"example.com/cliproj/pkg/ai/toolset"
	"example.com/cliproj/pkg/runtime"
	"example.com/cliproj/pkg/runtime/aisimple"
)

// Cmd provides server subcommands.
type Cmd struct {
	MCP mcpCmd `cmd:"" help:"MCP server commands"`
}

type mcpCmd struct {
	Transport string `help:"Transport type" enum:"stdio,http" default:"stdio" env:"MCP_TRANSPORT"`
	Addr      string `help:"HTTP listen address (HTTP transport only)" default:":8080" env:"MCP_ADDR"`
}

func (c *mcpCmd) Run(ctx context.Context) error {
	art := aisimple.MustGetAISimpleRuntime(ctx)

	reg := provider.NewDefaultRegistry(art.AIAPIKey())
	model, providerName, err := provider.Resolve(reg, provider.ResolveRequest{
		Provider:   art.AIProvider(),
		Model:      art.AIModel(),
		ModelClass: art.AIModelClass(),
		APIKey:     art.AIAPIKey(),
	})
	if err != nil {
		return err
	}

	deps := ai.NewGenerateDeps(reg, nil, retry.SPEEDY)
	toolReg := toolset.New(toolset.Config{
		Model:    model,
		Deps:     deps,
		SynthReq: ai.GenerateRequest{ProviderName: providerName, ModelName: model.ModelID()},
	})

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "cliproj",
		Version: runtime.BuildVersion,
	}, nil)

	toolReg.RegisterMCPTools(server)

	switch c.Transport {
	case "stdio":
		art.Logger().Info("starting MCP server", "transport", "stdio", "tools", toolReg.Names())
		return server.Run(ctx, &mcpsdk.StdioTransport{})

	case "http":
		if c.Addr == "" {
			c.Addr = ":8080"
		}
		handler := mcpsdk.NewStreamableHTTPHandler(func(_ *http.Request) *mcpsdk.Server {
			return server
		}, nil)

		srv := &http.Server{Addr: c.Addr, Handler: handler}
		art.Logger().Info("starting MCP server", "transport", "http", "addr", c.Addr, "tools", toolReg.Names())

		go func() {
			<-ctx.Done()
			srv.Close()
		}()

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported transport: %s", c.Transport)
	}
}
