// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel/propagation"
)

func TestPropagator(t *testing.T) {
	tests := []struct {
		name    string
		cfg     configOptions
		want    propagation.TextMapPropagator
		wantErr bool
		errMsg  string
	}{
		{
			name: "nil propagator config",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: nil,
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "valid tracecontext",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("tracecontext")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}),
			wantErr: false,
		},
		{
			name: "valid baggage",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("baggage")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(propagation.Baggage{}),
			wantErr: false,
		},
		{
			name: "valid b3",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("b3")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(b3.New()),
			wantErr: false,
		},
		{
			name: "valid b3multi",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("b3multi")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))),
			wantErr: false,
		},
		{
			name: "valid jaeger",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("jaeger")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(jaeger.Jaeger{}),
			wantErr: false,
		},
		{
			name: "valid xray",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("xray")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(xray.Propagator{}),
			wantErr: false,
		},
		{
			name: "valid ottrace",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("ottrace")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(ot.OT{}),
			wantErr: false,
		},
		{
			name: "multiple propagators",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("tracecontext"), strPtr("baggage"), strPtr("b3")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}, b3.New()),
			wantErr: false,
		},
		{
			name: "empty composite",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "nil propagator name",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{nil, strPtr("tracecontext")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}),
			wantErr: false,
		},
		{
			name: "unsupported propagator",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{strPtr("unknown")},
					},
				},
			},
			want:    nil,
			wantErr: true,
			errMsg:  "unsupported propagator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := propagator(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
