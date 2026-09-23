package dbsql

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracer(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))

	tracer := NewTracer()

	if tracer.ShouldTrace(context.Background()) {
		t.Fatal("a query outside any span must not open a root span")
	}

	ctx, parent := otel.Tracer("test").Start(context.Background(), "request")
	if !tracer.ShouldTrace(ctx) {
		t.Fatal("a query under a recording span must be traced")
	}

	_, end := tracer.StartTrace(ctx, "db.Query", "SELECT id FROM wfm.working_schedule WHERE domain_id = $1", nil)
	end(errors.New("boom"))
	parent.End()

	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("got %d spans, want 2", len(spans))
	}

	query := spans[0]
	if query.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Errorf("query span %q is not a child of the request span", query.Name())
	}

	// Named after the statement, not the pgw method: otherwise every database
	// span in a trace reads "db.Query".
	if query.Name() != "SELECT wfm.working_schedule" {
		t.Errorf("query span name = %q, want %q", query.Name(), "SELECT wfm.working_schedule")
	}

	if query.Status().Code != codes.Error {
		t.Errorf("failed query span status = %v, want Error", query.Status().Code)
	}

	var sawSQL bool

	for _, attr := range query.Attributes() {
		if attr.Key == "db.query.text" && attr.Value.AsString() == "SELECT id FROM wfm.working_schedule WHERE domain_id = $1" {
			sawSQL = true
		}
	}

	if !sawSQL {
		t.Error("query span carries no db.query.text attribute")
	}
}
