---
name: golang-context-logger
description: "Use github.com/adlandh/context-logger to attach zap logging fields from context.Context. Apply when adding context-aware zap logging, request IDs, deadlines, OpenTelemetry trace IDs, Sentry span fields, or custom context extractors in Go projects."
user-invocable: true
license: MIT
compatibility: Designed for AI coding agents working in Go projects that use zap and github.com/adlandh/context-logger.
metadata:
  author: adlandh
  version: "1.0.0"
  openclaw:
    emoji: "log"
    homepage: https://github.com/adlandh/context-logger
    requires:
      bins:
        - go
    install:
      - go get github.com/adlandh/context-logger
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
---

# Go Context Logger

Use `github.com/adlandh/context-logger` when a Go project uses `go.uber.org/zap` and needs log entries enriched from `context.Context` values, deadlines, tracing spans, or custom request-scoped metadata.

This package keeps the normal zap API: call `ctxLogger.Ctx(ctx)` to get a `*zap.Logger` with extracted fields attached, then call zap methods like `Info`, `Error`, or `Debug`.

## When To Use

Use this skill when the task involves:

- Adding request IDs, user IDs, tenant IDs, correlation IDs, or similar request-scoped values to zap logs.
- Logging context deadline metadata such as `context_deadline_at`, `context_time_left`, and `context_error`.
- Adding OpenTelemetry `trace_id` and `span_id` fields to logs.
- Adding Sentry `trace_id`, `span_id`, `span_status`, and `span_op` fields to logs.
- Writing a custom `ContextExtractor`.
- Refactoring direct zap usage so request handlers or services log with context-derived fields.

## Install

```bash
go get github.com/adlandh/context-logger
```

Optional extractor modules have separate `go.mod` files and must be installed explicitly:

```bash
go get github.com/adlandh/context-logger/otel-extractor
go get github.com/adlandh/context-logger/sentry-extractor
```

## Core API

- `ctxlog.New(logger, extractors...)` creates a `*ContextLogger`.
- `ctxlog.WithContext(logger, extractors...)` is equivalent to `New`.
- `ctxLogger.Ctx(ctx)` returns a `*zap.Logger` with fields extracted from `ctx`.
- `ctxLogger.With(extractors...)` returns a new logger with more extractors; it does not mutate the original.
- `ctxLogger.Logger()` returns the underlying `*zap.Logger`.
- `ctxlog.ContextExtractor` is `func(context.Context) []zap.Field`.

`New` and `WithContext` use `zap.NewNop()` when passed a nil logger. `Ctx(nil)` is supported and uses `context.Background()`.

## Basic Pattern

Prefer an unexported typed context key that implements `fmt.Stringer`. The key string becomes the zap field name.

```go
package app

import (
    "context"

    ctxlog "github.com/adlandh/context-logger"
    "go.uber.org/zap"
)

type contextKey string

func (k contextKey) String() string { return string(k) }

const requestIDKey = contextKey("request_id")

func NewLogger(logger *zap.Logger) *ctxlog.ContextLogger {
    return ctxlog.WithContext(logger, ctxlog.WithValueExtractor(requestIDKey))
}

func Handle(ctx context.Context, logger *ctxlog.ContextLogger) {
    ctx = context.WithValue(ctx, requestIDKey, "req-123")
    logger.Ctx(ctx).Info("request received")
}
```

## Built-In Extractors

### Context Values

Use `WithValueExtractor` for values already stored in `context.Context`.

```go
ctxLogger := ctxlog.WithContext(
    logger,
    ctxlog.WithValueExtractor(requestIDKey, userIDKey, tenantIDKey),
)
```

Rules:

- Keys must be comparable and implement `fmt.Stringer`.
- Values are emitted with `zap.Any(key.String(), value)`.
- Missing or nil values are skipped.
- Prefer package-local key types over raw strings to avoid context key collisions.

### Deadlines

Use `WithDeadlineExtractor` when cancellation timing is useful in logs.

```go
ctxLogger := ctxlog.WithContext(logger, ctxlog.WithDeadlineExtractor())
ctxLogger.Ctx(ctx).Info("processing request")
```

Fields:

- `context_deadline_at`: the deadline as `time.Time`.
- `context_time_left`: duration until the deadline.
- `context_error`: present only when `ctx.Err()` is non-nil.

No fields are emitted when the context has no deadline.

### Context Carrier

Use `WithContextCarrier` only when a custom zap core or encoder expects the raw context.

