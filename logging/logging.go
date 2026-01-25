package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
)

const LevelTrace slog.Level = -8

func ParseLevel(levelFlag string) (slog.Level, string) {
	switch strings.ToLower(levelFlag) {
	case "trace":
		return LevelTrace, "trace"
	case "debug":
		return slog.LevelDebug, "debug"
	case "warn", "warning":
		return slog.LevelWarn, "warn"
	case "error":
		return slog.LevelError, "error"
	default:
		return slog.LevelInfo, "info"
	}
}

func NewHandler(format string, level slog.Level, stdout *os.File, out io.Writer) (slog.Handler, bool, string) {
	if out == nil {
		out = io.Discard
	}

	switch format {
	case "json":
		return newJSONHandler(out, level), true, "Output format set to 'json'"
	case "text":
		return newTextHandler(out, level), false, "Output format set to 'text'"
	default:
		if isPipe(stdout) {
			return newJSONHandler(out, level), true, "Output format automatically set to 'json'"
		}
		return newTextHandler(out, level), false, "Output format automatically set to 'text'"
	}
}

func isPipe(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

type textHandler struct {
	mu     *sync.Mutex
	out    io.Writer
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func newTextHandler(out io.Writer, level slog.Level) *textHandler {
	return &textHandler{
		mu:    &sync.Mutex{},
		out:   out,
		level: level,
	}
}

func (h *textHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *textHandler) Handle(ctx context.Context, record slog.Record) error {
	if !h.Enabled(ctx, record.Level) {
		return nil
	}

	var b strings.Builder
	b.WriteString(record.Time.Format("15:04:05"))
	b.WriteString(" [")
	b.WriteString(strings.ToUpper(levelName(record.Level)))
	b.WriteString("] (tezbake) ")
	b.WriteString(record.Message)
	b.WriteByte('\n')

	attrs := make([]slog.Attr, 0, len(h.attrs)+record.NumAttrs())
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	prefix := ""
	if len(h.groups) > 0 {
		prefix = strings.Join(h.groups, ".") + "."
	}

	for _, attr := range attrs {
		appendAttrLines(&b, prefix, attr)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := io.WriteString(h.out, b.String())
	return err
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *textHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

type jsonHandler struct {
	mu     *sync.Mutex
	out    io.Writer
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func newJSONHandler(out io.Writer, level slog.Level) *jsonHandler {
	return &jsonHandler{
		mu:    &sync.Mutex{},
		out:   out,
		level: level,
	}
}

func (h *jsonHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *jsonHandler) Handle(ctx context.Context, record slog.Record) error {
	if !h.Enabled(ctx, record.Level) {
		return nil
	}

	payload := map[string]any{
		"level":     levelName(record.Level),
		"msg":       record.Message,
		"timestamp": strconv.FormatInt(record.Time.Unix(), 10),
		"module":    "tezbake",
	}

	attrs := make([]slog.Attr, 0, len(h.attrs)+record.NumAttrs())
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	prefix := ""
	if len(h.groups) > 0 {
		prefix = strings.Join(h.groups, ".") + "."
	}

	for _, attr := range attrs {
		appendAttrMap(payload, prefix, attr)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data = append(data, byte('\n'))

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err = h.out.Write(data)
	return err
}

func (h *jsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *jsonHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func levelName(level slog.Level) string {
	switch {
	case level < slog.LevelDebug:
		return "trace"
	case level < slog.LevelInfo:
		return "debug"
	case level < slog.LevelWarn:
		return "info"
	case level < slog.LevelError:
		return "warning"
	default:
		return "error"
	}
}

func slogValueAny(value slog.Value) any {
	switch value.Kind() {
	case slog.KindAny:
		if err, ok := value.Any().(error); ok {
			return err.Error()
		}
		return value.Any()
	case slog.KindBool:
		return value.Bool()
	case slog.KindDuration:
		return value.Duration()
	case slog.KindFloat64:
		return value.Float64()
	case slog.KindInt64:
		return value.Int64()
	case slog.KindString:
		return value.String()
	case slog.KindTime:
		return value.Time()
	case slog.KindUint64:
		return value.Uint64()
	default:
		return value.Any()
	}
}

func appendAttrLines(b *strings.Builder, prefix string, attr slog.Attr) {
	value := attr.Value.Resolve()
	key := prefix + attr.Key
	if value.Kind() == slog.KindGroup {
		groupPrefix := key + "."
		for _, groupAttr := range value.Group() {
			appendAttrLines(b, groupPrefix, groupAttr)
		}
		return
	}

	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(fmt.Sprint(slogValueAny(value)))
	b.WriteByte('\n')
}

func appendAttrMap(dst map[string]any, prefix string, attr slog.Attr) {
	value := attr.Value.Resolve()
	key := prefix + attr.Key
	if value.Kind() == slog.KindGroup {
		groupPrefix := key + "."
		for _, groupAttr := range value.Group() {
			appendAttrMap(dst, groupPrefix, groupAttr)
		}
		return
	}

	dst[key] = slogValueAny(value)
}
