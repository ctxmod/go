package retry

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// mcpServerModeEnv names the environment variable that signals the process
// is running as an MCP server. Speedy mode stays off in MCP mode because
// MCP callers are long-lived background processes, not interactive humans.
const mcpServerModeEnv = "MCP_SERVER_MODE"

// speedyModeEnv names the explicit override for speedy behavior. "1",
// "true", or "yes" force it on; "0", "false", or "no" force it off.
const speedyModeEnv = "SPEEDY_MODE"

// shouldUseSpeedyMode reports whether the retry executor should substitute
// the SPEEDY profile for the caller's chosen profile. The precedence, from
// strongest to weakest:
//  1. Explicit SPEEDY_MODE env var (on or off).
//  2. Default: on when stdout is a TTY and MCP_SERVER_MODE is unset.
func shouldUseSpeedyMode() bool {
	if explicit, ok := parseSpeedyEnv(os.Getenv(speedyModeEnv)); ok {
		return explicit
	}
	if os.Getenv(mcpServerModeEnv) != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// applySpeedyMode returns SPEEDY if shouldUseSpeedyMode is true, otherwise
// returns config unchanged. The helper is the canonical entry point for
// callers that want to opt into speedy retries without re-implementing the
// detection logic.
func applySpeedyMode(config RetryConfig) RetryConfig {
	if shouldUseSpeedyMode() {
		return SPEEDY
	}
	return config
}

// parseSpeedyEnv interprets an env var value as a boolean flag. The second
// return value reports whether the value was an explicit override; an empty
// string returns (false, false) so callers fall through to default logic.
func parseSpeedyEnv(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return false, false
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
