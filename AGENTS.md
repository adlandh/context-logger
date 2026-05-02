# Repository Guidelines for Agents

## Shape Of This Repo

- This is a Go 1.25 zap logging library, not an app. Root package is `github.com/adlandh/context-logger` in `logger.go`.
- `otel-extractor/` and `sentry-extractor/` are separate Go modules with their own `go.mod` and `go.sum`; they are not in a Go workspace.
- The extractor modules currently depend on the published root module version (`github.com/adlandh/context-logger v1.6.3`), not the local parent directory. Local root changes are not automatically exercised by submodule tests.
- `example/` is a separate Echo example module and is reference-only; CI does not test it.
- `skills/golang-context-logger/SKILL.md` documents how agents should use this package in downstream Go projects.

## Commands That Matter

- Root tests: `go test -cover -race ./...`
- Root single test: `go test -v -run TestContextLogger_WithValueExtractor ./...`
- OTel extractor tests: `cd otel-extractor && go test -cover -race ./...`
- OTel single test: `cd otel-extractor && go test -v -run TestOtelExtractor_WithTracerProvider ./...`
- Sentry extractor tests: `cd sentry-extractor && go test -cover -race ./...`
- Sentry single test: `cd sentry-extractor && go test -v -run TestSentryExtractor_WithSpan ./...`
- CI-equivalent test coverage files: root uses `go test -race -coverprofile=coverage.txt -covermode=atomic ./...`; submodules write `../coverage1.txt` and `../coverage2.txt`.

## Linting Gotchas

- `.golangci.yml` is v2 config for Go 1.25 and has `run.tests: false`; lint does not analyze tests unless config changes.
- CI and Lefthook download `.golangci.yml` from `adlandh/golangci-lint-config` before linting, overwriting the local file.
- Local lint matching Lefthook: `curl -sS https://raw.githubusercontent.com/adlandh/golangci-lint-config/refs/heads/main/.golangci.yml -o .golangci.yml && golangci-lint run && cd ./otel-extractor && golangci-lint run && cd ../sentry-extractor && golangci-lint run`.
- If you only need current checked-in config, run `golangci-lint run` separately in root, `otel-extractor/`, and `sentry-extractor/`.
- Formatters configured by golangci-lint are `gofmt` and `goimports`.

## API Conventions To Preserve

- `New` and `WithContext` are equivalent constructors and must keep the nil-logger fallback to `zap.NewNop()`.
- `Ctx(nil)` is supported and uses `context.Background()`.
- `Ctx(ctx)` applies all non-nil `ContextExtractor` functions and returns `c.logger.With(fields...)`; do not add logging side effects inside `Ctx`.
- `With(extractors...)` returns a new `ContextLogger` when adding extractors; it must not mutate the original extractor slice.
- `WithValueExtractor` accepts comparable `fmt.Stringer` keys and emits fields with `zap.Any(k.String(), val)` only when `ctx.Value(k)` is non-nil.
- `WithContextCarrier` intentionally uses `zapcore.SkipType`; standard zap encoders will not emit the carrier field.
- `WithDeadlineExtractor` only emits fields when the context has a deadline, and adds `context_error` only when `ctx.Err()` is non-nil.

## Testing Patterns

- Tests use `github.com/stretchr/testify/require` and `go.uber.org/zap/zaptest/observer`; keep new tests consistent with `newTestLogger` and `logAndAssert` helpers.
- Test context keys are typed values implementing `String() string`; avoid raw string keys in new extractor tests unless specifically testing carrier field names.
- When changing exported field names, update constants and tests in the affected module together.

## Release / Dependency Awareness

- Dependabot watches the three Go modules separately: `/`, `/otel-extractor`, and `/sentry-extractor`.
- If a change spans root plus extractor modules, remember the submodule manifests do not point at local root by default; verify how you want to test that cross-module change before assuming `go test` covers it.
