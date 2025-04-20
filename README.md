# Context Logger

[![Go Reference](https://pkg.go.dev/badge/github.com/adlandh/context-logger.svg)](https://pkg.go.dev/github.com/adlandh/context-logger)
[![Go Report Card](https://goreportcard.com/badge/github.com/adlandh/context-logger)](https://goreportcard.com/report/github.com/adlandh/context-logger)

A lightweight Go library that enhances [Zap logger](https://pkg.go.dev/go.uber.org/zap) by automatically adding fields with values from context.

## Features

- Seamlessly integrates with Zap logger
- Extracts values from context and adds them as structured log fields
- Supports multiple extractors that can be combined
- Includes built-in extractors for common use cases
- Extensible with custom extractors

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

### Web Application Example

See the [full example](./example/main.go) for a web application using Echo framework.

## Available Extractors

### Built-in Extractors

- **WithValueExtractor**: Extracts values from context using keys that implement `fmt.Stringer`

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

