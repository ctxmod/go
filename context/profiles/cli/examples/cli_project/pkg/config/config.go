// Package config loads the effective configuration with koanf.
//
// Kong owns the command grammar, help surface, and flag definitions; koanf
// owns the configuration graph. Load encodes the precedence policy as call
// order, lowest first:
//
//	defaults < {exe}/configs/default.yaml < {exe}/configs/profile/{profile}.yaml
//	  < user config < project-local config < --config files
//	  < auto-discovered .env files < --env-file files
//	  < process environment ({APP}_*) < explicitly-set CLI flags
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// LoadOptions carries everything Load needs from Kong and the environment.
type LoadOptions struct {
	AppName       string         // application name for file discovery and the env prefix
	Cwd           string         // resolved working directory
	ExeDir        string         // executable directory for bundled configs; empty resolves os.Executable
	Profile       string         // --profile value
	ConfigFiles   []string       // --config values (explicit paths, must exist)
	EnvFiles      []string       // --env-file values (explicit paths, must exist)
	FlagOverrides map[string]any // only explicitly-set flags, keyed by config key
}

// Source names one loaded layer; Load returns them in load order so an
// "inspect config" command can report where each value came from.
type Source struct {
	Name string
	K    *koanf.Koanf
}

// Defaults are the lowest-precedence values.
var Defaults = map[string]any{
	"log.level":   "info",
	"output.mode": "text",
	"verbose":     0,
	"ai.steps":    8,
}

// Load merges every source in precedence order and returns the merged
// instance plus each layer, in load order, for provenance reporting.
func Load(opts LoadOptions) (*koanf.Koanf, []Source, error) {
	if opts.ExeDir == "" {
		if exe, err := os.Executable(); err == nil {
			opts.ExeDir = filepath.Dir(exe)
		}
	}
	prefix := strings.ToUpper(opts.AppName) + "_"
	envKey := func(key string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(key, prefix)), "_", ".")
	}
	// Dotenv files use the same {APP}_ names as the process environment.
	dotenvParser := dotenv.ParserEnv(prefix, ".", envKey)
	fromFile := func(path string, required bool) func(k *koanf.Koanf) error {
		return func(k *koanf.Koanf) error { return loadFile(k, path, required, dotenvParser) }
	}
	userDir := userConfigDir(opts.AppName)

	type step struct {
		name string
		load func(k *koanf.Koanf) error
	}
	steps := []step{
		{"defaults", func(k *koanf.Koanf) error { return k.Load(confmap.Provider(Defaults, "."), nil) }},
		{"bundled", fromFile(filepath.Join(opts.ExeDir, "configs", "default.yaml"), false)},
	}
	if opts.Profile != "" {
		steps = append(steps, step{"profile", fromFile(filepath.Join(opts.ExeDir, "configs", "profile", opts.Profile+".yaml"), false)})
	}
	steps = append(steps,
		step{"user", fromFile(filepath.Join(userDir, "config.yaml"), false)},
		step{"local", fromFile(filepath.Join(opts.Cwd, opts.AppName+".yaml"), false)},
		step{"local-hidden", fromFile(filepath.Join(opts.Cwd, "."+opts.AppName+".yaml"), false)},
	)
	for _, p := range opts.ConfigFiles {
		steps = append(steps, step{"config " + p, fromFile(p, true)})
	}
	steps = append(steps,
		step{"dotenv", fromFile(filepath.Join(opts.Cwd, ".env"), false)},
		step{"dotenv-app", fromFile(filepath.Join(opts.Cwd, "."+opts.AppName+".env"), false)},
		step{"dotenv-user", fromFile(filepath.Join(userDir, ".env"), false)},
	)
	for _, p := range opts.EnvFiles {
		steps = append(steps, step{"env-file " + p, func(k *koanf.Koanf) error { return k.Load(file.Provider(p), dotenvParser) }})
	}
	steps = append(steps, step{"env", func(k *koanf.Koanf) error {
		return k.Load(env.Provider(".", env.Opt{
			Prefix:        prefix,
			TransformFunc: func(key, val string) (string, any) { return envKey(key), val },
		}), nil)
	}})
	if len(opts.FlagOverrides) > 0 {
		steps = append(steps, step{"flags", func(k *koanf.Koanf) error { return k.Load(confmap.Provider(opts.FlagOverrides, "."), nil) }})
	}

	final := koanf.New(".")
	var sources []Source
	for _, st := range steps {
		k := koanf.New(".")
		if err := st.load(k); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", st.name, err)
		}
		if len(k.Keys()) == 0 {
			continue
		}
		sources = append(sources, Source{Name: st.name, K: k})
		if err := final.Merge(k); err != nil {
			return nil, nil, fmt.Errorf("%s: merge: %w", st.name, err)
		}
	}
	return final, sources, nil
}

// Global is the typed view of the settings every command shares.
type Global struct {
	AppName    string
	SessionID  string
	Cwd        string
	Debug      bool
	DebugDir   string
	LogLevel   slog.Level
	OutputMode string
	Verbose    int
	TempDir    string
	CacheDir   string
	OutputDir  string
	AI         AI
}

// AI holds the model settings the AI commands read.
type AI struct {
	Provider   string `koanf:"provider"`
	Model      string `koanf:"model"`
	ModelClass string `koanf:"class"`
	APIKey     string `koanf:"key"`
	MaxSteps   int    `koanf:"steps"`
}

// ResolveGlobal decodes the merged configuration into Global.
func ResolveGlobal(k *koanf.Koanf, opts LoadOptions) (Global, error) {
	g := Global{
		AppName:    opts.AppName,
		Cwd:        opts.Cwd,
		Debug:      k.Bool("debug.enabled"),
		DebugDir:   k.String("debug.dir"),
		OutputMode: k.String("output.mode"),
		Verbose:    k.Int("verbose"),
		TempDir:    k.String("paths.temp"),
		CacheDir:   k.String("paths.cache"),
		OutputDir:  k.String("paths.output"),
		SessionID:  k.String("session"),
	}
	if err := g.LogLevel.UnmarshalText([]byte(k.String("log.level"))); err != nil {
		return Global{}, fmt.Errorf("log.level: %w", err)
	}
	if err := k.UnmarshalWithConf("ai", &g.AI, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return Global{}, fmt.Errorf("ai: %w", err)
	}
	return g, nil
}

// ResolveCwd resolves the working directory once: INIT_CWD, then PWD, then ".".
func ResolveCwd() string {
	if v := os.Getenv("INIT_CWD"); v != "" {
		return v
	}
	if v := os.Getenv("PWD"); v != "" {
		return v
	}
	return "."
}

func userConfigDir(appName string) string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName)
}

// loadFile loads path with the parser its extension selects. A missing
// auto-discovered file is skipped; a missing explicit file is an error.
func loadFile(k *koanf.Koanf, path string, required bool, dotenvParser koanf.Parser) error {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) && !required {
		return nil
	}
	return k.Load(file.Provider(path), parserFor(path, dotenvParser))
}

func parserFor(path string, dotenvParser koanf.Parser) koanf.Parser {
	switch {
	case strings.HasSuffix(path, ".env"):
		return dotenvParser
	case strings.HasSuffix(path, ".json"):
		return json.Parser()
	default:
		return yaml.Parser()
	}
}
