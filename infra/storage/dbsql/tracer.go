package dbsql

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/webitel/webitel-go-kit/infra/pgw"
)

const scopeName = "github.com/webitel/webitel-wfm/infra/storage/dbsql"

// Tracer records a client span per query under the caller's span.
type Tracer struct {
	tracer trace.Tracer
}

var _ pgw.Tracer = (*Tracer)(nil)

func NewTracer() *Tracer {
	return &Tracer{tracer: otel.Tracer(scopeName)}
}

func (t *Tracer) ShouldTrace(ctx context.Context) bool {
	return trace.SpanFromContext(ctx).IsRecording()
}

//nolint:spancheck // the span ends in the callback handed back to pgw
func (t *Tracer) StartTrace(ctx context.Context, method, sql string, _ []any) (context.Context, func(error)) {
	ctx, span := t.tracer.Start(ctx, spanName(method, sql),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("db.system.name", "postgresql"), attribute.String("db.query.text", sql)),
	)

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}
}

// spanName names the span after the statement instead of the bare pgw method,
// which otherwise makes every database span in a trace read "db.Query". Only
// the leading keyword and the table are used, to keep the name low cardinality.
func spanName(method, sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return method
	}

	verb := strings.ToUpper(fields[0])
	if verb == "UPDATE" && len(fields) > 1 {
		return verb + " " + table(fields[1])
	}

	for i := 1; i < len(fields)-1; i++ {
		switch strings.ToUpper(fields[i]) {
		case "FROM", "INTO":
			return verb + " " + table(fields[i+1])
		}
	}

	return verb
}

func table(field string) string {
	return strings.Trim(field, `"(,;`)
}
