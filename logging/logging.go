package logging

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/tez-capital/tezbake/constants"
)

const levelTrace slog.Level = -8

type PrettyHandlerOptions struct {
	slog.HandlerOptions
	NoColor bool
}

type PrettyTextLogHandler struct {
	slog.Handler
	l *log.Logger

	attrs   map[string][]slog.Attr
	groups  []string
	noColor bool
}

func isHiddenAttr(attr slog.Attr) bool {
	_, found := slices.BinarySearch(constants.LOG_TOP_LEVEL_HIDDEN_FIELDS, attr.Key)
	return found
}

const (
	colorReset   = "\033[0m"
	colorMagenta = "\033[35m"
	colorBlue    = "\033[34m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
)

func colorize(value string, color string, enabled bool) string {
	if !enabled || color == "" {
		return value
	}
	return color + value + colorReset
}

func levelColor(level slog.Level) string {
	switch {
	case level < slog.LevelDebug:
		return colorCyan
	case level < slog.LevelInfo:
		return colorMagenta
	case level < slog.LevelWarn:
		return colorBlue
	case level < slog.LevelError:
		return colorYellow
	default:
		return colorRed
	}
}

func levelName(level slog.Level) string {
	switch {
	case level < slog.LevelDebug:
		return "TRACE"
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARNING"
	default:
		return "ERROR"
	}
}

func (h *PrettyTextLogHandler) Handle(ctx context.Context, r slog.Record) error {
	fields := make(map[string]any, r.NumAttrs())

	for groupId, group := range h.attrs {
		for _, attr := range group {
			if groupId == "" {
				if isHiddenAttr(attr) {
					delete(fields, attr.Key)
					continue
				}
				fields[attr.Key] = attr.Value.Any()
			} else {
				if m, ok := fields[groupId].(map[string]any); ok {
					m[attr.Key] = attr.Value.Any()
				} else {
					fields[groupId] = map[string]any{
						attr.Key: attr.Value.Any(),
					}
				}
			}
		}
	}

	r.Attrs(func(a slog.Attr) bool {
		if !isHiddenAttr(a) {
			switch {
			case a.Key == "error" && a.Value.String() != "":
				fields[a.Key] = strings.Split(a.Value.String(), "\n")
			default:
				fields[a.Key] = a.Value.Any()
			}
		}
		return true
	})

	var fieldsSerializedRaw []byte
	if len(fields) != 0 {
		var err error
		fieldsSerializedRaw, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			slog.Error("failed to serialize fields", "error", err.Error())
		}
	}

	fieldsSerialized := string(fieldsSerializedRaw)
	if !h.noColor {
		fieldsSerialized = colorize(fieldsSerialized, colorWhite, true)
	}

	var b strings.Builder
	b.WriteString(r.Time.Format("15:04:05"))
	b.WriteString(" [")
	b.WriteString(colorize(strings.ToUpper(levelName(r.Level)), levelColor(r.Level), !h.noColor))
	b.WriteString("] (tezbake) ")
	b.WriteString(r.Message)

	h.l.Println(&b, fieldsSerialized)

	return nil
}

func (h *PrettyTextLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := maps.Clone(h.attrs)
	groupId := ""
	if len(h.groups) != 0 {
		groupId = h.groups[len(h.groups)-1]
	}
	newAttrs[groupId] = append(newAttrs[groupId], attrs...)

	return &PrettyTextLogHandler{
		Handler: h.Handler.WithAttrs(attrs),
		l:       h.l,
		attrs:   newAttrs,
		groups:  slices.Clone(h.groups),
		noColor: h.noColor,
	}
}

func (h *PrettyTextLogHandler) WithGroup(name string) slog.Handler {
	return &PrettyTextLogHandler{
		Handler: h.Handler.WithGroup(name),
		l:       h.l,
		attrs:   maps.Clone(h.attrs),
		groups:  append(h.groups, name),
		noColor: h.noColor,
	}
}

func NewPrettyTextLogHandler(
	out io.Writer,
	opts PrettyHandlerOptions,
) *PrettyTextLogHandler {
	h := &PrettyTextLogHandler{
		Handler: slog.NewJSONHandler(out, &opts.HandlerOptions),
		l:       log.New(out, "", 0),
		attrs:   make(map[string][]slog.Attr),
		noColor: opts.NoColor,
	}

	return h
}

type MultiWriter struct {
	writers []io.Writer
}

func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

type SlogMultiHandler struct {
	handlers []slog.Handler
}

func NewSlogMultiHandler(handlers ...slog.Handler) *SlogMultiHandler {
	return &SlogMultiHandler{
		handlers: handlers,
	}
}

func (h *SlogMultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, r.Level) {
			continue
		}
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *SlogMultiHandler) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

func (h *SlogMultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		newHandlers = append(newHandlers, handler.WithAttrs(attrs))
	}
	return &SlogMultiHandler{
		handlers: newHandlers,
	}
}

func (h *SlogMultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		newHandlers = append(newHandlers, handler.WithGroup(name))
	}
	return &SlogMultiHandler{
		handlers: newHandlers,
	}
}

func ParseLevel(levelFlag string) (slog.Level, string) {
	switch strings.ToLower(levelFlag) {
	case "trace":
		return levelTrace, "trace"
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
