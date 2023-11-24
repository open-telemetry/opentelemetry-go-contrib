// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewSDK(t *testing.T) {
	tests := []struct {
		name               string
		cfg                []ConfigurationOption
		wantTracerProvider any
		wantMeterProvider  any
		wantErr            error
		wantShutdownErr    error
	}{
		{
			name:               "no-configuration",
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
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
			wantTracerProvider: &sdktrace.TracerProvider{},
			wantMeterProvider:  &sdkmetric.MeterProvider{},
		},
	}
	for _, tt := range tests {
		sdk, err := NewSDK(tt.cfg...)
		require.Equal(t, tt.wantErr, err)
		assert.IsType(t, tt.wantTracerProvider, sdk.TracerProvider())
		assert.IsType(t, tt.wantMeterProvider, sdk.MeterProvider())
		require.Equal(t, tt.wantShutdownErr, sdk.Shutdown(context.Background()))
	}
}

func strPtr(val string) *string {
	return &val
}

func intPtr(val int) *int {
	return &val
}

func TestNewResource(t *testing.T) {
	tests := []struct {
		name         string
		config       *Resource
		wantResource *resource.Resource
	}{
		{
			name: "nil resource",
		},
		{
			name: "resource with service name",
			config: &Resource{
				Attributes: &Attributes{
					ServiceName: strPtr("myservice"),
				},
			},
			wantResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("myservice"),
			),
		},
	}
	for _, tt := range tests {
		res, err := newResource(tt.config)
		require.NoError(t, err)
		wantedRes, err := resource.Merge(resource.Default(), tt.wantResource)
		require.NoError(t, err)
		require.Equal(t, wantedRes, res)
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		wantEndpoint string
	}{
		{
			name:         "no prefix",
			endpoint:     "localhost:1234",
			wantEndpoint: "http://localhost:1234",
		},
		{
			name:         "http prefix",
			endpoint:     "http://localhost:1234",
			wantEndpoint: "http://localhost:1234",
		},
		{
			name:         "https prefix",
			endpoint:     "https://localhost:1234",
			wantEndpoint: "https://localhost:1234",
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.wantEndpoint, normalizeEndpoint(tt.endpoint))
	}
}
