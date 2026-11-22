package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"

	"example.com/cliproj/pkg/config"
	"example.com/cliproj/pkg/output"
	"example.com/cliproj/pkg/runtime"
	"example.com/cliproj/tools/cliproj/internal/app/agent"
	"example.com/cliproj/tools/cliproj/internal/app/demo"
	"example.com/cliproj/tools/cliproj/internal/app/deploy"
	"example.com/cliproj/tools/cliproj/internal/app/deploy/service"
	"example.com/cliproj/tools/cliproj/internal/app/list"
	"example.com/cliproj/tools/cliproj/internal/app/serve"
	"example.com/cliproj/tools/cliproj/internal/app/tool"
	"example.com/cliproj/tools/cliproj/internal/app/version"
)

const appName = "cliproj"

// Root defines the top-level CLI grammar. Global flags live here;
// subcommands are embedded structs with Run() methods defined in their
// own packages.
//
// Config-loading flags (ConfigFiles, EnvFile, Profile) feed the config
// loader rather than binding to application state. Every other global flag
// carries a `koanf` tag naming its config key: when the user sets it, the
// value overrides that key with the highest precedence; when they do not,
// the config layers decide. That is why none of them has a Kong default —
// a default would fill the field unconditionally and erase "unset".
//
// Field names are chosen so Kong's derived names are the intended argv
// spelling (EnvFile derives --env-file); name tags carry no dashes.
type Root struct {
	// Config loading flags.
	ConfigFiles []string `name:"config" help:"Config files to load in order." type:"path" sep:","`
	EnvFile     []string `help:"Dotenv files to load in order." type:"path" sep:","`
	Profile     string   `help:"Named config profile to activate." env:"CLIPROJ_PROFILE"`

	// Debug workspace flags.
	Debug    bool   `help:"Enable debug run artifacts." koanf:"debug.enabled"`
	DebugDir string `help:"Directory for debug runs." type:"path" koanf:"debug.dir"`

	// Session and artifact directory flags.
	Session   string `help:"Session identifier for artifact directory grouping." koanf:"session"`
	OutputDir string `help:"User artifacts directory (overrides the XDG default)." type:"path" koanf:"paths.output"`
	CacheDir  string `help:"Retained and cross-session cache directory (overrides the XDG default)." type:"path" koanf:"paths.cache"`
	TempDir   string `help:"Ephemeral temp directory (overrides the OS default)." type:"path" koanf:"paths.temp"`

	// Logging and output flags. The output enum mirrors output.ValidModes.
	LogLevel string `help:"Log level: debug, info, warn, or error." enum:"debug,info,warn,error," default:"" koanf:"log.level"`
	Output   string `help:"Output format: text, json, ndjson, yamlish, quiet, or terminal." enum:"text,json,ndjson,yamlish,quiet,terminal," default:"" koanf:"output.mode"`
	Verbose  int    `help:"Increase stderr verbosity." short:"v" type:"counter" koanf:"verbose"`

	// Subcommands.
	Agent   agent.Cmd   `cmd:"" help:"Ask an AI agent with tools."`
	Demo    demo.Cmd    `cmd:"" help:"Run a simulated multi-phase operation."`
	Deploy  deploy.Cmd  `cmd:"" help:"Deployment commands."`
	List    list.Cmd    `cmd:"" help:"List available items."`
	Serve   serve.Cmd   `cmd:"" help:"Server commands."`
	Tool    tool.Cmd    `cmd:"" help:"Invoke registered AI tools directly."`
	Version version.Cmd `cmd:"" help:"Print version information."`
}

