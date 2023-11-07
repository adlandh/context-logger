package contextlogger

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type MemorySink struct {
	*bytes.Buffer
}

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

func (s *MemorySink) Close() error { return nil }
func (s *MemorySink) Sync() error  { return nil }

func TestContextLogger(t *testing.T) {
	sink := &MemorySink{new(bytes.Buffer)}
	err := zap.RegisterSink("memory", func(*url.URL) (zap.Sink, error) {
		return sink, nil
	})
	require.NoError(t, err)

	conf := zap.NewProductionConfig()
	// Redirect all messages to the MemorySink.
	conf.OutputPaths = []string{"memory://"}

	l, err := conf.Build()
	require.NoError(t, err)

	t.Run("test context logger with no extractors", func(t *testing.T) {
		logger := WithContext(l)
		key := gofakeit.Word()
		val := gofakeit.SentenceSimple()
		ctx := context.WithValue(context.Background(), key, val)
		logger.Ctx(ctx).Info("test message")
		require.Contains(t, sink.String(), "test message")
		require.NotContains(t, sink.String(), key)
		require.NotContains(t, sink.String(), val)
	})

	t.Run("test context logger with value extractor", func(t *testing.T) {
		key1 := gofakeit.Word()
		key2 := gofakeit.Word()
		key3 := gofakeit.Word()

		val1 := gofakeit.SentenceSimple()
		val2 := gofakeit.Uint8()
		val3 := gofakeit.Bool()

		logger := WithContext(l, WithValueExtractor(key1, key2))

		ctx := context.WithValue(context.Background(), key1, val1)
		logger.Ctx(ctx).Info("first value")
		require.Contains(t, sink.String(), "first value")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", key1, val1))
		require.NotContains(t, sink.String(), key2)
		require.NotContains(t, sink.String(), key3)

		ctx = context.WithValue(ctx, key2, val2)
		logger.Ctx(ctx).Info("second value")
		require.Contains(t, sink.String(), "second value")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", key1, val1))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":%d", key2, val2))
		require.NotContains(t, sink.String(), key3)

		ctx = context.WithValue(ctx, key3, val3)
		logger.Ctx(ctx).Info("third value")
		require.Contains(t, sink.String(), "third value")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", key1, val1))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":%d", key2, val2))
		require.NotContains(t, sink.String(), key3)
	})

	t.Run("test context logger with otel extractor with no tracer", func(t *testing.T) {
		ctx := context.Background()
		logger := WithContext(l, WithOtelExtractor())
		logger.Ctx(ctx).Info("otel data")
		require.Contains(t, sink.String(), "otel data")
		require.NotContains(t, sink.String(), "trace_id")
		require.NotContains(t, sink.String(), "span_id")
	})

	t.Run("test context logger with otel extractor with tracer", func(t *testing.T) {
		tracerName := gofakeit.Word()
		spanName := gofakeit.Word()

		provider := trace.NewNoopTracerProvider()
		otel.SetTextMapPropagator(propagation.TraceContext{})
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x01},
			SpanID:  trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		})
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
		ctx, _ = provider.Tracer(tracerName).Start(ctx, spanName)
		logger := WithContext(l, WithOtelExtractor())
		logger.Ctx(ctx).Info("otel data")
		require.Contains(t, sink.String(), "otel data")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "trace_id", sc.TraceID().String()))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "span_id", sc.SpanID().String()))
	})
}
