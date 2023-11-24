// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"context"

	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel"
)

func ExampleNewSDK() {
	// NewSDK returns a configured SDK as configured
	// per the options and any error that occurred during
	// the initialization process.
	configuredSDK, err := config.NewSDK(
		config.WithContext(context.Background()),
		config.WithOpenTelemetryConfiguration(config.OpenTelemetryConfiguration{
			TracerProvider: &config.TracerProvider{},
			MeterProvider:  &config.MeterProvider{},
		}))
	if err != nil {
		// Handle error appropriately.
		panic(err)
	}

	// This SDK can then be used to get a TracerProvider and
	// MeterProvider
	otel.SetTracerProvider(configuredSDK.TracerProvider())
	otel.SetMeterProvider(configuredSDK.MeterProvider())
	if err := configuredSDK.Shutdown(context.Background()); err != nil {
		panic(err)
	}
}
