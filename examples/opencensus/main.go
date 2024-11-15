// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	ocmetric "go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	octrace "go.opencensus.io/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	// instrumenttype differentiates between our gauge and view metrics.
	keyType = tag.MustNewKey("instrumenttype")
	// Counts the number of lines read in from standard input.
	countMeasure = stats.Int64("test_count", "A count of something", stats.UnitDimensionless)
	countView    = &view.View{
		Name:        "test_count",
		Measure:     countMeasure,
		Description: "A count of something",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{keyType},
	}
)

func main() {
	log.Println("Using OpenTelemetry stdout exporters.")
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(fmt.Errorf("error creating trace exporter: %w", err))
	}
	metricsExporter, err := stdoutmetric.New()
	if err != nil {
		log.Fatal(fmt.Errorf("error creating metric exporter: %w", err))
	}
	tracing(traceExporter)
	if err := monitoring(metricsExporter); err != nil {
		log.Fatal(err)
	}
}

// tracing demonstrates overriding the OpenCensus DefaultTracer to send spans
// to the OpenTelemetry exporter by calling OpenCensus APIs.
func tracing(otExporter sdktrace.SpanExporter) {
	ctx := context.Background()

	log.Println("Configuring OpenCensus.  Not Registering any OpenCensus exporters.")
	octrace.ApplyConfig(octrace.Config{DefaultSampler: octrace.AlwaysSample()})

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(otExporter))
	otel.SetTracerProvider(tp)

	log.Println("Installing the OpenCensus bridge to make OpenCensus libraries write spans using OpenTelemetry.")
	opencensus.InstallTraceBridge()
	tp.ForceFlush(ctx)

	log.Println("Creating OpenCensus span, which should be printed out using the OpenTelemetry stdouttrace exporter.\n-- It should have no parent, since it is the first span.")
	ctx, outerOCSpan := octrace.StartSpan(ctx, "OpenCensusOuterSpan")
	outerOCSpan.End()
	tp.ForceFlush(ctx)

	log.Println("Creating OpenTelemetry span\n-- It should have the OpenCensus span as a parent, since the OpenCensus span was written with using OpenTelemetry APIs.")
	tracer := tp.Tracer("go.opentelemetry.io/contrib/examples/opencensus")
	ctx, otspan := tracer.Start(ctx, "OpenTelemetrySpan")
	otspan.End()
	tp.ForceFlush(ctx)

	log.Println("Creating OpenCensus span, which should be printed out using the OpenTelemetry stdouttrace exporter.\n-- It should have the OpenTelemetry span as a parent, since it was written using OpenTelemetry APIs")
	_, innerOCSpan := octrace.StartSpan(ctx, "OpenCensusInnerSpan")
	innerOCSpan.End()
	tp.ForceFlush(ctx)
}

// monitoring demonstrates creating an IntervalReader using the OpenTelemetry
// exporter to send metrics to the exporter by using either an OpenCensus
// registry or an OpenCensus view.
func monitoring(exporter metric.Exporter) error {
	log.Println("Adding the OpenCensus metric Producer to an OpenTelemetry Reader to export OpenCensus metrics using the OpenTelemetry stdout exporter.")
	// Register the OpenCensus metric Producer to add metrics from OpenCensus to the output.
	reader := metric.NewPeriodicReader(exporter, metric.WithProducer(opencensus.NewMetricProducer()))
	metric.NewMeterProvider(metric.WithReader(reader))

	log.Println("Registering a gauge metric using an OpenCensus registry.")
	r := ocmetric.NewRegistry()
	metricproducer.GlobalManager().AddProducer(r)
	gauge, err := r.AddInt64Gauge(
		"test_gauge",
		ocmetric.WithDescription("A gauge for testing"),
		ocmetric.WithConstLabel(map[metricdata.LabelKey]metricdata.LabelValue{
			{Key: keyType.Name()}: metricdata.NewLabelValue("gauge"),
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to add gauge: %w", err)
	}
	entry, err := gauge.GetEntry()
	if err != nil {
		return fmt.Errorf("failed to get gauge entry: %w", err)
	}

	log.Println("Registering a cumulative metric using an OpenCensus view.")
	if err := view.Register(countView); err != nil {
		return fmt.Errorf("failed to register views: %w", err)
	}
	ctx, err := tag.New(context.Background(), tag.Insert(keyType, "view"))
	if err != nil {
		return fmt.Errorf("failed to set tag: %w", err)
	}
	for i := int64(1); true; i++ {
		// update stats for our gauge
		entry.Set(i)
		// update stats for our view
		stats.Record(ctx, countMeasure.M(1))
		time.Sleep(time.Second)
	}
	return nil
}
