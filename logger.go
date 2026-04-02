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

const (
	FieldContextDeadlineAt = "context_deadline_at"
	FieldContextTimeLeft   = "context_time_left"
	FieldContextError      = "context_error"
)

type ContextLogger struct {
	logger     *zap.Logger
	extractors []ContextExtractor
}

// New creates a ContextLogger and falls back to a no-op logger when logger is nil.
func New(logger *zap.Logger, extractors ...ContextExtractor) *ContextLogger {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ContextLogger{
		logger:     logger,
		extractors: extractors,
	}
}

// WithContext creates a ContextLogger and falls back to a no-op logger when logger is nil.
func WithContext(logger *zap.Logger, extractors ...ContextExtractor) *ContextLogger {
	return New(logger, extractors...)
}

func (c *ContextLogger) Ctx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		ctx = context.Background()
	}

	additionalFields := make([]zap.Field, 0, len(c.extractors))

	for _, f := range c.extractors {
		additionalFields = append(additionalFields, f(ctx)...)
	}

	return c.logger.With(additionalFields...)
}

func (c *ContextLogger) With(extractors ...ContextExtractor) *ContextLogger {
	if len(extractors) == 0 {
		return c
	}

	if len(c.extractors) == 0 {
		return New(c.logger, extractors...)
	}

	combined := make([]ContextExtractor, len(c.extractors)+len(extractors))
	copy(combined, c.extractors)
	copy(combined[len(c.extractors):], extractors)

	return &ContextLogger{
		logger:     c.logger,
		extractors: combined,
	}
}

func (c *ContextLogger) Logger() *zap.Logger {
	return c.logger
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

		fields := make([]zap.Field, 0, 3)

		fields = append(fields,
			zap.Time(FieldContextDeadlineAt, deadline),
			zap.Duration(FieldContextTimeLeft, time.Until(deadline)),
		)

		if ctx.Err() != nil {
			fields = append(fields, zap.String(FieldContextError, ctx.Err().Error()))
		}

		return fields
	}
}
