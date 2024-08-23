// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport_test

import (
	"context"
	"os"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func Example_complete() {
	ctx := context.Background()

	// Only for demonstration purposes.
	_ = os.Setenv("OTEL_LOGS_EXPORTER", "otlp,console")
	_ = os.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	_ = os.Setenv("OTEL_METRICS_EXPORTER", "otlp")

	// Consider checking errors in your production code.
	logExporters, _ := autoexport.NewLogExporters(ctx)
	metricReaders, _ := autoexport.NewMetricReaders(ctx)
	traceExporters, _ := autoexport.NewSpanExporters(ctx)

	// Now that your exporters and readers are initialized,
	// you can simply initialize the different TracerProvider,
	// LoggerProvider and MeterProvider.
	// https://opentelemetry.io/docs/languages/go/getting-started/#initialize-the-opentelemetry-sdk

	// Traces
	var tracerProviderOpts []trace.TracerProviderOption
	for _, traceExporter := range traceExporters {
		tracerProviderOpts = append(tracerProviderOpts, trace.WithBatcher(traceExporter))
	}
	_ = trace.NewTracerProvider(tracerProviderOpts...)

	// Metrics
	var meterProviderOpts []metric.Option
	for _, metricReader := range metricReaders {
		meterProviderOpts = append(meterProviderOpts, metric.WithReader(metricReader))
	}
	_ = metric.NewMeterProvider(meterProviderOpts...)

	// Logs
	var loggerProviderOpts []log.LoggerProviderOption
	for _, logExporter := range logExporters {
		loggerProviderOpts = append(loggerProviderOpts, log.WithProcessor(
			log.NewBatchProcessor(logExporter),
		))
	}
	_ = log.NewLoggerProvider(loggerProviderOpts...)
}
