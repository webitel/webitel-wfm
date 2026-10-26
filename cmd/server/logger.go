package server

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	"go.uber.org/fx"

	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/internal/model"
)

func ProvideLogger(cfg *config.Config, lc fx.Lifecycle) (*slog.Logger, error) {
	settings := cfg.Log
	if !settings.Console && !settings.Otel && settings.File == "" {
		settings.Console = true
	}

	level := parseLevel(settings.Level)
	opts := &slog.HandlerOptions{Level: level}

	var handlers []slog.Handler

	if settings.Console {
		handlers = append(handlers, newHandler(os.Stdout, settings.JSON, opts))
	}

	if settings.File != "" {
		f, err := os.OpenFile(settings.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{OnStop: func(context.Context) error { return f.Close() }})

		handlers = append(handlers, newHandler(f, settings.JSON, opts))
	}

	if settings.Otel {
		service := resource.NewSchemaless(
			semconv.ServiceName(model.ServiceName),
			semconv.ServiceVersion(model.Version),
			semconv.ServiceInstanceID(cfg.Service.NodeID),
		)

		shutdown, err := otelsdk.Configure(context.Background(),
			otelsdk.WithResource(service),
			otelsdk.WithLogBridge(func() {
				handlers = append(handlers, levelHandler{level: level, Handler: otelslog.NewHandler("slog")})
			}),
		)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{OnStop: func(ctx context.Context) error { return shutdown(ctx) }})
	}

	// A configuration that asks only for otel still has to log: the exporter
	// may be disabled or unconfigured, and a handlerless logger is silent.
	var handler slog.Handler

	switch len(handlers) {
	case 0:
		handler = newHandler(os.Stdout, settings.JSON, opts)
	case 1:
		handler = handlers[0]
	default:
		handler = multiHandler{handlers: handlers}
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

func newHandler(w io.Writer, json bool, opts *slog.HandlerOptions) slog.Handler {
	if json {
		return slog.NewJSONHandler(w, opts)
	}

	return slog.NewTextHandler(w, opts)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// levelHandler bounds a handler that has no level of its own.
type levelHandler struct {
	slog.Handler

	level slog.Level
}

func (h levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.Handler.Enabled(ctx, level)
}

func (h levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelHandler{Handler: h.Handler.WithAttrs(attrs), level: h.level}
}

func (h levelHandler) WithGroup(name string) slog.Handler {
	return levelHandler{Handler: h.Handler.WithGroup(name), level: h.level}
}

// multiHandler fans every record out to all configured handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (h multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (h multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, r.Level) {
			_ = hh.Handle(ctx, r)
		}
	}

	return nil
}

func (h multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		out[i] = hh.WithAttrs(attrs)
	}

	return multiHandler{handlers: out}
}

func (h multiHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		out[i] = hh.WithGroup(name)
	}

	return multiHandler{handlers: out}
}
