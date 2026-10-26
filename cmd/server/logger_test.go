package server

import (
	"context"
	"log/slog"
	"testing"

	"go.uber.org/fx/fxtest"

	"github.com/webitel/webitel-go-kit/appconfig"

	"github.com/webitel/webitel-wfm/config"
)

// A configuration that asks only for otel must still produce a logger that
// logs. The exporter may be disabled or unconfigured, in which case the bridge
// never installs a handler — and a handlerless logger is silent, including for
// the startup errors that would explain the misconfiguration.
func TestProvideLoggerNeverGoesSilent(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")

	cfg := &config.Config{Log: appconfig.Log{Otel: true, Level: "info"}}

	log, err := ProvideLogger(cfg, fxtest.NewLifecycle(t))
	if err != nil {
		t.Fatal(err)
	}

	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if !log.Enabled(context.Background(), level) {
			t.Errorf("logger drops %v records: the service would start with no log output at all", level)
		}
	}
}

func TestProvideLoggerDefaultsToConsole(t *testing.T) {
	log, err := ProvideLogger(&config.Config{Log: appconfig.Log{}}, fxtest.NewLifecycle(t))
	if err != nil {
		t.Fatal(err)
	}

	if !log.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("an empty log configuration produced a silent logger")
	}
}

func TestProvideLoggerHonoursLevel(t *testing.T) {
	cfg := &config.Config{Log: appconfig.Log{Console: true, Level: "error"}}

	log, err := ProvideLogger(cfg, fxtest.NewLifecycle(t))
	if err != nil {
		t.Fatal(err)
	}

	if log.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("level=error still admits debug records")
	}

	if !log.Enabled(context.Background(), slog.LevelError) {
		t.Error("level=error rejects error records")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{" warn ", slog.LevelWarn},
		{"Error", slog.LevelError},
		{"", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		if got := parseLevel(tt.in); got != tt.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// recorder counts what reaches a handler.
type recorder struct {
	slog.Handler

	records []slog.Record
	attrs   []slog.Attr
}

func (r *recorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *recorder) Handle(_ context.Context, rec slog.Record) error {
	r.records = append(r.records, rec)

	return nil
}

func (r *recorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &recorder{attrs: append(r.attrs, attrs...)}
}

func (r *recorder) WithGroup(string) slog.Handler { return r }

// otelslog carries no level of its own, so without this wrapper log.level could
// not stop every request's debug record from being exported.
func TestLevelHandler(t *testing.T) {
	inner := &recorder{}
	h := levelHandler{Handler: inner, level: slog.LevelWarn}
	ctx := context.Background()

	if h.Enabled(ctx, slog.LevelDebug) || h.Enabled(ctx, slog.LevelInfo) {
		t.Error("records below the configured level are still admitted")
	}

	if !h.Enabled(ctx, slog.LevelWarn) || !h.Enabled(ctx, slog.LevelError) {
		t.Error("records at or above the configured level are rejected")
	}

	// The bound must survive the derivations slog makes internally.
	if got := h.WithAttrs([]slog.Attr{slog.String("k", "v")}); got.Enabled(ctx, slog.LevelDebug) {
		t.Error("WithAttrs dropped the level bound")
	}

	if got := h.WithGroup("g"); got.Enabled(ctx, slog.LevelDebug) {
		t.Error("WithGroup dropped the level bound")
	}
}

func TestMultiHandler(t *testing.T) {
	first, second := &recorder{}, &recorder{}
	log := slog.New(multiHandler{handlers: []slog.Handler{first, second}})

	log.Info("hello")

	if len(first.records) != 1 || len(second.records) != 1 {
		t.Fatalf("record reached %d and %d handlers, want one each", len(first.records), len(second.records))
	}

	// One handler declining must not silence the others.
	quiet := levelHandler{Handler: &recorder{}, level: slog.LevelError}
	third := &recorder{}
	log = slog.New(multiHandler{handlers: []slog.Handler{quiet, third}})

	log.Info("hello")

	if len(third.records) != 1 {
		t.Error("a handler that declined the record silenced the rest")
	}
}
