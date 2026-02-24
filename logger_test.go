package contextlogger

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const testContextKey = "context-logger-context"

type contextKeyInt int

func (k contextKeyInt) String() string {
	return fmt.Sprintf("%d", k)
}

type contextKeyString string

func (k contextKeyString) String() string {
	return string(k)
}

type contextKeyStruct struct {
	V float32
}

func (k contextKeyStruct) String() string {
	return "contextKeyStruct"
}

func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, observed := observer.New(zap.InfoLevel)
	return zap.New(core).With(zap.String("text", "test")), observed
}

func logAndAssert(
	t *testing.T,
	observed *observer.ObservedLogs,
	logger *ContextLogger,
	ctx context.Context,
	msg string,
) map[string]interface{} {
	t.Helper()
	logger.Ctx(ctx).Info(msg)
	entries := observed.TakeAll()
	require.Len(t, entries, 1)
	require.Equal(t, msg, entries[0].Message)
	return entries[0].ContextMap()
}

func TestContextLogger_NoExtractors(t *testing.T) {
	logger, observed := newTestLogger()

	t.Run("logs without context values", func(t *testing.T) {
		observed.TakeAll()
		cl := WithContext(logger)
		fields := logAndAssert(t, observed, cl, context.Background(), "test-message")

		require.Equal(t, "test", fields["text"])
		_, ok := fields["context_key"]
		require.False(t, ok)
	})

	t.Run("handles nil context", func(t *testing.T) {
		observed.TakeAll()
		cl := WithContext(logger)
		fields := logAndAssert(t, observed, cl, nil, "nil-context-message")

		require.Equal(t, "test", fields["text"])
	})

	t.Run("ignores context values without extractors", func(t *testing.T) {
		observed.TakeAll()
		cl := WithContext(logger)
		key := contextKeyString("test-key")
		ctx := context.WithValue(context.Background(), key, "test-value")
		fields := logAndAssert(t, observed, cl, ctx, "ignored-message")

		_, ok := fields[key.String()]
		require.False(t, ok)
	})
}

func TestContextLogger_WithValueExtractor(t *testing.T) {
	logger, observed := newTestLogger()

	t.Run("extracts string key", func(t *testing.T) {
		observed.TakeAll()
		key := contextKeyString("user_id")
		val := "user-123"
		cl := WithContext(logger, WithValueExtractor(key))
		ctx := context.WithValue(context.Background(), key, val)
		fields := logAndAssert(t, observed, cl, ctx, "user-log")

		require.Equal(t, val, fields[key.String()])
	})

	t.Run("extracts multiple keys", func(t *testing.T) {
		observed.TakeAll()
		key1 := contextKeyString("user_id")
		key2 := contextKeyInt(1)
		key3 := contextKeyStruct{V: 1.0}
		val1 := "user-456"
		val2 := uint8(99)
		val3 := 1.9

		cl := WithContext(logger, WithValueExtractor(key1), WithValueExtractor(key2), WithValueExtractor(key3))
		ctx := context.WithValue(context.Background(), key1, val1)
		ctx = context.WithValue(ctx, key2, val2)
		ctx = context.WithValue(ctx, key3, val3)
		fields := logAndAssert(t, observed, cl, ctx, "multi-key-log")

		require.Equal(t, val1, fields[key1.String()])
		require.Equal(t, val2, fields[key2.String()])
		require.Equal(t, val3, fields[key3.String()])
	})

	t.Run("skips missing keys", func(t *testing.T) {
		observed.TakeAll()
		key := contextKeyString("missing")
		cl := WithContext(logger, WithValueExtractor(key))
		fields := logAndAssert(t, observed, cl, context.Background(), "missing-key")

		_, ok := fields[key.String()]
		require.False(t, ok)
	})

	t.Run("skips nil values", func(t *testing.T) {
		observed.TakeAll()
		key := contextKeyString("nil-value")
		cl := WithContext(logger, WithValueExtractor(key))
		ctx := context.WithValue(context.Background(), key, nil)
		fields := logAndAssert(t, observed, cl, ctx, "nil-value-log")

		_, ok := fields[key.String()]
		require.False(t, ok)
	})

	t.Run("handles empty key slice", func(t *testing.T) {
		observed.TakeAll()
		cl := WithContext(logger, WithValueExtractor[contextKeyString]())
		fields := logAndAssert(t, observed, cl, context.Background(), "empty-keys")

		require.Equal(t, "test", fields["text"])
	})

	t.Run("preserves logger fields", func(t *testing.T) {
		observed.TakeAll()
		key := contextKeyString("trace_id")
		cl := WithContext(logger, WithValueExtractor(key))
		ctx := context.WithValue(context.Background(), key, "abc-123")
		fields := logAndAssert(t, observed, cl, ctx, "preserved")

		require.Equal(t, "test", fields["text"])
		require.Equal(t, "abc-123", fields["trace_id"])
	})
}

