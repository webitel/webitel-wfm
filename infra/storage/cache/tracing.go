package cache

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/webitel/webitel-wfm/infra/storage/cache"

const (
	Key = attribute.Key("cache.key")
)

type Tracer struct {
	tracer trace.Tracer
	attrs  []attribute.KeyValue
}

func NewTracer() *Tracer {
	return &Tracer{
		tracer: otel.Tracer(tracerName, trace.WithInstrumentationVersion("1.0.0")),
		attrs: []attribute.KeyValue{
			semconv.DBSystemCache,
		},
	}
}

func (t *Tracer) Start(ctx context.Context, op, k string) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
		trace.WithAttributes(Key.String(k)),
	}

	ctx, _ = t.tracer.Start(ctx, op, opts...)

	return ctx
}

func (t *Tracer) End(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}
