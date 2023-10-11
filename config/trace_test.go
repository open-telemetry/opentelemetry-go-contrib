// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/trace"
)

func TestInitTracerPovider(t *testing.T) {
	tests := []struct {
		name         string
		cfg          configOptions
		wantProvider trace.TracerProvider
	}{
		{
			name:         "no-tracer-provider-configured",
			wantProvider: trace.NewNoopTracerProvider(),
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.wantProvider, initTracerProvider(tt.cfg))
	}
}
