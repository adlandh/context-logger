// Package contextlogger provides a context logger implementation for zap.Logger.
package contextlogger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ContextExtractor func(ctx context.Context) []zap.Field

type ContextLogger struct {
	logger     *zap.Logger
	extractors []ContextExtractor
}

func WithContext(logger *zap.Logger, extractors ...ContextExtractor) *ContextLogger {
	return &ContextLogger{
		logger:     logger,
		extractors: extractors,
	}
}

func (c ContextLogger) Ctx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		ctx = context.Background()
	}

	additionalFields := make([]zap.Field, 0, len(c.extractors))

	for _, f := range c.extractors {
		additionalFields = append(additionalFields, f(ctx)...)
	}

	return c.logger.With(additionalFields...)
}

func WithValueExtractor[T interface {
	comparable
	fmt.Stringer
}](key ...T) ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		if len(key) == 0 {
			return nil
		}

		fields := make([]zap.Field, 0, len(key))

		for _, k := range key {
			if val := ctx.Value(k); val != nil {
				fields = append(fields, zap.Any(k.String(), val))
			}
		}

		return fields
	}
}

func WithContextCarrier(fieldName string) ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		if fieldName == "" {
			return nil
		}

		return []zap.Field{
			{
				Key:       fieldName,
				Type:      zapcore.SkipType,
				Interface: ctx,
			},
		}
	}
}

func WithDeadlineExtractor() ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		deadline, ok := ctx.Deadline()
		if !ok {
			return nil
		}

		dlfields := make([]zap.Field, 2)

		dlfields[0] = zap.Time("context_deadline_at", deadline)
		dlfields[1] = zap.Duration("context_time_left", time.Until(deadline))
		return dlfields
	}
}
