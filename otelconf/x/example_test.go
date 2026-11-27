// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x_test

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/contrib/otelconf"
	"go.opentelemetry.io/contrib/otelconf/x"
)

func Example() {
	b, err := os.ReadFile(filepath.Join("testdata", "v1.0.0.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	// Parse a configuration file into an OpenTelemetryConfiguration model.
	c, err := otelconf.ParseYAML(b)
	if err != nil {
		log.Fatal(err)
	}

	// Create SDK components with the parsed configuration.
	s, err := otelconf.NewSDK(otelconf.WithOpenTelemetryConfiguration(*c))
	if err != nil {
		log.Fatal(err)
	}

	// Ensure shutdown is eventually called for all components of the SDK.
	defer func() {
		if err := s.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	// Set the global providers.
	otel.SetTracerProvider(s.TracerProvider())
	otel.SetMeterProvider(s.MeterProvider())
	otel.SetLoggerProvider(s.LoggerProvider())
	// Set the global propagator.
	otel.SetTextMapPropagator(s.Propagator())
}

// Serve metrics on the application's own HTTP server: build the Prometheus
// exporter in code with the application's registry and pass it as a reader
// with WithMeterProviderOptions. The configuration then needs a meter_provider,
// but no Prometheus reader of its own.
func ExampleWithMeterProviderOptions_prometheusRegisterer() {
	reg := prometheus.NewRegistry()
	reader, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		log.Fatal(err)
	}

	s, err := x.NewSDK(
		x.WithOpenTelemetryConfiguration(x.OpenTelemetryConfiguration{
			MeterProvider: &x.MeterProvider{},
		}),
		x.WithMeterProviderOptions(sdkmetric.WithReader(reader)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := s.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetMeterProvider(s.MeterProvider())

	// Expose the registry next to the application's other handlers.
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}
