package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"go.uber.org/zap"

	"go.opentelemetry.io/contrib/examples/config/telemetry"
)

var otelCfgFile string

func main() {
	ctx := context.Background()
	flag.StringVar(&otelCfgFile, "otel", "./otel.yaml", "otel config file")

	telemetryShutdown, err := telemetry.Setup(ctx, otelCfgFile)
	if err != nil {
		fmt.Printf("Error setting up telemetry: %v\n", err)
		os.Exit(1)
	}

	logger := telemetry.Logger()
	logger.Info("Starting the config example")

	// ensure telemetry is shutdown, flushing any remaining data
	defer func() {
		if err := telemetryShutdown(context.Background()); err != nil {
			logger.Fatal("Error shutting down telemetry", zap.Error(err))
		}
	}()

	// catch signals to allow for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	// Your application code here

	// here's an example of a span
	_, span := telemetry.Tracer().Start(ctx, "main")
	span.AddEvent("doing boo")
	span.End()

	// and here's an example of a metric
	counter, err := telemetry.Meter().Int64Counter("my-metric")
	if err != nil {
		logger.Error("Failed to create counter", zap.Error(err))
	}
	counter.Add(ctx, 100)

	// wait for a signal to stop
	<-ctx.Done()
	logger.Info("Stopping the config example")
}
