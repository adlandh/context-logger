module github.com/adlandh/context-logger/otel-extractor

go 1.22

toolchain go1.22.0

replace github.com/adlandh/context-logger => ../

require (
	github.com/adlandh/context-logger v0.0.0-00010101000000-000000000000
	github.com/brianvoe/gofakeit/v7 v7.0.2
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/otel v1.25.0
	go.opentelemetry.io/otel/trace v1.25.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.25.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)