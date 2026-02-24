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

func (*transportMock) Configure(_ sentry.ClientOptions) {}

func (t *transportMock) SendEvent(event *sentry.Event) {
	t.Lock()
	defer t.Unlock()
	t.events = append(t.events, event)
}

func (*transportMock) Flush(_ time.Duration) bool {
	return true
}

func (t *transportMock) FlushWithContext(_ context.Context) bool {
	return t.Flush(0)
}

func (t *transportMock) Events() []*sentry.Event {
	t.Lock()
	defer t.Unlock()
	return t.events
}

func (*transportMock) Close() {}

type testContextKey string

func (k testContextKey) String() string {
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

func setUpSentry(t *testing.T) *transportMock {
	transport := &transportMock{}
	err := sentry.Init(sentry.ClientOptions{
		Transport:   transport,
		Environment: "test",
	})
	require.NoError(t, err)
	t.Cleanup(func() { sentry.Flush(time.Second) })
	return transport
}

func TestSentryExtractor_NoSpan(t *testing.T) {
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
		_, ok = fields[FieldSpanStatus]
		require.False(t, ok)
		_, ok = fields[FieldSpanOp]
		require.False(t, ok)
	})

	t.Run("empty hub has no span", func(t *testing.T) {
		observed.TakeAll()
		require.Nil(t, sentry.TransactionFromContext(context.Background()))
		fields := logAndAssert(t, context.Background(), observed, cl, "no-transaction")

		require.Equal(t, "test", fields["text"])
		_, ok := fields[FieldTraceID]
		require.False(t, ok)
	})
}

func TestSentryExtractor_WithSpan(t *testing.T) {
	_ = setUpSentry(t)
	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With())

	t.Run("extracts span info from root span", func(t *testing.T) {
		observed.TakeAll()
		rootSpan := sentry.StartSpan(context.Background(), "root-operation")
		defer rootSpan.Finish()

		fields := logAndAssert(t, rootSpan.Context(), observed, cl, "root-span")

		require.Equal(t, rootSpan.TraceID.String(), fields[FieldTraceID])
		require.Equal(t, rootSpan.SpanID.String(), fields[FieldSpanID])
		require.Equal(t, rootSpan.Status.String(), fields[FieldSpanStatus])
		require.Equal(t, rootSpan.Op, fields[FieldSpanOp])
	})

	t.Run("extracts span info from child span", func(t *testing.T) {
		observed.TakeAll()
		rootSpan := sentry.StartSpan(context.Background(), "root-span")
		defer rootSpan.Finish()

		childSpan := sentry.StartSpan(rootSpan.Context(), "child-operation")
		defer childSpan.Finish()

		fields := logAndAssert(t, childSpan.Context(), observed, cl, "child-span")

		require.Equal(t, rootSpan.TraceID.String(), fields[FieldTraceID])
		require.Equal(t, childSpan.SpanID.String(), fields[FieldSpanID])
		require.Equal(t, childSpan.Status.String(), fields[FieldSpanStatus])
		require.Equal(t, childSpan.Op, fields[FieldSpanOp])
		require.NotEqual(t, rootSpan.SpanID.String(), fields[FieldSpanID])
	})

	t.Run("preserves logger fields", func(t *testing.T) {
		observed.TakeAll()
		span := sentry.StartSpan(context.Background(), "test-span")
		defer span.Finish()

		fields := logAndAssert(t, span.Context(), observed, cl, "preserve-fields")

		require.Equal(t, "test", fields["text"])
		require.NotEmpty(t, fields[FieldTraceID])
		require.NotEmpty(t, fields[FieldSpanID])
	})
}

func TestSentryExtractor_MultipleSpans(t *testing.T) {
	_ = setUpSentry(t)
	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With())

	t.Run("nested spans have different span IDs", func(t *testing.T) {
		observed.TakeAll()
		root := sentry.StartSpan(context.Background(), "root")
		defer root.Finish()

		child1 := sentry.StartSpan(root.Context(), "child1")
		fields1 := logAndAssert(t, child1.Context(), observed, cl, "child1-log")
		child1.Finish()

		child2 := sentry.StartSpan(root.Context(), "child2")
		fields2 := logAndAssert(t, child2.Context(), observed, cl, "child2-log")
		child2.Finish()

		require.Equal(t, fields1["trace_id"], fields2["trace_id"])
		require.NotEqual(t, fields1["span_id"], fields2["span_id"])
	})
}

func TestSentryExtractor_CombinedWithOtherExtractors(t *testing.T) {
	_ = setUpSentry(t)
	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With(), ctxLogger.WithValueExtractor[testContextKey](testContextKey("request_id")))

	t.Run("works with value extractor", func(t *testing.T) {
		observed.TakeAll()
		span := sentry.StartSpan(context.Background(), "sentry-span")
		defer span.Finish()

		ctx := context.WithValue(span.Context(), testContextKey("request_id"), "req-456")
		fields := logAndAssert(t, ctx, observed, cl, "combined")

		require.Equal(t, "req-456", fields["request_id"])
		require.Equal(t, span.TraceID.String(), fields[FieldTraceID])
		require.Equal(t, span.SpanID.String(), fields[FieldSpanID])
	})
}

func TestSentryExtractor_SpanStatus(t *testing.T) {
	_ = setUpSentry(t)
	logger, observed := newTestLogger()
	cl := ctxLogger.WithContext(logger, With())

	t.Run("status can be set to error", func(t *testing.T) {
		observed.TakeAll()
		span := sentry.StartSpan(context.Background(), "error-span")
		span.Status = sentry.SpanStatusInternalError
		defer span.Finish()

		fields := logAndAssert(t, span.Context(), observed, cl, "error-check")

		require.Equal(t, sentry.SpanStatusInternalError.String(), fields[FieldSpanStatus])
	})
}
