package tool

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"example.com/cliproj/pkg/output"
	"example.com/cliproj/pkg/runtime"
)

// outputFormat selects how tool output is rendered to the caller.
type outputFormat string

const (
	// formatJSON routes the raw JSON result through the runtime Result sink,
	// letting the output mode (json, ndjson, text, terminal) handle rendering.
	formatJSON outputFormat = "json"

	// formatText attempts to unwrap string-valued output and render it as
	// plain text via rt.Success. Structured values fall back to JSON.
	formatText outputFormat = "text"

	// formatTable renders slice-of-struct values as tab-separated columns with
	// struct field names as headers. Non-tabular values fall back to JSON.
	formatTable outputFormat = "table"
)

// writeOutput renders the tool's raw JSON output according to the requested
// format. All rendering routes through the runtime so output modes continue
// to apply.
func writeOutput(rt *runtime.Runtime, raw json.RawMessage, format outputFormat) error {
	switch format {
	case "", formatJSON:
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return runtime.Exitf(runtime.ExitSoftware, "decode output: %v", err)
		}
		rt.Result(decoded)
		return nil

	case formatText:
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			rt.Success("%s", s)
			return nil
		}
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return runtime.Exitf(runtime.ExitSoftware, "decode output: %v", err)
		}
		rt.Result(decoded)
		return nil

	case formatTable:
		return writeTable(rt, raw)

	default:
		return runtime.Exit(runtime.ExitUsage, fmt.Errorf("unknown output format: %q", format))
	}
}

// writeTable renders a JSON array of objects as a table. The table goes
// through rt.Result like every other payload — never straight to os.Stdout —
// so text mode prints columns while json, ndjson, and yamlish still get the
// rows as data. Any other shape falls back to plain JSON output.
func writeTable(rt *runtime.Runtime, raw json.RawMessage) error {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "decode output: %v", err)
	}
	slice, ok := decoded.([]any)
	if !ok || len(slice) == 0 {
		rt.Result(decoded)
		return nil
	}
	if _, ok := slice[0].(map[string]any); !ok {
		rt.Result(decoded)
		return nil
	}
	rt.Result(table{rows: slice})
	return nil
}

// table is a result payload with its own text view.
type table struct {
	rows []any
}

// FormatText renders the rows as tab-aligned columns.
func (t table) FormatText(output.FmtArgs) string {
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	writeMapTable(tw, t.rows)
	_ = tw.Flush()
	return b.String()
}

// FormatData keeps the rows as data in structured modes.
func (t table) FormatData(output.FmtArgs) any { return t.rows }

// writeMapTable writes a slice of maps as a tab-separated table to w.
func writeMapTable(w io.Writer, rows []any) {
	if len(rows) == 0 {
		return
	}

	_, ok := rows[0].(map[string]any)
	if !ok {
		return
	}

	// Collect every key; sort for a deterministic column order (map
	// iteration order is random).
	seen := make(map[string]bool)
	var keys []string
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		for k := range m {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}

	sort.Strings(keys)

	// Header row.
	for i, k := range keys {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, k)
	}
	fmt.Fprintln(w)

	// Data rows.
	for _, row := range rows {
		m := row.(map[string]any)
		for i, k := range keys {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprintf(w, "%v", m[k])
		}
		fmt.Fprintln(w)
	}
}
