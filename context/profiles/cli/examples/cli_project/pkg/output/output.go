// Package output is the mode-aware renderer every command writes through.
//
// Commands call Info, Progress, Result, and friends without knowing whether
// the run renders human text, one JSON document, an NDJSON event stream, a
// yamlish view, or nothing at all. The mode is chosen once, at construction,
// from configuration — never by sniffing whether stdout is a terminal — so a
// piped and an interactive invocation of the same command behave the same.
//
// Stream discipline: stdout carries only the result payload; diagnostics,
// progress, and interactive UI go to stderr.
package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// Mode selects how results and diagnostics render.
type Mode string

// The six output modes.
const (
	ModeText     Mode = "text"     // human-readable result on stdout
	ModeJSON     Mode = "json"     // one JSON document on stdout at Close
	ModeNDJSON   Mode = "ndjson"   // one JSON record per line on stdout
	ModeYamlish  Mode = "yamlish"  // dense YAML-like text on stdout
	ModeQuiet    Mode = "quiet"    // nothing on stdout; the exit code is the outcome
	ModeTerminal Mode = "terminal" // interactive rendering; falls back to text off a TTY
)

// ValidModes lists every accepted mode. Flag enums that name modes must stay
// in sync with this list (see the grammar footguns reference).
var ValidModes = []Mode{ModeText, ModeJSON, ModeNDJSON, ModeYamlish, ModeQuiet, ModeTerminal}

// ParseMode parses a mode name. The empty string selects ModeText.
func ParseMode(s string) (Mode, error) {
	if s == "" {
		return ModeText, nil
	}
	m := Mode(strings.ToLower(s))
	if slices.Contains(ValidModes, m) {
		return m, nil
	}
	names := make([]string, len(ValidModes))
	for i, v := range ValidModes {
		names[i] = string(v)
	}
	return "", fmt.Errorf("unknown output mode %q (valid: %s)", s, strings.Join(names, ", "))
}

// FmtArgs carries the resolved formatting context into formatting
// interfaces, so implementations never consult global state.
type FmtArgs struct {
	Format   Mode
	UseColor bool
}

// TextFormattable renders its own text view. When args.UseColor is false
// the implementation must return plain text.
type TextFormattable interface {
	FormatText(args FmtArgs) string
}

// DataFormattable supplies the canonical payload for every structured mode
// (json, ndjson, yamlish); it may branch on args.Format.
type DataFormattable interface {
	FormatData(args FmtArgs) any
}

// Option configures an Output.
type Option func(*Output)

// WithMode selects the output mode.
func WithMode(m Mode) Option { return func(o *Output) { o.mode = m } }

// WithColor enables ANSI color on stderr diagnostics and in FmtArgs.
func WithColor(on bool) Option { return func(o *Output) { o.color = on } }

// WithVerbosity sets how many diagnostic levels reach stderr: 0 (the
// default) shows warnings and errors only, 1 adds info, success, progress,
// and checkpoints, and 2 adds lifecycle events in text mode.
func WithVerbosity(n int) Option { return func(o *Output) { o.verbosity = n } }

// WithRetainedDir captures every output call to a timestamped log file in
// dir, for post-mortem diagnostics that never touch stdout or stderr.
func WithRetainedDir(dir string) Option { return func(o *Output) { o.retainedDir = dir } }

// Output is the concrete renderer. Its method set is the output contract;
// a consumer that wants a narrower seam declares its own interface.
type Output struct {
	mode        Mode
	color       bool
	verbosity   int
	retainedDir string

	mu       sync.Mutex
	stdout   io.Writer
	stderr   io.Writer
	results  []any
	failure  *failure
	capture  *bufio.Writer
	captureF *os.File
	closed   bool
}

