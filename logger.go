// Package contextlogger provides a context logger implementation for zap.Logger.
package contextlogger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const ContextKey = "context-logger-context"

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
	var additionalFields []zap.Field

	for _, f := range c.extractors {
		additionalFields = append(additionalFields, f(ctx)...)
	}

	additionalFields = append(additionalFields, zap.Field{
		Key:       ContextKey,
		Type:      zapcore.SkipType,
		Interface: ctx,
	})

	return c.logger.With(additionalFields...)
}

func WithValueExtractor(key ...fmt.Stringer) ContextExtractor {
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
