package contextlogger

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const contextKey = "context-logger-context"

type contextKeyInt int

func (k contextKeyInt) String() string {
	return fmt.Sprintf("%d", k)
}

type contextKeyString string

func (k contextKeyString) String() string {
	return string(k)
}

type contextKeyStruct struct{}

func (k contextKeyStruct) String() string {
	return "contextKeyStruct"
}

func TestContextLogger(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	l := zap.New(core).With(
		zap.String("text", "test"),
	)

	t.Run("test context logger with no extractors", func(t *testing.T) {
		observed.TakeAll()
		message := "test-message-1"
		logger := WithContext(l)
		key := contextKeyInt(7)
		val := "test-value-1"
		ctx := context.WithValue(context.Background(), key, val)
		logger.Ctx(ctx).Info(message)

		entries := observed.TakeAll()
		require.Len(t, entries, 1)
		entry := entries[0]
		fields := entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		_, ok := fields[key.String()]
		require.False(t, ok)
		_, ok = fields[contextKey]
		require.False(t, ok)
	})

	t.Run("test context logger with value extractor", func(t *testing.T) {
		observed.TakeAll()
		message := "test-message-2"
		key1 := contextKeyString("key-one")
		key2 := contextKeyInt(8)
		key3 := contextKeyStruct{}

		val1 := "value-one"
		val2 := uint8(42)
		val3 := true

		logger := WithContext(l, WithValueExtractor(key1), WithValueExtractor(key2), WithContextCarrier(contextKey))

		ctx := context.WithValue(context.Background(), key1, val1)
		logger.Ctx(ctx).Info(message)

		entries := observed.TakeAll()
		require.Len(t, entries, 1)
		entry := entries[0]
		fields := entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		require.Equal(t, val1, fields[key1.String()])
		_, ok := fields[key2.String()]
		require.False(t, ok)
		_, ok = fields[key3.String()]
		require.False(t, ok)
		_, ok = fields[contextKey]
		require.False(t, ok)

		message = "test-message-3"
		ctx = context.WithValue(ctx, key2, val2)
		logger.Ctx(ctx).Info(message)

		entries = observed.TakeAll()
		require.Len(t, entries, 1)
		entry = entries[0]
		fields = entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		require.Equal(t, val1, fields[key1.String()])
		require.Equal(t, val2, fields[key2.String()])
		_, ok = fields[key3.String()]
		require.False(t, ok)
		_, ok = fields[contextKey]
		require.False(t, ok)

		message = "test-message-4"
		ctx = context.WithValue(ctx, key3, val3)
		logger.Ctx(ctx).Info(message)

		entries = observed.TakeAll()
		require.Len(t, entries, 1)
		entry = entries[0]
		fields = entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		require.Equal(t, val1, fields[key1.String()])
		require.Equal(t, val2, fields[key2.String()])
		_, ok = fields[key3.String()]
		require.False(t, ok)
		_, ok = fields[contextKey]
		require.False(t, ok)
	})
}
