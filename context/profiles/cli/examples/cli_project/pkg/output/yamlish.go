package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// yamlish renders v as dense, YAML-like text: maps as `key: value` lines,
// slices as `- item` lines, nested values indented by two spaces. It is for
// humans, not parsers; machine consumers use json or ndjson.
func yamlish(v any, indent int) string {
	v = normalize(v)
	pad := strings.Repeat("  ", indent)
	var b strings.Builder
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if isScalar(t[k]) {
				fmt.Fprintf(&b, "%s%s: %s\n", pad, k, scalar(t[k]))
				continue
			}
			fmt.Fprintf(&b, "%s%s:\n%s", pad, k, yamlish(t[k], indent+1))
		}
	case []any:
		for _, item := range t {
			if isScalar(item) {
				fmt.Fprintf(&b, "%s- %s\n", pad, scalar(item))
				continue
			}
			fmt.Fprintf(&b, "%s-\n%s", pad, yamlish(item, indent+1))
		}
	default:
		fmt.Fprintf(&b, "%s%s\n", pad, scalar(t))
	}
	return b.String()
}

// normalize turns structs and typed collections into the generic shapes
// encoding/json produces, so yamlish honors json tags.
func normalize(v any) any {
	switch v.(type) {
	case nil, string, bool, float64, int, int64, map[string]any, []any:
		return v
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Sprint(v)
	}
	return out
}

func isScalar(v any) bool {
	switch normalize(v).(type) {
	case map[string]any, []any:
		return false
	}
	return true
}

func scalar(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprint(v)
}
