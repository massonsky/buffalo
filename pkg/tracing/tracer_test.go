package tracing

import (
	"context"
	"errors"
	"testing"
)

func TestNoopTracer_DefaultIsNoop(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "noop")
	span.SetAttribute("k", "v")
	span.RecordError(errors.New("boom"))
	span.SetStatus(StatusError, "fail")
	span.End()
	span.End() // idempotent
}

func TestMemoryTracer_RecordsSpan(t *testing.T) {
	mt := NewMemoryTracer()
	prev := GlobalTracer()
	SetTracer(mt)
	t.Cleanup(func() { SetTracer(prev) })

	ctx := context.Background()
	_, sp := StartSpan(ctx, "build.compile", WithAttributes(map[string]any{"lang": "go"}))
	sp.SetAttribute("file", "x.proto")
	sp.SetStatus(StatusOK, "")
	sp.End()
	sp.End() // second call is no-op

	spans := mt.Spans()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	got := spans[0]
	if got.Name != "build.compile" {
		t.Errorf("name=%q", got.Name)
	}
	if got.Attributes["lang"] != "go" || got.Attributes["file"] != "x.proto" {
		t.Errorf("attrs=%v", got.Attributes)
	}
	if got.StatusCode != StatusOK {
		t.Errorf("status=%d", got.StatusCode)
	}
	if got.Duration() < 0 {
		t.Errorf("negative duration")
	}
}

func TestSetTracer_NilResetsToNoop(t *testing.T) {
	mt := NewMemoryTracer()
	SetTracer(mt)
	SetTracer(nil)
	_, sp := StartSpan(context.Background(), "x")
	sp.End()
	if len(mt.Spans()) != 0 {
		t.Errorf("memory tracer should not have received spans after reset")
	}
}
