// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/propagators/envcar"
)

var (
	traceID = trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0x7b, 0, 0, 0, 0, 0, 0, 0x1, 0xc8}
	spanID  = trace.SpanID{0, 0, 0, 0, 0, 0, 0, 0x7b}
	prop    = propagation.TraceContext{}
)

func TestExtractValidTraceContextEnvCarrier(t *testing.T) {
	stateStr := "key1=value1,key2=value2"
	state, err := trace.ParseTraceState(stateStr)
	require.NoError(t, err)

	tests := []struct {
		name string
		envs map[string]string
		want trace.SpanContext
	}{
		{
			name: "sampled",
			envs: map[string]string{
				"TRACEPARENT": "00-000000000000007b00000000000001c8-000000000000007b-01",
			},
			want: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
				Remote:     true,
			}),
		},
		{
			name: "valid tracestate",
			envs: map[string]string{
				"TRACEPARENT": "00-000000000000007b00000000000001c8-000000000000007b-00",
				"TRACESTATE":  stateStr,
			},
			want: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceState: state,
				Remote:     true,
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			ctx = prop.Extract(ctx, envcar.Carrier{})
			assert.Equal(t, tc.want, trace.SpanContextFromContext(ctx))
		})
	}
}

func TestInjectTraceContextEnvCarrier(t *testing.T) {
	stateStr := "key1=value1,key2=value2"
	state, err := trace.ParseTraceState(stateStr)
	require.NoError(t, err)

	tests := []struct {
		name string
		want map[string]string
		sc   trace.SpanContext
	}{
		{
			name: "sampled",
			want: map[string]string{
				"TRACEPARENT": "00-000000000000007b00000000000001c8-000000000000007b-01",
			},
			sc: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
				Remote:     true,
			}),
		},
		{
			name: "with tracestate",
			want: map[string]string{
				"TRACEPARENT": "00-000000000000007b00000000000001c8-000000000000007b-00",
				"TRACESTATE":  stateStr,
			},
			sc: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceState: state,
				Remote:     true,
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			ctx = trace.ContextWithRemoteSpanContext(ctx, tc.sc)
			c := envcar.Carrier{
				SetEnvFunc: func(key, value string) error {
					t.Setenv(key, value)
					return nil
				},
			}

			prop.Inject(ctx, c)

			for k, v := range tc.want {
				if got := os.Getenv(k); got != v {
					t.Errorf("got %s=%s, want %s=%s", k, got, k, v)
				}
			}
		})
	}
}

func TestCarrierKeys(t *testing.T) {
	t.Setenv("TRACEPARENT", "value")

	c := envcar.Carrier{}
	keys := c.Keys()

	// Keys returns lowercased keys
	assert.Contains(t, keys, "traceparent")
}

func TestCarrierSetNilFunc(t *testing.T) {
	c := envcar.Carrier{} // SetEnvFunc is nil
	c.Set("key", "value") // should not panic, just no-op
}

func TestCarrierGetCaseInsensitive(t *testing.T) {
	t.Setenv("TRACEPARENT", "myvalue")

	c := envcar.Carrier{}
	assert.Equal(t, "myvalue", c.Get("traceparent"))
	assert.Equal(t, "myvalue", c.Get("traceparent"))
}

func TestCarrierSetUppercasesKey(t *testing.T) {
	var gotKey string
	c := envcar.Carrier{
		SetEnvFunc: func(key, value string) error {
			gotKey = key
			return nil
		},
	}

	c.Set("traceparent", "value")
	assert.Equal(t, "TRACEPARENT", gotKey)
}
