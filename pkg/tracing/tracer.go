// Package tracing provides a minimal, zero-dependency tracing API used by
// Buffalo to instrument long-running build steps (proto compilation, dep
// fetch, cache lookups, etc.). It is intentionally compatible with the
// OpenTelemetry semantic model — Span, attributes, status — so that a future
// adapter can forward spans to an OTel SDK without touching call sites.
//
// By default tracing is a no-op: the global tracer drops every span. Call
// SetTracer to install a real implementation (e.g. one bridging to
// go.opentelemetry.io/otel) when the dependency is acceptable for the build.
package tracing

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Tracer creates spans. Implementations must be safe for concurrent use.
type Tracer interface {
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// Span represents an in-flight unit of work.
type Span interface {
	SetAttribute(key string, value any)
	RecordError(err error)
	SetStatus(code StatusCode, description string)
	End()
}

// StatusCode mirrors the OpenTelemetry status semantics.
type StatusCode int

const (
	StatusUnset StatusCode = iota
	StatusOK
	StatusError
)

// SpanOption configures a span at creation time.
type SpanOption func(*SpanConfig)

// SpanConfig is the materialized set of SpanOptions.
type SpanConfig struct {
	Attributes map[string]any
}

// WithAttributes seeds the new span with key/value pairs.
func WithAttributes(attrs map[string]any) SpanOption {
	return func(c *SpanConfig) {
		if c.Attributes == nil {
			c.Attributes = make(map[string]any, len(attrs))
		}
		for k, v := range attrs {
			c.Attributes[k] = v
		}
	}
}

// --- global tracer plumbing -------------------------------------------------

type tracerHolder struct{ t Tracer }

var globalTracer atomic.Value // always holds tracerHolder

func init() {
	globalTracer.Store(tracerHolder{t: noopTracer{}})
}

// SetTracer installs t as the process-wide tracer. Passing nil resets to no-op.
func SetTracer(t Tracer) {
	if t == nil {
		globalTracer.Store(tracerHolder{t: noopTracer{}})
		return
	}
	globalTracer.Store(tracerHolder{t: t})
}

// GlobalTracer returns the currently installed tracer.
func GlobalTracer() Tracer { return globalTracer.Load().(tracerHolder).t }

// StartSpan is a convenience wrapper around GlobalTracer().StartSpan.
func StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return GlobalTracer().StartSpan(ctx, name, opts...)
}

// --- no-op implementation ---------------------------------------------------

type noopTracer struct{}

func (noopTracer) StartSpan(ctx context.Context, _ string, _ ...SpanOption) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) SetAttribute(string, any)     {}
func (noopSpan) RecordError(error)            {}
func (noopSpan) SetStatus(StatusCode, string) {}
func (noopSpan) End()                         {}

// --- in-memory implementation (for tests / debug) ---------------------------

// MemoryTracer records spans in memory. Safe for tests; not for production.
type MemoryTracer struct {
	mu    sync.Mutex
	spans []*RecordedSpan
}

// NewMemoryTracer returns a fresh in-memory tracer.
func NewMemoryTracer() *MemoryTracer { return &MemoryTracer{} }

// Spans returns a snapshot of recorded spans.
func (m *MemoryTracer) Spans() []*RecordedSpan {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*RecordedSpan, len(m.spans))
	copy(out, m.spans)
	return out
}

// StartSpan implements Tracer.
func (m *MemoryTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	cfg := SpanConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	rs := &RecordedSpan{
		Name:       name,
		StartTime:  time.Now(),
		Attributes: cfg.Attributes,
		tracer:     m,
	}
	if rs.Attributes == nil {
		rs.Attributes = map[string]any{}
	}
	return ctx, rs
}

// RecordedSpan is a span captured by MemoryTracer.
type RecordedSpan struct {
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]any
	Errors     []error
	StatusCode StatusCode
	StatusDesc string

	ended  bool
	mu     sync.Mutex
	tracer *MemoryTracer
}

// SetAttribute implements Span.
func (s *RecordedSpan) SetAttribute(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.Attributes[key] = value
}

// RecordError implements Span.
func (s *RecordedSpan) RecordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended || err == nil {
		return
	}
	s.Errors = append(s.Errors, err)
}

// SetStatus implements Span.
func (s *RecordedSpan) SetStatus(code StatusCode, desc string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.StatusCode = code
	s.StatusDesc = desc
}

// End implements Span.
func (s *RecordedSpan) End() {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	s.EndTime = time.Now()
	s.mu.Unlock()

	s.tracer.mu.Lock()
	s.tracer.spans = append(s.tracer.spans, s)
	s.tracer.mu.Unlock()
}

// Duration returns the wall-clock duration. Zero before End().
func (s *RecordedSpan) Duration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ended {
		return 0
	}
	return s.EndTime.Sub(s.StartTime)
}
