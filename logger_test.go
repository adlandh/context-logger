package contextlogger

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type contextKeyInt int

func (k contextKeyInt) String() string {
	return fmt.Sprintf("%d", k)
}

type contextKeyString string

func (k contextKeyString) String() string {
	return string(k)
}

type memorySink struct {
	*bytes.Buffer
}

type contextKeyStruct struct{}

func (k contextKeyStruct) String() string {
	return "contextKeyStruct"
}

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

func (s *memorySink) Close() error { return nil }
func (s *memorySink) Sync() error  { return nil }

type transportMock struct {
	sync.Mutex
	events []*sentry.Event
}

func (t *transportMock) Configure(_ sentry.ClientOptions) {}
func (t *transportMock) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}
func (t *transportMock) Flush(_ time.Duration) bool {
	return true
}
func (t *transportMock) Events() []*sentry.Event {
	return t.events
}

func TestContextLogger(t *testing.T) {
	sink := &memorySink{new(bytes.Buffer)}
	err := zap.RegisterSink("memory", func(*url.URL) (zap.Sink, error) {
		return sink, nil
	})
	require.NoError(t, err)

	conf := zap.NewProductionConfig()
	// Redirect all messages to the memorySink.
	conf.OutputPaths = []string{"memory://"}

	l, err := conf.Build()
	require.NoError(t, err)

	l = l.With(
		zap.String("text", "test"),
	)

	transport := &transportMock{}

	t.Run("test context logger with no extractors", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.SentenceSimple()
		logger := WithContext(l)
		key := contextKeyInt(gofakeit.Int8())
		val := gofakeit.SentenceSimple()
		ctx := context.WithValue(context.Background(), key, val)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.NotContains(t, sink.String(), key)
		require.NotContains(t, sink.String(), val)
	})

	t.Run("test context logger with value extractor", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.SentenceSimple()
		key1 := contextKeyString(gofakeit.Word())
		key2 := contextKeyInt(gofakeit.Int8())
		key3 := contextKeyStruct{}

		val1 := gofakeit.SentenceSimple()
		val2 := gofakeit.Uint8()
		val3 := gofakeit.Bool()

		logger := WithContext(l, WithValueExtractor(key1, key2))

		ctx := context.WithValue(context.Background(), key1, val1)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", key1, val1))
		require.NotContains(t, sink.String(), key2)
		require.NotContains(t, sink.String(), key3)

		message = gofakeit.SentenceSimple()
		ctx = context.WithValue(ctx, key2, val2)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", key1, val1))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":%d", key2, val2))
		require.NotContains(t, sink.String(), key3)

		message = gofakeit.SentenceSimple()
		ctx = context.WithValue(ctx, key3, val3)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", key1, val1))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":%d", key2, val2))
		require.NotContains(t, sink.String(), key3)
	})

	t.Run("test context logger with otel extractor with no tracer", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.SentenceSimple()
		ctx := context.Background()
		spanContext := trace.SpanContextFromContext(ctx)
		require.False(t, spanContext.IsValid())
		logger := WithContext(l, WithOtelExtractor())
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.NotContains(t, sink.String(), "trace_id")
		require.NotContains(t, sink.String(), "span_id")
	})

	t.Run("test context logger with otel extractor with tracer", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.SentenceSimple()
		tracerName := gofakeit.Word()
		spanName := gofakeit.Word()

		provider := noop.NewTracerProvider()
		otel.SetTextMapPropagator(propagation.TraceContext{})
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x01},
			SpanID:  trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		})
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
		ctx, _ = provider.Tracer(tracerName).Start(ctx, spanName)
		logger := WithContext(l, WithOtelExtractor())
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "trace_id", sc.TraceID().String()))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "span_id", sc.SpanID().String()))
	})

	t.Run("test context logger with sentry extractor with no tracer", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.SentenceSimple()
		ctx := context.Background()
		transaction := sentry.TransactionFromContext(ctx)
		require.Nil(t, transaction)
		logger := WithContext(l, WithSentryExtractor())
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.NotContains(t, sink.String(), "trace_id")
		require.NotContains(t, sink.String(), "span_id")
		require.NotContains(t, sink.String(), "span_status")
	})

	t.Run("test context logger with sentry extractor with tracer", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.SentenceSimple()
		err := sentry.Init(sentry.ClientOptions{
			Transport:   transport,
			Environment: "test",
		})

		require.NoError(t, err)
		spanName := gofakeit.Word()

		span := sentry.StartSpan(context.Background(), spanName)
		ctx := span.Context()
		defer span.Finish()

		logger := WithContext(l, WithSentryExtractor())
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "trace_id", span.TraceID.String()))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "span_id", span.SpanID.String()))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "span_status", span.Status.String()))
	})

	t.Run("test context logger with sentry extractor with otel tracer", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.SentenceSimple()
		tracerName := gofakeit.Word()
		spanName := gofakeit.Word()

		provider := noop.NewTracerProvider()
		otel.SetTextMapPropagator(propagation.TraceContext{})
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x01},
			SpanID:  trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		})
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
		ctx, _ = provider.Tracer(tracerName).Start(ctx, spanName)
		transaction := sentry.TransactionFromContext(ctx)
		require.Nil(t, transaction)
		logger := WithContext(l, WithOtelExtractor(), WithSentryExtractor())
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), "\"text\":\"test\"")
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "trace_id", sc.TraceID().String()))
		require.Contains(t, sink.String(), fmt.Sprintf("\"%s\":\"%s\"", "span_id", sc.SpanID().String()))
		require.NotContains(t, sink.String(), "span_status")
	})

}
