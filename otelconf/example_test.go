// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf_test

import (
	"context"
	"log"
	"os"
	"path/filepath"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
)

func Example() {
	b, err := os.ReadFile(filepath.Join("testdata", "v0.3.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	// parse a configuration file into an OpenTelemetryConfiguration model
	c, err := otelconf.ParseYAML(b)

	if err != nil {
		log.Fatal(err)
	}

	// instantiating SDK components with the parsed configuration
	s, err := otelconf.NewSDK(otelconf.WithOpenTelemetryConfiguration(*c))
	if err != nil {
		log.Fatal(err)
	}

	// ensure shutdown is eventually called for all components of the SDK
	defer func() {
		if err := s.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	// set global meter provider
	otel.SetMeterProvider(s.MeterProvider())

	// set global logger provider
	global.SetLoggerProvider(s.LoggerProvider())

	// set global tracer provider
	otel.SetTracerProvider(s.TracerProvider())

}
