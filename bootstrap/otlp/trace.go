package otlp

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var Tracer = otel.GetTracerProvider().Tracer("go.opentelemetry.io/contrib/bootstrap/otlp")

func setupTrace(ctx context.Context) (ShutdownFunc, error) {
	exporter, err := otlptrace.New(ctx, otlptracehttp.NewClient())

	if err != nil {
		return emptyShutdown, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.Default(),
		),
	)

	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider.Shutdown, nil
}
