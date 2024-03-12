// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build go1.18
// +build go1.18

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var res = resource.NewWithAttributes(
	semconv.SchemaURL,
	semconv.ServiceName("host-instrumentation-example"),
)

func main() {
	exp, err := stdoutmetric.New()
	if err != nil {
		log.Fatal(err)
	}

	// Register the exporter with an SDK via a periodic reader.
	read := metric.NewPeriodicReader(exp, metric.WithInterval(1*time.Second))
	provider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(read))
	defer func() {
		err := provider.Shutdown(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.Print("Starting host instrumentation:")
	err = host.Start(host.WithMeterProvider(provider))
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
}