// Run is the application entry point. It parses CLI arguments, loads
// configuration, constructs the runtime, and executes the selected
// command. The return value becomes the process exit code.
func Run(args []string) int {
	var cli Root
	parser, err := kong.New(
		&cli,
		kong.Name(appName),
		kong.Description("Example CLI application."),
		kong.Writers(os.Stdout, os.Stderr),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.UsageOnError(),
	)
	if err != nil {
		// A grammar defect (for example a duplicate flag) is a bug in this
		// binary, not a usage error; it surfaces before any argument parses.
		fmt.Fprintf(os.Stderr, "grammar: %v\n", err)
		return int(runtime.ExitSoftware)
	}

	kctx, err := parser.Parse(args[1:])
	if err != nil {
		parser.Errorf("%v", err)
		return int(runtime.ExitUsage)
	}

	global, paths, code := loadConfig(&cli, kctx)
	if code != 0 {
		return code
	}

	renderer, logger, code := setupOutput(global, paths)
	if code != 0 {
		return code
	}

	rt := runtime.NewRuntime(logger, renderer, global, paths)
	defer rt.Close()

	ctx, cancel := runtime.NewSignalContext(context.Background(), rt)
	defer cancel()

	if global.Debug {
		rt.Hooks().Before = append(rt.Hooks().Before, func(_ context.Context, rt *runtime.Runtime, cmd string) error {
			rt.Logger().Debug("dispatch", "command", cmd, "session", rt.SessionID(), "retained", rt.Paths().Retained)
			return nil
		})
	}

	kctx.Bind(rt)
	kctx.BindTo(ctx, (*context.Context)(nil))
	kctx.BindTo(&cli.Deploy, (*service.Deployment)(nil))
	err = runtime.RunCommand(ctx, rt, kctx.Command(), func(ctx context.Context) error {
		return kctx.Run()
	})

	code = int(runtime.ExitCodeFromError(ctx, err))
	switch {
	case err == nil, runtime.IsSilentExit(err):
	case ctx.Err() != nil && code == int(runtime.ExitOK):
		renderer.Warn("interrupted")
	default:
		logger.Debug("command failed", "error", err, "exit_code", code)
		renderer.Error("command failed: %v", err)
	}
	return code
}

// loadConfig resolves the working directory, session ID, user flags, and
// runtime paths from the parsed CLI. It returns a non-zero exit code when
// config loading or path resolution fails.
func loadConfig(cli *Root, kctx *kong.Context) (config.Global, runtime.Paths, int) {
	opts := config.LoadOptions{
		AppName:       appName,
		Cwd:           config.ResolveCwd(),
		ConfigFiles:   cli.ConfigFiles,
		EnvFiles:      cli.EnvFile,
		Profile:       cli.Profile,
		FlagOverrides: config.ExtractUserFlags(kctx),
	}

	k, _, err := config.Load(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return config.Global{}, runtime.Paths{}, int(runtime.ExitConfig)
	}
	global, err := config.ResolveGlobal(k, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return config.Global{}, runtime.Paths{}, int(runtime.ExitConfig)
	}
	if global.SessionID == "" {
		global.SessionID = runtime.GenerateSessionID()
	}

	paths, err := runtime.ResolvePaths(global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paths: %v\n", err)
		return config.Global{}, runtime.Paths{}, int(runtime.ExitSoftware)
	}
	return global, paths, 0
}

// setupOutput constructs the renderer and the slog logger from the resolved
// configuration. The mode comes from configuration only — never from
// whether stdout is a terminal.
func setupOutput(global config.Global, paths runtime.Paths) (*output.Output, *slog.Logger, int) {
	mode, err := output.ParseMode(global.OutputMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "output: %v\n", err)
		return nil, nil, int(runtime.ExitConfig)
	}
	opts := []output.Option{
		output.WithMode(mode),
		output.WithColor(output.ColorAuto(os.Stderr)),
		output.WithVerbosity(global.Verbose),
	}
	if global.Debug {
		// Debug runs capture every output call under the run's Retained
		// directory (or --debug-dir) for post-mortem inspection.
		dir := paths.Retained
		if global.DebugDir != "" {
			dir = global.DebugDir
		}
		opts = append(opts, output.WithRetainedDir(dir))
	}
	renderer, err := output.New(os.Stdout, os.Stderr, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "output: %v\n", err)
		return nil, nil, int(runtime.ExitConfig)
	}
	logger := slog.New(output.NewSlogHandler(renderer, global.LogLevel))
	return renderer, logger, 0
}
