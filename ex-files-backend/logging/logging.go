package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// traceHandler wraps a slog.Handler to attach trace_id / span_id attributes
// from the current OpenTelemetry span (if any). Loki indexes neither, but
// they're queryable via `| json` and Grafana's derived-fields config jumps
// to the matching trace in Tempo.
type traceHandler struct{ slog.Handler }

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{Handler: h.Handler.WithGroup(name)}
}

// Init configures the default slog logger with a JSON handler that also
// attaches OTEL trace IDs when present.
// Log level is read from the LOG_LEVEL env var (debug, info, warn, error).
// Defaults to info if unset or unrecognised.
func Init() {
	var level slog.Level
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(traceHandler{Handler: base}))
}

// Audit emits a structured audit event log line. Promtail extracts `kind` and
// `event` as Loki labels, so dashboards filter with `{kind="audit", event="..."}`
// and read the remaining attrs (actor_id, target_id, target_type, ...) via `| json`.
func Audit(ctx context.Context, event string, actorID uint, attrs ...slog.Attr) {
	base := []slog.Attr{
		slog.String("kind", "audit"),
		slog.String("event", event),
		slog.Uint64("actor_id", uint64(actorID)),
	}
	slog.LogAttrs(ctx, slog.LevelInfo, "audit", append(base, attrs...)...)
}