type failure struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// New constructs an Output writing to stdout and stderr.
func New(stdout, stderr io.Writer, opts ...Option) (*Output, error) {
	o := &Output{mode: ModeText, stdout: stdout, stderr: stderr}
	for _, opt := range opts {
		opt(o)
	}
	if _, err := ParseMode(string(o.mode)); err != nil {
		return nil, err
	}
	if o.mode == ModeTerminal && !isTerminal(stderr) {
		fmt.Fprintln(stderr, "output: terminal mode needs a terminal on stderr; falling back to text")
		o.mode = ModeText
	}
	if o.retainedDir != "" {
		if err := os.MkdirAll(o.retainedDir, 0o755); err != nil {
			return nil, fmt.Errorf("output: create retained dir: %w", err)
		}
		f, err := os.Create(filepath.Join(o.retainedDir, "output.log"))
		if err != nil {
			return nil, fmt.Errorf("output: create capture file: %w", err)
		}
		o.captureF, o.capture = f, bufio.NewWriter(f)
	}
	return o, nil
}

// Mode reports the effective output mode.
func (o *Output) Mode() Mode { return o.mode }

// FmtArgs reports the formatting context handed to formatting interfaces.
func (o *Output) FmtArgs() FmtArgs { return FmtArgs{Format: o.mode, UseColor: o.color} }

// Interactive reports whether stderr is a terminal a TUI can draw on.
func (o *Output) Interactive() bool { return isTerminal(o.stderr) }

// Info reports progress-worthy information on stderr.
func (o *Output) Info(msg string, args ...any) { o.diag("info", 1, msg, args...) }

// Warn reports a recoverable problem on stderr.
func (o *Output) Warn(msg string, args ...any) { o.diag("warn", 0, msg, args...) }

// Error reports a failure on stderr; it is never suppressed.
func (o *Output) Error(msg string, args ...any) { o.diag("error", -1, msg, args...) }

// Debug writes straight to stderr in every mode. Callers gate it on an
// explicit debug request (the slog handler's level does), and it bypasses
// the mode and verbosity filters on purpose: quiet mode suppresses every
// other diagnostic, and routing requested debug output through that filter
// would drop it silently.
func (o *Output) Debug(msg string, args ...any) {
	text := fmt.Sprintf(msg, args...)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.captureLine("debug", text)
	fmt.Fprintf(o.stderr, "debug: %s\n", text)
}

// Success reports a completed step on stderr.
func (o *Output) Success(msg string, args ...any) { o.diag("success", 1, msg, args...) }

// Progress reports an in-flight step on stderr.
func (o *Output) Progress(msg string, args ...any) { o.diag("progress", 1, msg, args...) }

// Status reports a one-line state change on stderr.
func (o *Output) Status(msg string) { o.diag("status", 1, "%s", msg) }

// Checkpoint marks a named milestone on stderr.
func (o *Output) Checkpoint(id, msg string) { o.diag("checkpoint", 1, "%s: %s", id, msg) }

// Fatal reports err, closes the output, and exits non-zero. Prefer returning
// an error from Run; Fatal is for failures before the runtime exists.
func (o *Output) Fatal(err error) {
	o.diag("fatal", -1, "%v", err)
	_ = o.Close()
	os.Exit(1)
}

// Started records the start of an operation. Lifecycle events reach stderr
// at verbosity 2 (kind-discriminated JSON lines in json and ndjson); the
// json envelope and the ndjson terminal record carry the final status at
// every verbosity.
func (o *Output) Started(operation string, metadata map[string]any) error {
	o.event("started", map[string]any{"operation": operation, "metadata": metadata})
	return nil
}

// Completed records successful completion.
func (o *Output) Completed(result map[string]any) error {
	o.event("completed", map[string]any{"status": "ok", "result": result})
	return nil
}

// Failed records failure; json and ndjson carry it into the final status.
func (o *Output) Failed(code, message string, details map[string]any) error {
	o.mu.Lock()
	o.failure = &failure{Code: code, Message: message, Details: details}
	o.mu.Unlock()
	o.event("failed", map[string]any{"code": code, "message": message, "details": details})
	return nil
}

