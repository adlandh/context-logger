package contextlogger

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testText = "\"text\":\"test\""

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

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

type memorySink struct {
	*bytes.Buffer
}

func (s *memorySink) Close() error { return nil }
func (s *memorySink) Sync() error  { return nil }

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

	t.Run("test context logger with no extractors", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.Sentence()
		logger := WithContext(l)
		key := contextKeyInt(gofakeit.Int8())
		val := gofakeit.Sentence()
		ctx := context.WithValue(context.Background(), key, val)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), testText)
		require.NotContains(t, sink.String(), key)
		require.NotContains(t, sink.String(), val)
		require.NotContains(t, sink.String(), ContextKey)
	})

	t.Run("test context logger with value extractor", func(t *testing.T) {
		sink.Reset()
		message := gofakeit.Sentence()
		key1 := contextKeyString(gofakeit.Word())
		key2 := contextKeyInt(gofakeit.Int8())
		key3 := contextKeyStruct{}

		val1 := gofakeit.Sentence()
		val2 := gofakeit.Uint8()
		val3 := gofakeit.Bool()

		logger := WithContext(l, WithValueExtractor(key1, key2))

		ctx := context.WithValue(context.Background(), key1, val1)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), testText)
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", key1, val1))
		require.NotContains(t, sink.String(), key2)
		require.NotContains(t, sink.String(), key3)
		require.NotContains(t, sink.String(), ContextKey)

		message = gofakeit.Sentence()
		ctx = context.WithValue(ctx, key2, val2)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), testText)
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", key1, val1))
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%d", key2, val2))
		require.NotContains(t, sink.String(), key3)
		require.NotContains(t, sink.String(), ContextKey)

		message = gofakeit.Sentence()
		ctx = context.WithValue(ctx, key3, val3)
		logger.Ctx(ctx).Info(message)
		require.Contains(t, sink.String(), message)
		require.Contains(t, sink.String(), testText)
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%q", key1, val1))
		require.Contains(t, sink.String(), fmt.Sprintf("%q:%d", key2, val2))
		require.NotContains(t, sink.String(), key3)
		require.NotContains(t, sink.String(), ContextKey)
	})
}
