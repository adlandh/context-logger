package otelextractor

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func BenchmarkWith(b *testing.B) {
	extractor := With()

	remoteSpanContext := createSpanContext(
		[]byte{0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19},
		[]byte{0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11},
	)

	remoteSpanCtx := trace.ContextWithRemoteSpanContext(context.Background(), remoteSpanContext)

	provider := noop.NewTracerProvider()
	activeSpanCtx, span := provider.Tracer("benchmark").Start(remoteSpanCtx, "active-span")
	defer span.End()

	benchmarks := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "no_span",
			ctx:  context.Background(),
		},
		{
			name: "remote_span",
			ctx:  remoteSpanCtx,
		},
		{
			name: "active_span",
			ctx:  activeSpanCtx,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = extractor(bm.ctx)
			}
		})
	}
}
