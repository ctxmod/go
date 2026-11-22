package output

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// NewSlogHandler returns an slog.Handler that routes records through out:
// Info, Warn, and Error map to the matching Output methods and Debug to
// Output.Debug, so log lines obey the same mode and stream discipline as
// every other diagnostic. Attributes render inline as `key=value` pairs.
func NewSlogHandler(out *Output, level slog.Leveler) slog.Handler {
	return &slogHandler{out: out, level: level}
}

type slogHandler struct {
	out    *Output
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
}

func (h *slogHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	write := func(a slog.Attr) bool {
		key := a.Key
		if len(h.groups) > 0 {
			key = strings.Join(h.groups, ".") + "." + key
		}
		fmt.Fprintf(&b, " %s=%v", key, a.Value.Any())
		return true
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(write)
	msg := b.String()
	switch {
	case r.Level >= slog.LevelError:
		h.out.Error("%s", msg)
	case r.Level >= slog.LevelWarn:
		h.out.Warn("%s", msg)
	case r.Level >= slog.LevelInfo:
		h.out.Info("%s", msg)
	default:
		h.out.Debug("%s", msg)
	}
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &c
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	c := *h
	c.groups = append(append([]string{}, h.groups...), name)
	return &c
}
