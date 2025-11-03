package sentryextractor

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	ctxLogger "github.com/adlandh/context-logger"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testText = "\"text\":\"test\""

type memorySink struct {
	*bytes.Buffer
}

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

func (s *memorySink) Close() error { return nil }
func (s *memorySink) Sync() error  { return nil }

var _ sentry.Transport = (*transportMock)(nil)

type transportMock struct {
	sync.Mutex
	events []*sentry.Event
}

func (*transportMock) Configure(_ sentry.ClientOptions) { /* stub */ }
func (t *transportMock) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}
func (*transportMock) Flush(_ time.Duration) bool {
	return true
}
func (t *transportMock) FlushWithContext(ctx context.Context) bool {
	return t.Flush(0)
}
func (t *transportMock) Events() []*sentry.Event {
	return t.events
}
func (*transportMock) Close() { /* stub */ }

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

	t.Run("test context logger with sentry extractor with no tracer", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.Sentence()
		ctx := context.Background()
		transaction := sentry.TransactionFromContext(ctx)
		require.Nil(t, transaction)
		logger := ctxLogger.WithContext(l, With())
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), testText)
		require.NotContains(t, sink.String(), "trace_id")
		require.NotContains(t, sink.String(), "span_id")
		require.NotContains(t, sink.String(), "span_status")
		require.NotContains(t, sink.String(), ctxLogger.ContextKey)
	})

	t.Run("test context logger with sentry extractor with tracer", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.Sentence()
		err := sentry.Init(sentry.ClientOptions{
			Transport:   transport,
			Environment: "test",
		})

		require.NoError(t, err)
		spanName := gofakeit.Word()

		rootspan := sentry.StartSpan(context.Background(), spanName+"_root")
		ctx := rootspan.Context()
		defer rootspan.Finish()

		span := sentry.StartSpan(ctx, spanName)
		ctx = span.Context()
		defer span.Finish()

		logger := ctxLogger.WithContext(l, With())
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), testText)
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", "trace_id", rootspan.TraceID.String()))
		require.NotContains(t, sink.String(), fmt.Sprintf("%q:%q", "span_id", rootspan.SpanID.String()))
		require.NotContains(t, sink.String(), fmt.Sprintf("%q:%q", "span_op", rootspan.Op))
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", "trace_id", span.TraceID.String()))
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", "span_id", span.SpanID.String()))
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", "span_status", span.Status.String()))
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", "span_op", span.Op))
		require.NotContains(t, sink.String(), ctxLogger.ContextKey)
	})
}
