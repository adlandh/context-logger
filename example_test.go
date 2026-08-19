package contextlogger_test

import (
	"context"
	"os"

	ctxlog "github.com/adlandh/context-logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type exampleContextKey string

func (k exampleContextKey) String() string { return string(k) }

const exampleRequestIDKey = exampleContextKey("request_id")

// newExampleLogger returns a JSON logger writing to stdout so example output
// can be verified with // Output: comments.
func newExampleLogger() *zap.Logger {
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
	return zap.New(zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel))
}

func ExampleNew() {
	logger := newExampleLogger()
	ctxLogger := ctxlog.New(logger, ctxlog.WithValueExtractor(exampleRequestIDKey))

	ctx := context.WithValue(context.Background(), exampleRequestIDKey, "req-7f3")
	ctxLogger.Ctx(ctx).Info("request completed")

	// Output:
	// {"msg":"request completed","request_id":"req-7f3"}
}

func ExampleContextLogger_Ctx() {
	logger := newExampleLogger()
	ctxLogger := ctxlog.WithContext(logger, ctxlog.WithValueExtractor(exampleRequestIDKey))

	// Add an extractor for a narrower scope; the base logger is unchanged.
	userIDKey := exampleContextKey("user_id")
	auditLogger := ctxLogger.With(ctxlog.WithValueExtractor(userIDKey))

	ctx := context.WithValue(context.Background(), exampleRequestIDKey, "req-7f3")
	ctx = context.WithValue(ctx, userIDKey, "user-123")
	auditLogger.Ctx(ctx).Info("audit event recorded")

	// Output:
	// {"msg":"audit event recorded","request_id":"req-7f3","user_id":"user-123"}
}
