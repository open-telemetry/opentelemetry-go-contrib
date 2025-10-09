// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf_test

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
)

func Example() {
	b, err := os.ReadFile(filepath.Join("testdata", "v0.3.yaml"))
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
	global.SetLoggerProvider(s.LoggerProvider())
}
