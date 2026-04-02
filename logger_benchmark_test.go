package contextlogger

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

type benchmarkContextKey string

func (k benchmarkContextKey) String() string {
	return string(k)
}

func BenchmarkContextLoggerCtx(b *testing.B) {
	userIDKey := benchmarkContextKey("user_id")
	requestIDKey := benchmarkContextKey("request_id")

	backgroundCtx := context.Background()
	valueCtx := context.WithValue(backgroundCtx, userIDKey, "user-123")
	combinedCtx := context.WithValue(valueCtx, requestIDKey, "req-456")
	deadlineCtx, cancel := context.WithDeadline(backgroundCtx, time.Now().Add(time.Hour))
	defer cancel()
	combinedDeadlineCtx := context.WithValue(deadlineCtx, userIDKey, "user-123")
	combinedDeadlineCtx = context.WithValue(combinedDeadlineCtx, requestIDKey, "req-456")

	benchmarks := []struct {
		name   string
		logger *ContextLogger
		ctx    context.Context
	}{
		{
			name:   "no_extractors/background",
			logger: WithContext(zap.NewNop()),
			ctx:    backgroundCtx,
		},
		{
			name:   "no_extractors/nil_context",
			logger: WithContext(zap.NewNop()),
			ctx:    nil,
		},
		{
			name:   "value_extractor/single_key",
			logger: WithContext(zap.NewNop(), WithValueExtractor(userIDKey)),
			ctx:    valueCtx,
		},
		{
			name:   "value_extractor/two_keys",
			logger: WithContext(zap.NewNop(), WithValueExtractor(userIDKey, requestIDKey)),
			ctx:    combinedCtx,
		},
		{
			name:   "deadline_extractor",
			logger: WithContext(zap.NewNop(), WithDeadlineExtractor()),
			ctx:    deadlineCtx,
		},
		{
			name: "combined_extractors",
			logger: WithContext(
				zap.NewNop(),
				WithValueExtractor(userIDKey, requestIDKey),
				WithDeadlineExtractor(),
			),
			ctx: combinedDeadlineCtx,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = bm.logger.Ctx(bm.ctx)
			}
		})
	}
}

func BenchmarkContextLoggerWith(b *testing.B) {
	base := WithContext(zap.NewNop(), WithValueExtractor(benchmarkContextKey("user_id")))
	requestIDKey := benchmarkContextKey("request_id")

	b.ReportAllocs()
	for b.Loop() {
		_ = base.With(WithValueExtractor(requestIDKey), WithDeadlineExtractor())
	}
}
