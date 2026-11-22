package provider

import (
	"os"
	"strings"
)

// IsProviderEnabled reports whether providerName is enabled at call time.
// A provider is considered disabled when the environment variable
// DISABLE_{PROVIDER} is set to "true" (case-insensitive), where {PROVIDER}
// is the upper-cased provider identifier with dashes replaced by underscores.
func IsProviderEnabled(providerName string) bool {
	envKey := "DISABLE_" + strings.ToUpper(strings.ReplaceAll(providerName, "-", "_"))
	return !strings.EqualFold(os.Getenv(envKey), "true")
}
