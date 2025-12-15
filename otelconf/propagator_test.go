// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/propagators/ot"
)

func TestPropagator(t *testing.T) {
	tests := []struct {
		name    string
		cfg     OpenTelemetryConfigurationPropagator
		want    propagation.TextMapPropagator
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil propagator config",
			cfg:     nil,
			want:    propagation.NewCompositeTextMapPropagator(),
			wantErr: false,
		},
		{
			name: "valid tracecontext",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{
						Tracecontext: TraceContextPropagator{},
					},
				},
			},
			want:    propagation.TraceContext{},
			wantErr: false,
		},
		{
			name: "valid baggage",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{
						Baggage: BaggagePropagator{},
					},
				},
			},
			want:    propagation.Baggage{},
			wantErr: false,
		},
		{
			name: "valid b3",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{
						B3: B3Propagator{},
					},
				},
			},
			want:    b3.New(),
			wantErr: false,
		},
		{
			name: "valid b3multi",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{
						B3Multi: B3MultiPropagator{},
					},
				},
			},
			want:    b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
			wantErr: false,
		},
		{
			name: "valid jaeger",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{
						Jaeger: JaegerPropagator{},
					},
				},
			},
			want:    jaeger.Jaeger{},
			wantErr: false,
		},
		{
			name: "valid ottrace",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{
						Ottrace: OpenTracingPropagator{},
					},
				},
			},
			want:    ot.OT{},
			wantErr: false,
		},
		// {
		// 	name: "valid xray",
		// 	cfg: configOptions{
		// 		opentelemetryConfig: OpenTelemetryConfiguration{
		// 			Propagator: &PropagatorJson{
		// 				Composite: []TextMapPropagator{
		// 					{
		// 						AdditionalProperties: map[string]any{
		// 							"xray": "",
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	want:    xray.Propagator{},
		// 	wantErr: false,
		// },
		{
			name: "multiple propagators",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{
						Tracecontext: TraceContextPropagator{},
					},
					{
						Baggage: BaggagePropagator{},
					},
					{
						B3: B3Propagator{},
					},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}, b3.New()),
			wantErr: false,
		},
		{
			name: "empty composite",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{},
				},
			},
			want:    propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
			wantErr: false,
		},
		// {
		// 	name: "empty propagator name",
		// 	cfg: configOptions{
		// 		opentelemetryConfig: OpenTelemetryConfiguration{
		// 			Propagator: &Propagator{
		// 				Composite: []TextMapPropagator{
		// 					{ptr(""), ptr("tracecontext")},
		// 			},
		// 		},
		// 	},
		// 	want:    propagation.TraceContext{},
		// 	wantErr: false,
		// },
		// {
		// 	name: "nil propagator name",
		// 	cfg: configOptions{
		// 		opentelemetryConfig: OpenTelemetryConfiguration{
		// 			Propagator: &Propagator{
		// 				Composite: []TextMapPropagator{
		// 					{nil, ptr("tracecontext")},
		// 			},
		// 		},
		// 	},
		// 	want:    nil,
		// 	wantErr: true,
		// },
		// {
		// 	name: "unsupported propagator",
		// 	cfg: configOptions{
		// 		opentelemetryConfig: OpenTelemetryConfiguration{
		// 			Propagator: &Propagator{
		// 				Composite: []TextMapPropagator{
		// 					{
		// 						AdditionalProperties: map[string]any {
		// 							"unknown": map[string]string{},
		// 						},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	want:    propagation.NewCompositeTextMapPropagator(),
		// 	wantErr: true,
		// 	errMsg:  "unknown propagator",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newPropagator(tt.cfg)
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
