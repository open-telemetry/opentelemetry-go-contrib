package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/samplers/jaeger_remote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	exporter, _ := stdouttrace.New(
		stdouttrace.WithoutTimestamps(),
	)

	jaegerRemoteSampler := jaeger_remote.New(
		// decrease polling interval to get quicker feedback
		jaeger_remote.WithPollingInterval(10*time.Second),
		// once the strategy is fetched, sample rate will drop
		jaeger_remote.WithInitialSamplingRate(1),
	)

	tp := trace.NewTracerProvider(
		trace.WithSampler(jaegerRemoteSampler),
		trace.WithSyncer(exporter), // for production usage, use trace.WithBatcher(exp)
	)
	otel.SetTracerProvider(tp)

	go generateSpans()

	// wait until program is interrupted
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

func generateSpans() {
	tracer := otel.GetTracerProvider().Tracer("example")

	for {
		_, span := tracer.Start(context.Background(), "span created at "+time.Now().String())
		time.Sleep(100 * time.Millisecond)
		span.End()

		time.Sleep(900 * time.Millisecond)
	}
}
