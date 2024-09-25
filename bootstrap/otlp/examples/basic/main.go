package main

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bootstrap/otlp"
)

func main() {
	os.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=example,example.name=basic")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	os.Setenv("OTEL_TRACES_SAMPLER", "parentbased_traceidratio")
	os.Setenv("OTEL_TRACES_SAMPLER_ARG", "1.0")
	os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "2000")

	ctx := context.Background()

	shutdown := otlp.Setup(ctx)
	defer shutdown(ctx)

	process(ctx)

	time.Sleep(time.Second * 10)
}

func process(ctx context.Context) {
	ctx, span := otlp.Tracer.Start(ctx, "process")
	defer span.End()

	task1(ctx)
	task2(ctx)
}

func task1(ctx context.Context) {
	ctx, span := otlp.Tracer.Start(ctx, "task 1")
	defer span.End()

	task1_1(ctx)
}

func task1_1(ctx context.Context) {
	ctx, span := otlp.Tracer.Start(ctx, "task 1.1")
	defer span.End()
}

func task2(ctx context.Context) {
	ctx, span := otlp.Tracer.Start(ctx, "task 2")
	defer span.End()
}
