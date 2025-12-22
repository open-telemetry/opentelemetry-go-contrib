// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropagator(t *testing.T) {
	tests := []struct {
		name    string
		cfg     OpenTelemetryConfigurationPropagator
		want    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil propagator config",
			cfg:     nil,
			want:    []string{},
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
			want:    []string{"traceparent", "tracestate"},
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
			want:    []string{"baggage"},
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
			want:    []string{"x-b3-traceid", "x-b3-spanid", "x-b3-sampled", "x-b3-flags"},
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
			want:    []string{"x-b3-traceid", "x-b3-spanid", "x-b3-sampled", "x-b3-flags"},
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
			want:    []string{"uber-trace-id"},
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
			want:    []string{"ot-tracer-traceid", "ot-tracer-spanid", "ot-tracer-sampled"},
			wantErr: false,
		},
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
			want:    []string{"tracestate", "baggage", "x-b3-traceid", "x-b3-spanid", "x-b3-sampled", "x-b3-flags", "traceparent"},
			wantErr: false,
		},
		{
			name: "empty composite",
			cfg: &PropagatorJson{
				Composite: []TextMapPropagator{
					{},
				},
			},
			want:    []string{"tracestate", "baggage", "traceparent"},
			wantErr: false,
		},
		{
			name: "multiple propagators via composite_list",
			cfg: &PropagatorJson{
				CompositeList: ptr("tracecontext,baggage,b3"),
			},
			want:    []string{"tracestate", "baggage", "x-b3-traceid", "x-b3-spanid", "x-b3-sampled", "x-b3-flags", "traceparent"},
			wantErr: false,
		},
		{
			name: "valid xray",
			cfg: &PropagatorJson{
				CompositeList: ptr("xray"),
			},
			want:    []string{"X-Amzn-Trace-Id"},
			wantErr: false,
		},
		{
			name: "empty propagator name",
			cfg: &PropagatorJson{
				CompositeList: ptr(""),
			},
			want:    []string{},
			wantErr: true,
			errMsg:  "unknown propagator",
		},
		{
			name: "unsupported propagator",
			cfg: &PropagatorJson{
				CompositeList: ptr("random-garbage,baggage,b3"),
			},
			want:    []string{},
			wantErr: true,
			errMsg:  "unknown propagator",
		},
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
			slices.Sort(tt.want)
			gotFields := got.Fields()
			slices.Sort(gotFields)
			assert.Equal(t, tt.want, gotFields)
		})
	}
}
