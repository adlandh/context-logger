package otelextractor

import (
	"context"
	"testing"

	ctxLogger "github.com/adlandh/context-logger"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type contextKey string

func (k contextKey) String() string {
	return string(k)
}

func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, observed := observer.New(zap.InfoLevel)
	return zap.New(core).With(zap.String("text", "test")), observed
}

func logAndAssert(
	t *testing.T,
	ctx context.Context,
	observed *observer.ObservedLogs,
	logger *ctxLogger.ContextLogger,
	msg string,
) map[string]interface{} {
	t.Helper()
	logger.Ctx(ctx).Info(msg)
	entries := observed.TakeAll()
	require.Len(t, entries, 1)
	require.Equal(t, msg, entries[0].Message)
	return entries[0].ContextMap()
}

func createSpanContext(traceID, spanID []byte) trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID(traceID),
		SpanID:  trace.SpanID(spanID),
	})
}

func setUpPropagator(t *testing.T) {
	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(prev) })
}

func TestOtelExtractor_NoSpanContext(t *testing.T) {
	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With())

	t.Run("background context has no span", func(t *testing.T) {
		observed.TakeAll()
		fields := logAndAssert(t, context.Background(), observed, cl, "no-span")

		require.Equal(t, "test", fields["text"])
		_, ok := fields[FieldTraceID]
		require.False(t, ok)
		_, ok = fields[FieldSpanID]
		require.False(t, ok)
	})

	t.Run("canceled context has no span", func(t *testing.T) {
		observed.TakeAll()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		fields := logAndAssert(t, ctx, observed, cl, "canceled")

		require.Equal(t, "test", fields["text"])
		_, ok := fields[FieldTraceID]
		require.False(t, ok)
	})
}

func TestOtelExtractor_WithTracerProvider(t *testing.T) {
	provider := noop.NewTracerProvider()
	setUpPropagator(t)

	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With())

	t.Run("extracts trace and span IDs from active span", func(t *testing.T) {
		observed.TakeAll()
		sc := createSpanContext(
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x01},
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		)
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
		ctx, span := provider.Tracer("test-tracer").Start(ctx, "test-span")

		fields := logAndAssert(t, ctx, observed, cl, "active-span")

		require.Equal(t, span.SpanContext().TraceID().String(), fields[FieldTraceID])
		require.Equal(t, span.SpanContext().SpanID().String(), fields[FieldSpanID])
	})

	t.Run("extracts from remote span context only", func(t *testing.T) {
		observed.TakeAll()
		sc := createSpanContext(
			[]byte{0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19},
			[]byte{0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11},
		)
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

		fields := logAndAssert(t, ctx, observed, cl, "remote-span")

		require.Equal(t, sc.TraceID().String(), fields[FieldTraceID])
		require.Equal(t, sc.SpanID().String(), fields[FieldSpanID])
	})

	t.Run("preserves logger fields", func(t *testing.T) {
		observed.TakeAll()
		sc := createSpanContext(
			[]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
			[]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		)
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

		fields := logAndAssert(t, ctx, observed, cl, "preserve-fields")

		require.Equal(t, "test", fields["text"])
		require.NotEmpty(t, fields[FieldTraceID])
		require.NotEmpty(t, fields[FieldSpanID])
	})
}

func TestOtelExtractor_InvalidSpanContext(t *testing.T) {
	provider := noop.NewTracerProvider()
	setUpPropagator(t)

	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With())

	t.Run("zero trace ID is invalid", func(t *testing.T) {
		observed.TakeAll()
		sc := createSpanContext(
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		)
		require.False(t, sc.IsValid())
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

		fields := logAndAssert(t, ctx, observed, cl, "invalid-trace")

		_, ok := fields[FieldTraceID]
		require.False(t, ok)
		_, ok = fields[FieldSpanID]
		require.False(t, ok)
	})

	t.Run("zero span ID is invalid", func(t *testing.T) {
		observed.TakeAll()
		sc := createSpanContext(
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		)
		require.False(t, sc.IsValid())
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

		fields := logAndAssert(t, ctx, observed, cl, "invalid-span")

		_, ok := fields[FieldTraceID]
		require.False(t, ok)
		_, ok = fields[FieldSpanID]
		require.False(t, ok)
	})

	t.Run("span from noop provider has no valid context", func(t *testing.T) {
		observed.TakeAll()
		_, span := provider.Tracer("test").Start(context.Background(), "noop-span")
		defer span.End()

		sc := span.SpanContext()
		require.False(t, sc.IsValid())

		fields := logAndAssert(t, context.Background(), observed, cl, "noop-span")

		_, ok := fields[FieldTraceID]
		require.False(t, ok)
	})
}

func TestOtelExtractor_CombinedWithOtherExtractors(t *testing.T) {
	setUpPropagator(t)

	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With(), ctxLogger.WithValueExtractor[contextKey](contextKey("request_id")))

	t.Run("works with value extractor", func(t *testing.T) {
		observed.TakeAll()
		sc := createSpanContext(
			[]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
			[]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
		)
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
		ctx = context.WithValue(ctx, contextKey("request_id"), "req-123")

		fields := logAndAssert(t, ctx, observed, cl, "combined")

		require.Equal(t, "req-123", fields["request_id"])
		require.Equal(t, sc.TraceID().String(), fields[FieldTraceID])
		require.Equal(t, sc.SpanID().String(), fields[FieldSpanID])
	})
}
