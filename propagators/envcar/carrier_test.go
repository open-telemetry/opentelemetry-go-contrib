// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar_test

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
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
			ctx = prop.Extract(ctx, &envcar.Carrier{})
			assert.Equal(t, tc.want, trace.SpanContextFromContext(ctx))
		})
	}
}

func TestExtractIgnoresNonNormalizedEnvNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows environment variables are case-insensitive, so this test is not applicable.")
	}

	// Guard against TRACEPARENT/TRACESTATE being set in the outer test environment.
	t.Setenv("TRACEPARENT", "")
	t.Setenv("TRACESTATE", "")

	t.Setenv("traceparent", "00-000000000000007b00000000000001c8-000000000000007b-00")
	t.Setenv("tracestate", "key1=value1,key2=value2")

	ctx := prop.Extract(t.Context(), &envcar.Carrier{})

	assert.False(t, trace.SpanContextFromContext(ctx).IsValid())
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
				SetEnvFunc: func(key, value string) {
					t.Setenv(key, value)
				},
			}

			prop.Inject(ctx, &c)

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
	t.Setenv("envcar_non_normalized_key", "ignored")

	c := envcar.Carrier{}
	keys := c.Keys()

	assert.Contains(t, keys, "TRACEPARENT")
	assert.NotContains(t, keys, "envcar_non_normalized_key")
	assert.NotContains(t, keys, "ENVCAR_NON_NORMALIZED_KEY")
}

func TestCarrierSetNilFunc(_ *testing.T) {
	c := envcar.Carrier{} // SetEnvFunc is nil
	c.Set("key", "value") // should not panic, just no-op
}

func TestCarrierGetNormalizesKey(t *testing.T) {
	t.Setenv("TRACEPARENT", "myvalue")

	c := envcar.Carrier{}
	assert.Equal(t, "myvalue", c.Get("traceparent"))
	assert.Equal(t, "myvalue", c.Get("TRACEPARENT"))
}

func TestCarrierGetIgnoresNonNormalizedEnvNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows environment variables are case-insensitive, so this test is not applicable.")
	}

	t.Setenv("envcar_get_non_normalized_key", "ignored")

	c := envcar.Carrier{}
	assert.Empty(t, c.Get("envcar_get_non_normalized_key"))
	assert.Empty(t, c.Get("ENVCAR_GET_NON_NORMALIZED_KEY"))
}

func TestCarrierSetUppercasesUnderscoresKey(t *testing.T) {
	var gotKey string
	var gotValue string
	c := envcar.Carrier{
		SetEnvFunc: func(key, value string) {
			gotKey = key
			gotValue = value
		},
	}

	c.Set("traceparent", "value")
	assert.Equal(t, "TRACEPARENT", gotKey)
	assert.Equal(t, "value", gotValue)

	c.Set("key with spaces", "value with spaces")
	assert.Equal(t, "KEY_WITH_SPACES", gotKey)
	assert.Equal(t, "value with spaces", gotValue)

	c.Set("Mój Bagaż", "🧳")
	assert.Equal(t, "M_J_BAGA_", gotKey)
	assert.Equal(t, "🧳", gotValue)
}

func TestCarrierGetReadsDirectly(t *testing.T) {
	t.Setenv("TRACEPARENT", "myvalue")
	c := envcar.Carrier{}
	require.Equal(t, "myvalue", c.Get("TRACEPARENT"))

	t.Setenv("TRACEPARENT", "bad")
	t.Setenv("ENVCAR_GET_READS_DIRECTLY_NEW", "value")

	assert.Equal(t, "bad", c.Get("TRACEPARENT"))
	assert.Contains(t, c.Keys(), "ENVCAR_GET_READS_DIRECTLY_NEW")
}

func TestCarrierKeysFetchOnce(t *testing.T) {
	t.Setenv("TRACEPARENT", "myvalue")
	c := envcar.Carrier{}
	require.Contains(t, c.Keys(), "TRACEPARENT")

	t.Setenv("ENVCAR_KEYS_FETCH_ONCE_NEW", "value")

	assert.Contains(t, c.Keys(), "TRACEPARENT")
	assert.NotContains(t, c.Keys(), "ENVCAR_KEYS_FETCH_ONCE_NEW")
}

func TestConcurrentChildProcesses(t *testing.T) {
	// Test that concurrent goroutines can each spawn child processes
	// with their own unique trace context.
	const numGoroutines = 10

	type result struct {
		index int
		want  string
		got   string
		err   error
	}

	results := make(chan result, numGoroutines)
	var wg sync.WaitGroup
	baseCtx := t.Context()

	for i := range numGoroutines {
		wg.Go(func() {
			// Create a unique trace ID for this goroutine.
			traceID := trace.TraceID{byte(i + 1)}
			spanID := trace.SpanID{byte(i + 1)}
			spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			})
			ctx := trace.ContextWithSpanContext(baseCtx, spanCtx)

			// Each goroutine gets its own cmd with its own Env slice.
			cmd := exec.CommandContext(t.Context(), "printenv", "TRACEPARENT")
			cmd.Env = os.Environ()

			// Each goroutine gets its own carrier that writes to its cmd.Env.
			carrier := envcar.Carrier{
				SetEnvFunc: func(key, value string) {
					cmd.Env = append(cmd.Env, key+"="+value)
				},
			}

			// Inject this goroutine's trace context.
			prop := propagation.TraceContext{}
			prop.Inject(ctx, &carrier)

			// Run the child process and capture output.
			out, err := cmd.Output()

			// Expected traceparent format for this goroutine's trace ID.
			expected := "00-" + traceID.String() + "-" + spanID.String() + "-01"

			results <- result{
				index: i,
				want:  expected,
				got:   strings.TrimSpace(string(out)),
				err:   err,
			}
		})
	}

	wg.Wait()
	close(results)

	// Verify each goroutine's child process received the correct trace context.
	for r := range results {
		require.NoError(t, r.err, "goroutine %d failed to run child process", r.index)
		assert.Equal(t, r.want, r.got,
			"goroutine %d: child process received wrong trace context", r.index)
	}
}
