// Package contextlogger provides a context logger implementation for zap.Logger.
package contextlogger

import (
	"context"

	"go.uber.org/zap"
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
	var additionalFields []zap.Field

	for _, f := range c.extractors {
		additionalFields = append(additionalFields, f(ctx)...)
	}

	if len(additionalFields) == 0 {
		return c.logger
	}

	return c.logger.With(additionalFields...)
}