func TestContextLogger_WithContextCarrier(t *testing.T) {
	logger, observed := newTestLogger()

	t.Run("adds context as carrier field", func(t *testing.T) {
		cl := WithContext(logger, WithContextCarrier(testContextKey))
		ctx := context.Background()
		cl.Ctx(ctx).Info("carrier-test")

		entries := observed.TakeAll()
		require.Len(t, entries, 1)

		observedCtx := entries[0].Context
		require.NotNil(t, observedCtx)
	})

	t.Run("skips empty field name", func(t *testing.T) {
		cl := WithContext(logger, WithContextCarrier(""))
		fields := logAndAssert(t, observed, cl, context.Background(), "empty-name")

		_, ok := fields[""]
		require.False(t, ok)
	})

	t.Run("works with value extractor", func(t *testing.T) {
		key := contextKeyString("request_id")
		cl := WithContext(logger, WithContextCarrier(testContextKey), WithValueExtractor(key))
		ctx := context.WithValue(context.Background(), key, "req-789")

		observed.TakeAll()
		cl.Ctx(ctx).Info("combined")
		entries := observed.TakeAll()

		require.Len(t, entries, 1)
		fields := entries[0].ContextMap()
		require.Equal(t, "req-789", fields[key.String()])
	})
}

func TestContextLogger_WithDeadlineExtractor(t *testing.T) {
	logger, observed := newTestLogger()

	t.Run("no deadline returns no fields", func(t *testing.T) {
		observed.TakeAll()
		cl := WithContext(logger, WithDeadlineExtractor())
		fields := logAndAssert(t, observed, cl, context.Background(), "no-deadline")

		_, ok := fields["context_deadline_at"]
		require.False(t, ok)
	})

	t.Run("deadline not reached", func(t *testing.T) {
		observed.TakeAll()
		deadline := time.Now().Add(2 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		cl := WithContext(logger, WithDeadlineExtractor())
		fields := logAndAssert(t, observed, cl, ctx, "future-deadline")

		deadlineAt, ok := fields["context_deadline_at"].(time.Time)
		require.True(t, ok)
		require.WithinDuration(t, deadline, deadlineAt, 50*time.Millisecond)

		timeLeft := extractDuration(t, fields["context_time_left"])
		require.Greater(t, timeLeft, time.Duration(0))
		require.LessOrEqual(t, timeLeft, 2*time.Second)
	})

	t.Run("deadline reached", func(t *testing.T) {
		observed.TakeAll()
		deadline := time.Now().Add(-2 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		cl := WithContext(logger, WithDeadlineExtractor())
		fields := logAndAssert(t, observed, cl, ctx, "past-deadline")

		deadlineAt, ok := fields["context_deadline_at"].(time.Time)
		require.True(t, ok)
		require.WithinDuration(t, deadline, deadlineAt, 50*time.Millisecond)

		timeLeft := extractDuration(t, fields["context_time_left"])
		require.Less(t, timeLeft, time.Duration(0))

		require.Equal(t, "context deadline exceeded", fields["context_error"])
	})

	t.Run("canceled context", func(t *testing.T) {
		observed.TakeAll()
		deadline := time.Now().Add(2 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		cancel()

		cl := WithContext(logger, WithDeadlineExtractor())
		fields := logAndAssert(t, observed, cl, ctx, "canceled")

		deadlineAt, ok := fields["context_deadline_at"].(time.Time)
		require.True(t, ok)
		require.WithinDuration(t, deadline, deadlineAt, 50*time.Millisecond)

		require.Equal(t, "context canceled", fields["context_error"])
	})
}

func TestContextLogger_CombinedExtractors(t *testing.T) {
	logger, observed := newTestLogger()

	key1 := contextKeyString("user_id")
	key2 := contextKeyInt(1)

	cl := WithContext(
		logger,
		WithValueExtractor(key1),
		WithValueExtractor(key2),
		WithContextCarrier(testContextKey),
		WithDeadlineExtractor(),
	)

	t.Run("all extractors work together", func(t *testing.T) {
		deadline := time.Now().Add(5 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		ctx = context.WithValue(ctx, key1, "user-999")
		ctx = context.WithValue(ctx, key2, uint16(42))

		observed.TakeAll()
		cl.Ctx(ctx).Info("combined-test")
		entries := observed.TakeAll()
		require.Len(t, entries, 1)

		fields := entries[0].ContextMap()

		require.Equal(t, "user-999", fields[key1.String()])
		require.Equal(t, uint16(42), fields[key2.String()])
		require.NotNil(t, fields["context_deadline_at"])
		require.NotNil(t, fields["context_time_left"])
	})
}

func TestContextLogger_NilContext(t *testing.T) {
	logger, observed := newTestLogger()

	t.Run("nil context defaults to background", func(t *testing.T) {
		observed.TakeAll()
		cl := WithContext(logger, WithValueExtractor(contextKeyString("key")))
		cl.Ctx(nil).Info("nil-context")

		entries := observed.TakeAll()
		require.Len(t, entries, 1)
		require.Equal(t, "nil-context", entries[0].Message)
	})
}

func extractDuration(t *testing.T, v interface{}) time.Duration {
	t.Helper()
	switch d := v.(type) {
	case time.Duration:
		return d
	case int64:
		return time.Duration(d)
	default:
		t.Fatalf("unexpected type for duration: %T", v)
		return 0
	}
}

func TestContextLogger_New(t *testing.T) {
	logger, observed := newTestLogger()

	t.Run("creates logger with extractors", func(t *testing.T) {
		observed.TakeAll()
		key := contextKeyString("user_id")
		cl := New(logger, WithValueExtractor(key))
		ctx := context.WithValue(context.Background(), key, "user-123")
		fields := logAndAssert(t, observed, cl, ctx, "new-test")

		require.Equal(t, "user-123", fields[key.String()])
	})

	t.Run("creates logger without extractors", func(t *testing.T) {
		observed.TakeAll()
		cl := New(logger)
		fields := logAndAssert(t, observed, cl, context.Background(), "no-extractors")

		require.Equal(t, "test", fields["text"])
	})
}

func TestContextLogger_WithMethod(t *testing.T) {
	logger, observed := newTestLogger()

	t.Run("adds extractors to existing logger", func(t *testing.T) {
		observed.TakeAll()
		key1 := contextKeyString("user_id")
		key2 := contextKeyString("request_id")

		cl := WithContext(logger, WithValueExtractor(key1))
		cl = cl.With(WithValueExtractor(key2))

		ctx := context.WithValue(context.Background(), key1, "user-456")
		ctx = context.WithValue(ctx, key2, "req-789")

		fields := logAndAssert(t, observed, cl, ctx, "with-test")

		require.Equal(t, "user-456", fields[key1.String()])
		require.Equal(t, "req-789", fields[key2.String()])
	})

	t.Run("does not modify original logger", func(t *testing.T) {
		observed.TakeAll()
		key := contextKeyString("new_key")

		cl := WithContext(logger)
		cl.With(WithValueExtractor(key))

		ctx := context.WithValue(context.Background(), key, "value")
		fields := logAndAssert(t, observed, cl, ctx, "original-unchanged")

		_, ok := fields[key.String()]
		require.False(t, ok)
	})
}

func TestContextLogger_Logger(t *testing.T) {
	logger, _ := newTestLogger()

	t.Run("returns underlying zap logger", func(t *testing.T) {
		cl := WithContext(logger, WithValueExtractor(contextKeyString("key")))
		returned := cl.Logger()

		require.Same(t, logger, returned)
	})
}