```go
ctxLogger := ctxlog.WithContext(logger, ctxlog.WithContextCarrier("ctx"))
```

The carrier uses zap's skip field type, so it is not normally emitted by standard zap encoders.

## OpenTelemetry

Use the OpenTelemetry extractor when traces are already stored in context.

```go
import (
    ctxlog "github.com/adlandh/context-logger"
    otelextractor "github.com/adlandh/context-logger/otel-extractor"
)

ctxLogger := ctxlog.WithContext(logger, otelextractor.With())
ctxLogger.Ctx(ctx).Info("request handled")
```

Fields:

- `trace_id`
- `span_id`

The extractor emits nothing when `trace.SpanContextFromContext(ctx)` is invalid.

## Sentry

Use the Sentry extractor when Sentry spans are already stored in context.

```go
import (
    ctxlog "github.com/adlandh/context-logger"
    sentryextractor "github.com/adlandh/context-logger/sentry-extractor"
)

ctxLogger := ctxlog.WithContext(logger, sentryextractor.With())
ctxLogger.Ctx(ctx).Info("request handled")
```

Fields:

- `trace_id`
- `span_id`
- `span_status`
- `span_op`

The extractor emits nothing when `sentry.SpanFromContext(ctx)` returns nil.

## Custom Extractors

Create custom extractors for request-scoped metadata that is not covered by the built-ins.

```go
func WithUserAgent() ctxlog.ContextExtractor {
    return func(ctx context.Context) []zap.Field {
        value, ok := ctx.Value(userAgentKey).(string)
        if !ok || value == "" {
            return nil
        }

        return []zap.Field{zap.String("user_agent", value)}
    }
}
```

Guidelines:

- Extractors should be idempotent and safe to call on every log entry.
- Return nil when no field should be added.
- Keep extractors cheap; they run on each `Ctx(ctx)` call.
- Do not perform I/O or mutate context inside extractors.
- Prefer explicit zap field types like `zap.String`, `zap.Int`, or `zap.Duration` over `zap.Any` when the type is known.

## Handler Integration

For HTTP handlers, store metadata in `r.Context()` in middleware, then log with the request context.

```go
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func handler(ctxLogger *ctxlog.ContextLogger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctxLogger.Ctx(r.Context()).Info("request received")
        w.WriteHeader(http.StatusNoContent)
    }
}
```

## Composition

Combine built-in, tracing, and custom extractors at construction time.

```go
ctxLogger := ctxlog.WithContext(
    logger,
    ctxlog.WithValueExtractor(requestIDKey, userIDKey),
    ctxlog.WithDeadlineExtractor(),
    otelextractor.With(),
)
```

Use `With` to add extractors for a narrower scope without changing the base logger.

```go
auditLogger := ctxLogger.With(ctxlog.WithValueExtractor(auditIDKey))
auditLogger.Ctx(ctx).Info("audit event recorded")
```

## Testing

Use `zaptest/observer` to assert emitted fields.

```go
core, observed := observer.New(zap.InfoLevel)
logger := zap.New(core)

ctxLogger := ctxlog.WithContext(logger, ctxlog.WithValueExtractor(requestIDKey))
ctx := context.WithValue(context.Background(), requestIDKey, "req-123")

ctxLogger.Ctx(ctx).Info("test message")

entries := observed.TakeAll()
require.Len(t, entries, 1)
require.Equal(t, "req-123", entries[0].ContextMap()["request_id"])
```

Run the relevant module tests after changes:

```bash
go test -cover -race ./...
```

For extractor submodules:

```bash
cd otel-extractor && go test -cover -race ./...
cd sentry-extractor && go test -cover -race ./...
```

## Common Pitfalls

- Do not replace `ctx` with `context.Background()` in request paths just to log; propagate the existing context.
- Do not use raw string context keys in application code.
- Do not expect `WithContextCarrier` fields to appear in normal zap JSON output.
- Do not create extractors that allocate heavily, call external services, or depend on mutable global state.
- Do not store `*ContextLogger` in context; inject it like any other logger dependency.
- Do not call `Ctx(ctx)` once and reuse the returned `*zap.Logger` across different requests; call it with the current context when logging.

## Cross-References

- Use the `golang-context` skill for general `context.Context` propagation, cancellation, and value-key rules.
- Use the `golang-observability` skill when designing broader logging, tracing, metrics, and alerting.
- Use the `golang-testing` skill when adding or refactoring tests around context-aware logging.
