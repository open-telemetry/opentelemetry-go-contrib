// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf_test

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/contrib/otelconf"
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

func ExampleWithConfigurationCustomizer() {
	// The service instance ID is only known at runtime, so it cannot be part
	// of the configuration file named by OTEL_CONFIG_FILE.
	instanceID := "instance-1"

	s, err := otelconf.NewSDK(
		otelconf.WithConfigurationCustomizer(func(cfg *otelconf.OpenTelemetryConfiguration) error {
			if cfg.Resource == nil {
				cfg.Resource = &otelconf.Resource{}
			}
			cfg.Resource.Attributes = append(cfg.Resource.Attributes, otelconf.AttributeNameValue{
				Name:  "service.instance.id",
				Value: instanceID,
			})
			// Keep the metric readers passed with WithMeterProviderOptions even
			// when the file has no meter_provider section.
			if cfg.MeterProvider == nil {
				cfg.MeterProvider = &otelconf.MeterProvider{}
			}
			return nil
		}),
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
}
