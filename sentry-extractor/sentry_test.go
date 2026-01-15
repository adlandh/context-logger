package sentryextractor

import (
	"context"
	"sync"
	"testing"
	"time"

	ctxLogger "github.com/adlandh/context-logger"
	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

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
	core, observed := observer.New(zap.InfoLevel)
	l := zap.New(core).With(
		zap.String("text", "test"),
	)

	transport := &transportMock{}

	t.Run("test context logger with sentry extractor with no tracer", func(t *testing.T) {
		observed.TakeAll()
		message := "sentry-message-1"
		ctx := context.Background()
		transaction := sentry.TransactionFromContext(ctx)
		require.Nil(t, transaction)
		logger := ctxLogger.WithContext(l, With())
		logger.Ctx(ctx).Info(message)

		entries := observed.TakeAll()
		require.Len(t, entries, 1)
		entry := entries[0]
		fields := entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		_, ok := fields["trace_id"]
		require.False(t, ok)
		_, ok = fields["span_id"]
		require.False(t, ok)
		_, ok = fields["span_status"]
		require.False(t, ok)
	})

	t.Run("test context logger with sentry extractor with tracer", func(t *testing.T) {
		observed.TakeAll()
		message := "sentry-message-2"
		err := sentry.Init(sentry.ClientOptions{
			Transport:   transport,
			Environment: "test",
		})

		require.NoError(t, err)
		spanName := "test-span"

		rootspan := sentry.StartSpan(context.Background(), spanName+"_root")
		ctx := rootspan.Context()
		defer rootspan.Finish()

		span := sentry.StartSpan(ctx, spanName)
		ctx = span.Context()
		defer span.Finish()

		logger := ctxLogger.WithContext(l, With())
		logger.Ctx(ctx).Info(message)

		entries := observed.TakeAll()
		require.Len(t, entries, 1)
		entry := entries[0]
		fields := entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		require.Equal(t, rootspan.TraceID.String(), fields["trace_id"])
		require.Equal(t, span.SpanID.String(), fields["span_id"])
		require.Equal(t, span.Status.String(), fields["span_status"])
		require.Equal(t, span.Op, fields["span_op"])
		require.NotEqual(t, rootspan.SpanID.String(), fields["span_id"])
		require.NotEqual(t, rootspan.Op, fields["span_op"])
	})
}
