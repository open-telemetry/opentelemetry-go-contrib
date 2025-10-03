// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/propagators/ot"
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
			want:    propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
			wantErr: false,
		},
		{
			name: "valid tracecontext",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("tracecontext")},
					},
				},
			},
			want:    propagation.TraceContext{},
			wantErr: false,
		},
		{
			name: "valid baggage",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("baggage")},
					},
				},
			},
			want:    propagation.Baggage{},
			wantErr: false,
		},
		{
			name: "valid b3",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("b3")},
					},
				},
			},
			want:    b3.New(),
			wantErr: false,
		},
		{
			name: "valid b3multi",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("b3multi")},
					},
				},
			},
			want:    b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
			wantErr: false,
		},
		{
			name: "valid jaeger",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("jaeger")},
					},
				},
			},
			want:    jaeger.Jaeger{},
			wantErr: false,
		},
		{
			name: "valid xray",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("xray")},
					},
				},
			},
			want:    xray.Propagator{},
			wantErr: false,
		},
		{
			name: "valid ottrace",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("ottrace")},
					},
				},
			},
			want:    ot.OT{},
			wantErr: false,
		},
		{
			name: "multiple propagators",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("tracecontext"), ptr("baggage"), ptr("b3")},
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
			want:    propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
			wantErr: false,
		},
		{
			name: "empty propagator name",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr(""), ptr("tracecontext")},
					},
				},
			},
			want:    propagation.TraceContext{},
			wantErr: false,
		},
		{
			name: "nil propagator name",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{nil, ptr("tracecontext")},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unsupported propagator",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					Propagator: &Propagator{
						Composite: []*string{ptr("unknown")},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(),
			wantErr: true,
			errMsg:  "unknown propagator",
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
