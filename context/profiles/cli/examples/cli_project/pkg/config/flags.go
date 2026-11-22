package config

import (
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// loadingFlags feed Load itself, so they never become config keys.
var loadingFlags = map[string]bool{"config": true, "env-file": true, "profile": true, "help": true}

// ExtractUserFlags walks the parsed Kong context and returns only the flags
// the user explicitly set — on the command line, or through one of the
// flag's env bindings — keyed by config key. Unset flags are absent (no
// zero-value or default injection), so they never override a value from a
// lower layer. Kong's own Value.Set is no guide here: applying a default
// marks a value set too, so the walk reads the parse path instead.
//
// The config key comes from a `koanf:"..."` struct tag on the flag when
// present (Kong keeps unknown tag keys), else from the flag name with
// dashes turned into dots.
func ExtractUserFlags(kctx *kong.Context) map[string]any {
	explicit := map[*kong.Flag]bool{}
	for _, p := range kctx.Path {
		if p.Flag != nil {
			explicit[p.Flag] = true
		}
	}
	out := map[string]any{}
	for _, f := range kctx.Flags() {
		if loadingFlags[f.Name] || !(explicit[f] || envSet(f.Tag.Envs)) {
			continue
		}
		key := f.Tag.Get("koanf")
		if key == "" {
			key = strings.ReplaceAll(f.Name, "-", ".")
		}
		out[key] = kctx.FlagValue(f)
	}
	return out
}

func envSet(names []string) bool {
	for _, n := range names {
		if v, ok := os.LookupEnv(n); ok && v != "" {
			return true
		}
	}
	return false
}
