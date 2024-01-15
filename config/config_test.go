// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
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

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantType interface{}
	}{
		{
			name:     "valid JSON",
			input:    `{"file_format": "json", "disabled": false}`,
			wantErr:  nil,
			wantType: &OpenTelemetryConfiguration{},
		},
		{
			name:     "invalid JSON",
			input:    `{"file_format": "json", "disabled":}`,
			wantErr:  fmt.Errorf("invalid character '}' looking for beginning of value"),
			wantType: nil,
		},
		{
			name:     "string containing valid JSON",
			input:    `{"file_format": "json", "disabled": false}I AM INVALID JSON`,
			wantErr:  fmt.Errorf("invalid character 'I' after top-level value"),
			wantType: nil,
		},
		{
			name:     "missing required field",
			input:    `{"foo": "bar"}`,
			wantErr:  fmt.Errorf("field file_format in OpenTelemetryConfiguration: required"),
			wantType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := ParseJSON(([]byte)(tt.input)); err != nil {
				require.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.IsType(t, tt.wantType, got)
			}
		})
	}
}
