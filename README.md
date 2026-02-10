# Context Logger

[![Go Reference](https://pkg.go.dev/badge/github.com/adlandh/context-logger.svg)](https://pkg.go.dev/github.com/adlandh/context-logger)
[![Go Report Card](https://goreportcard.com/badge/github.com/adlandh/context-logger)](https://goreportcard.com/report/github.com/adlandh/context-logger)

A lightweight Go library that enhances [Zap logger](https://pkg.go.dev/go.uber.org/zap) by automatically adding fields from `context.Context`.

## Features

- Seamlessly integrates with Zap logger
- Extracts values from context and adds them as structured log fields
- Supports multiple extractors that can be combined
- Includes built-in extractors for common use cases
- Extensible with custom extractors
- Keeps Zap APIs familiar via a context-aware wrapper

## Installation

```bash
go get github.com/adlandh/context-logger
```

## Usage

### Basic Usage

```go
package main

import (
	"context"

	ctxlog "github.com/adlandh/context-logger"
	"go.uber.org/zap"
)

type contextKey string

func (c contextKey) String() string {
	return string(c)
}

var userIDKey = contextKey("user_id")

func main() {
	// Create a Zap logger
	logger, _ := zap.NewProduction()

	// Create a context logger with a value extractor
	ctxLogger := ctxlog.WithContext(logger, ctxlog.WithValueExtractor(userIDKey))

	// Create a context with a value
	ctx := context.WithValue(context.Background(), userIDKey, "user-123")

	// Log with the context
	// This will automatically include "user_id":"user-123" in the log entry
	ctxLogger.Ctx(ctx).Info("User action performed")
}
```

`Ctx(ctx)` returns a `*zap.Logger` with fields extracted from `ctx` attached via `With(...)`; `Ctx(nil)` is supported and uses `context.Background()`.

### Extractors and Composition

`WithContext` accepts one or more extractors. Each extractor can add fields derived from the context, and all extractors are applied for every log call.

```go
ctxLogger := ctxlog.WithContext(
	logger,
	ctxlog.WithValueExtractor(userIDKey),
	ctxlog.WithValueExtractor(contextKey("request_id")),
	ctxlog.WithDeadlineExtractor(),
	ctxlog.WithContextCarrier("ctx"),
)
```

### Context Key Guidelines

`WithValueExtractor` expects keys that implement `fmt.Stringer`. This lets the extractor use the key's string value as the log field name.

Use a typed key (like the `contextKey` example) instead of raw string keys to avoid collisions across packages.

```go
type contextKey string

func (c contextKey) String() string { return string(c) }
```

### Web Application Example

See the [full example](./example/main.go) for a web application using Echo framework.

## Available Extractors

### Built-in Extractors

- **WithValueExtractor**: Extracts values from context using keys that implement `fmt.Stringer`
- **WithDeadlineExtractor**: Extracts deadline metadata from context (`context_deadline_at`, `context_time_left`) and adds `context_error` when the context is done
- **WithContextCarrier**: Attaches the `context.Context` to the logger for custom cores/encoders (field is not emitted by default)

Usage note: `WithContextCarrier` is useful when you have a custom zap core/encoder that knows how to pull values from the context. The carrier field is a skip-type field, so it will not appear in logs unless your core/encoder handles it explicitly.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

ctxLogger := ctxlog.WithContext(logger, ctxlog.WithDeadlineExtractor())
ctxLogger.Ctx(ctx).Info("processing request")
// Adds:
// - context_deadline_at (time.Time)
// - context_time_left (time.Duration)
// - context_error (string, only when ctx.Err() is non-nil)
```

### Additional Extractors (in separate modules)

- **Sentry Extractor**: Extracts Sentry trace information (trace_id, span_id, span_status, span_op)
  ```go
  import "github.com/adlandh/context-logger/sentry-extractor"

  // Use the extractor
  ctxLogger := ctxlog.WithContext(logger, sentryextractor.With())
  ```

- **OpenTelemetry Extractor**: Extracts OpenTelemetry trace information (trace_id, span_id)
  ```go
  import "github.com/adlandh/context-logger/otel-extractor"

  // Use the extractor
  ctxLogger := ctxlog.WithContext(logger, otelextractor.With())
  ```

### Installing Extractor Modules

Each extractor module has its own `go.mod`, so install them explicitly:

```bash
go get github.com/adlandh/context-logger/otel-extractor
go get github.com/adlandh/context-logger/sentry-extractor
```

## Creating Custom Extractors

You can create custom extractors by implementing the `ContextExtractor` function type:

```go
func MyCustomExtractor() ctxlog.ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		// Extract values from context
		// Return them as zap.Field slice
		return []zap.Field{
			zap.String("custom_field", "custom_value"),
		}
	}
}
```

## API Overview

- `WithContext(logger, extractors...)` wraps a Zap logger and returns a context-aware facade
- `Ctx(ctx)` returns a logger bound to that context for the next call
- `ContextExtractor` is `func(context.Context) []zap.Field`

## Testing

```bash
go test -cover -race ./...
cd otel-extractor && go test -cover -race ./...
cd sentry-extractor && go test -cover -race ./...
```
