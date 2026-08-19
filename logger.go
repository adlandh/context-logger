// Package contextlogger provides a context logger implementation for zap.Logger.
package contextlogger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ContextExtractor extracts zap fields from a context.
type ContextExtractor func(ctx context.Context) []zap.Field

const (
	// FieldContextDeadlineAt identifies the context deadline timestamp field.
	FieldContextDeadlineAt = "context_deadline_at"
	// FieldContextTimeLeft identifies the duration remaining before the deadline.
	FieldContextTimeLeft = "context_time_left"
	// FieldContextError identifies the context cancellation or deadline error.
	FieldContextError = "context_error"
	// FieldContextCause identifies a cancellation cause distinct from the context error.
	FieldContextCause = "context_cause"
)

// ContextLogger attaches fields extracted from a context to a zap logger.
type ContextLogger struct {
	logger     *zap.Logger
	extractors []ContextExtractor
}

// New creates a ContextLogger and falls back to a no-op logger when logger is nil.
func New(logger *zap.Logger, extractors ...ContextExtractor) *ContextLogger {
	if logger == nil {
		logger = zap.NewNop()
	}

	copiedExtractors := append([]ContextExtractor(nil), extractors...)

	return &ContextLogger{
		logger:     logger,
		extractors: copiedExtractors,
	}
}

// WithContext creates a ContextLogger and falls back to a no-op logger when logger is nil.
func WithContext(logger *zap.Logger, extractors ...ContextExtractor) *ContextLogger {
	return New(logger, extractors...)
}

// Ctx returns the underlying logger with fields extracted from ctx.
// A nil context is treated as context.Background().
func (c *ContextLogger) Ctx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		ctx = context.Background()
	}

	additionalFields := make([]zap.Field, 0, len(c.extractors))

	for _, f := range c.extractors {
		if f == nil {
			continue
		}

		additionalFields = append(additionalFields, f(ctx)...)
	}

	return c.logger.With(additionalFields...)
}

// With returns a new ContextLogger with the additional extractors.
// It returns the receiver unchanged when no extractors are provided.
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

// Logger returns the underlying zap logger.
func (c *ContextLogger) Logger() *zap.Logger {
	return c.logger
}

// WithValueExtractor extracts non-nil context values using each key's string
// representation as the zap field name.
func WithValueExtractor[T interface {
	comparable
	fmt.Stringer
}](key ...T) ContextExtractor {
	keys := append([]T(nil), key...)

	return func(ctx context.Context) []zap.Field {
		if len(keys) == 0 {
			return nil
		}

		fields := make([]zap.Field, 0, len(keys))

		for _, k := range keys {
			if val := ctx.Value(k); val != nil {
				fields = append(fields, zap.Any(k.String(), val))
			}
		}

		return fields
	}
}

// WithContextCarrier exposes ctx to custom zap cores under fieldName.
// Standard zap encoders skip the carrier field.
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

// WithDeadlineExtractor adds context_deadline_at and context_time_left when the context
// has a deadline. After cancellation or deadline expiry it adds context_error, and
// context_cause when the cancellation cause (see context.WithCancelCause) differs from
// the context error.
func WithDeadlineExtractor() ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		deadline, ok := ctx.Deadline()
		if !ok {
			return nil
		}

		fields := make([]zap.Field, 0, 4)

		fields = append(fields,
			zap.Time(FieldContextDeadlineAt, deadline),
			zap.Duration(FieldContextTimeLeft, time.Until(deadline)),
		)

		if err := ctx.Err(); err != nil {
			fields = append(fields, zap.String(FieldContextError, err.Error()))

			//nolint:errorlint // exact equality is intentional: Cause returns ctx.Err() verbatim
			// when no cause was set, while errors.Is would also drop wrapped causes.
			if cause := context.Cause(ctx); cause != err {
				fields = append(fields, zap.String(FieldContextCause, cause.Error()))
			}
		}

		return fields
	}
}