// Result emits a result payload on stdout. JSON mode buffers results into
// one envelope written at Close, so a test must close before it asserts.
func (o *Output) Result(data any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.captureLine("result", fmt.Sprint(data))
	switch o.mode {
	case ModeQuiet:
	case ModeJSON:
		o.results = append(o.results, o.dataOf(data))
	case ModeNDJSON:
		o.writeJSONLine(o.stdout, map[string]any{"kind": "result", "value": o.dataOf(data)})
	case ModeYamlish:
		fmt.Fprint(o.stdout, yamlish(o.dataOf(data), 0))
	default:
		fmt.Fprintln(o.stdout, o.textOf(data))
	}
}

// Close flushes buffered output (the JSON envelope, the NDJSON terminal
// record) and the capture file. It is safe to call more than once.
func (o *Output) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil
	}
	o.closed = true
	status := "ok"
	if o.failure != nil {
		status = "failed"
	}
	switch o.mode {
	case ModeJSON:
		env := map[string]any{"version": 1, "status": status, "value": o.results, "result_records": len(o.results)}
		if o.results == nil {
			env["value"] = []any{}
		}
		if o.failure != nil {
			env["code"], env["message"] = o.failure.Code, o.failure.Message
		}
		enc := json.NewEncoder(o.stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(env)
	case ModeNDJSON:
		o.writeJSONLine(o.stdout, map[string]any{"kind": "completed", "status": status})
	}
	if o.capture != nil {
		_ = o.capture.Flush()
		return o.captureF.Close()
	}
	return nil
}

func (o *Output) diag(kind string, level int, msg string, args ...any) {
	text := fmt.Sprintf(msg, args...)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.captureLine(kind, text)
	if level > o.verbosity {
		return
	}
	switch o.mode {
	case ModeJSON, ModeNDJSON:
		o.writeJSONLine(o.stderr, map[string]any{"kind": kind, "message": text})
	case ModeQuiet:
		if level < 0 {
			fmt.Fprintf(o.stderr, "%s: %s\n", kind, text)
		}
	default:
		fmt.Fprintf(o.stderr, "%s %s\n", o.paint(kind), text)
	}
}

func (o *Output) event(kind string, fields map[string]any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.captureLine(kind, fmt.Sprint(fields))
	if o.verbosity < 2 {
		return
	}
	switch o.mode {
	case ModeJSON, ModeNDJSON:
		fields["kind"] = kind
		o.writeJSONLine(o.stderr, fields)
	case ModeText, ModeTerminal:
		fmt.Fprintf(o.stderr, "%s %v\n", o.paint(kind), fields)
	}
}

func (o *Output) dataOf(data any) any {
	if d, ok := data.(DataFormattable); ok {
		return d.FormatData(o.FmtArgs())
	}
	return data
}

func (o *Output) textOf(data any) string {
	if t, ok := data.(TextFormattable); ok {
		return strings.TrimRight(t.FormatText(o.FmtArgs()), "\n")
	}
	switch v := data.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	}
	return strings.TrimRight(yamlish(o.dataOf(data), 0), "\n")
}

func (o *Output) writeJSONLine(w io.Writer, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(map[string]any{"kind": "error", "message": err.Error()})
	}
	fmt.Fprintf(w, "%s\n", b)
}

func (o *Output) captureLine(kind, text string) {
	if o.capture != nil {
		fmt.Fprintf(o.capture, "%s %s: %s\n", time.Now().UTC().Format(time.RFC3339Nano), kind, text)
	}
}

func (o *Output) paint(kind string) string {
	label := kind + ":"
	if !o.color {
		return label
	}
	code := map[string]string{"error": "31", "fatal": "31", "warn": "33", "success": "32", "checkpoint": "36"}[kind]
	if code == "" {
		code = "2"
	}
	return "\x1b[" + code + "m" + label + "\x1b[0m"
}

// ColorAuto reports whether color suits w: it is a terminal and NO_COLOR
// is unset.
func ColorAuto(w io.Writer) bool {
	return os.Getenv("NO_COLOR") == "" && isTerminal(w)
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
