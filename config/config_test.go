// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestNewSDK(t *testing.T) {
	tests := []struct {
		name           string
		cfg            []ConfigurationOption
		tracerProvider any
		meterProvider  any
		err            error
	}{
		{
			name:           "no-configuration",
			tracerProvider: trace.NewNoopTracerProvider(),
			meterProvider:  noop.NewMeterProvider(),
		},
		{
			name: "with-configuration",
			cfg: []ConfigurationOption{
				WithContext(context.Background()),
				WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
					TracerProvider: &TracerProvider{},
					MeterProvider:  &MeterProvider{},
				}),
			},
			tracerProvider: &sdktrace.TracerProvider{},
			meterProvider:  &sdkmetric.MeterProvider{},
		},
	}
	for _, tt := range tests {
		sdk, err := NewSDK(tt.cfg...)
		require.Equal(t, tt.err, err)
		require.IsType(t, tt.tracerProvider, sdk.TracerProvider())
		require.IsType(t, tt.meterProvider, sdk.MeterProvider())
	}
}
