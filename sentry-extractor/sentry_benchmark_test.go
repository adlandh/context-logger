package sentryextractor

import (
	"context"
	"testing"

	"github.com/getsentry/sentry-go"
)

func BenchmarkWith(b *testing.B) {
	err := sentry.Init(sentry.ClientOptions{
		Transport: &transportMock{},
	})
	if err != nil {
		b.Fatalf("init sentry: %v", err)
	}

	extractor := With()

	rootSpan := sentry.StartSpan(context.Background(), "root-operation")
	defer rootSpan.Finish()

	childSpan := sentry.StartSpan(rootSpan.Context(), "child-operation")
	defer childSpan.Finish()

	benchmarks := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "no_span",
			ctx:  context.Background(),
		},
		{
			name: "root_span",
			ctx:  rootSpan.Context(),
		},
		{
			name: "child_span",
			ctx:  childSpan.Context(),
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
