// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestInitMeterProvider(t *testing.T) {
	tests := []struct {
		name         string
		cfg          configOptions
		wantProvider metric.MeterProvider
		wantErr      error
	}{
		{
			name:         "no-meter-provider-configured",
			wantProvider: noop.NewMeterProvider(),
		},
	}
	for _, tt := range tests {
		mp, err := initMeterProvider(tt.cfg)
		require.Equal(t, tt.wantProvider, mp)
		require.NoError(t, tt.wantErr, err)
	}
}
