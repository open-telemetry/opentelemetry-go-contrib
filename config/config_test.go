// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewSDK(t *testing.T) {
	tests := []struct {
		name               string
		cfg                []ConfigurationOption
		wantTracerProvider any
		wantMeterProvider  any
		wantLoggerProvider any
		wantErr            error
		wantShutdownErr    error
	}{
		{
			name:               "no-configuration",
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
			wantLoggerProvider: lognoop.NewLoggerProvider(),
		},
		{
			name: "with-configuration",
			cfg: []ConfigurationOption{
				WithContext(context.Background()),
				WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
					TracerProvider: &TracerProvider{},
					MeterProvider:  &MeterProvider{},
					LoggerProvider: &LoggerProvider{},
				}),
			},
			wantTracerProvider: &sdktrace.TracerProvider{},
			wantMeterProvider:  &sdkmetric.MeterProvider{},
			wantLoggerProvider: &sdklog.LoggerProvider{},
		},
	}
	for _, tt := range tests {
		sdk, err := NewSDK(tt.cfg...)
		require.Equal(t, tt.wantErr, err)
		assert.IsType(t, tt.wantTracerProvider, sdk.TracerProvider())
		assert.IsType(t, tt.wantMeterProvider, sdk.MeterProvider())
		assert.IsType(t, tt.wantLoggerProvider, sdk.LoggerProvider())
		require.Equal(t, tt.wantShutdownErr, sdk.Shutdown(context.Background()))
	}
}

func ptr[T any](v T) *T {
	return &v
}
