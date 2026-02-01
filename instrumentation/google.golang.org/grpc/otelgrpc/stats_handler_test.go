// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
)

func TestWithPublicEndpoint(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	remoteSpan := trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
		Remote:  true,
	}
	prop := propagation.TraceContext{}
	h := NewServerHandler(
		WithPublicEndpoint(),
		WithPropagators(prop),
		WithTracerProvider(provider),
	)

	sc := trace.NewSpanContext(remoteSpan)
	ctx := trace.ContextWithSpanContext(t.Context(), sc)

	ctx = h.TagRPC(ctx, &stats.RPCTagInfo{
		FullMethodName: "some.package/Method",
		FailFast:       true,
	})

	h.HandleRPC(ctx, &stats.Begin{
		Client:                    false,
		BeginTime:                 time.Time{},
		FailFast:                  true,
		IsClientStream:            false,
		IsServerStream:            false,
		IsTransparentRetryAttempt: false,
	})

	h.HandleRPC(ctx, &stats.End{
		Client:    false,
		BeginTime: time.Time{},
		EndTime:   time.Time{},
		Trailer:   metadata.MD{},
		Error:     nil,
	})

	// Recorded span should be linked with an incoming span context.
	assert.NoError(t, spanRecorder.ForceFlush(ctx))
	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Len(t, spans[0].Links(), 1, "should contain link")
	require.True(t, sc.Equal(spans[0].Links()[0].SpanContext), "should link incoming span context")
}

func TestWithPublicEndpointFn(t *testing.T) {
	remoteSpan := trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}
	prop := propagation.TraceContext{}

	for _, tt := range []struct {
		name          string
		fn            func(context.Context, *stats.RPCTagInfo) bool
		handlerAssert func(*testing.T, trace.SpanContext)
		spansAssert   func(*testing.T, trace.SpanContext, []sdktrace.ReadOnlySpan)
	}{
		{
			name: "with the method returning true",
			fn: func(context.Context, *stats.RPCTagInfo) bool {
				return true
			},
			handlerAssert: func(t *testing.T, sc trace.SpanContext) {
				// Should be with new root trace.
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, sc trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				require.Len(t, spans[0].Links(), 1, "should contain link")
				require.True(t, sc.Equal(spans[0].Links()[0].SpanContext), "should link incoming span context")
			},
		},
		{
			name: "with the method returning false",
			fn: func(context.Context, *stats.RPCTagInfo) bool {
				return false
			},
			handlerAssert: func(t *testing.T, sc trace.SpanContext) {
				// Should have remote span as parent
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.Equal(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, _ trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				require.Empty(t, spans[0].Links(), "should not contain link")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spanRecorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(spanRecorder),
			)

			h := NewServerHandler(
				WithPublicEndpointFn(tt.fn),
				WithPropagators(prop),
				WithTracerProvider(provider),
			)

			sc := trace.NewSpanContext(remoteSpan)
			ctx := trace.ContextWithSpanContext(t.Context(), sc)

			ctx = h.TagRPC(ctx, &stats.RPCTagInfo{
				FullMethodName: "some.package/Method",
				FailFast:       true,
			})

			h.HandleRPC(ctx, &stats.Begin{
				Client:                    false,
				BeginTime:                 time.Time{},
				FailFast:                  true,
				IsClientStream:            false,
				IsServerStream:            false,
				IsTransparentRetryAttempt: false,
			})

			h.HandleRPC(ctx, &stats.End{
				Client:    false,
				BeginTime: time.Time{},
				EndTime:   time.Time{},
				Trailer:   metadata.MD{},
				Error:     nil,
			})

			// Recorded span should be linked with an incoming span context.
			assert.NoError(t, spanRecorder.ForceFlush(ctx))
			spans := spanRecorder.Ended()
			tt.spansAssert(t, sc, spans)
		})
	}
}

func TestNilProviderOption(t *testing.T) {
	// Passing a nil TracerProvider or MeterProvider should not panic and
	// should use the global provider instead.
	t.Run("nil TracerProvider", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = NewClientHandler(WithTracerProvider(nil))
		})
		assert.NotPanics(t, func() {
			_ = NewServerHandler(WithTracerProvider(nil))
		})
	})

	t.Run("nil MeterProvider", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = NewClientHandler(WithMeterProvider(nil))
		})
		assert.NotPanics(t, func() {
			_ = NewServerHandler(WithMeterProvider(nil))
		})
	})

	t.Run("both nil", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = NewClientHandler(WithTracerProvider(nil), WithMeterProvider(nil))
		})
		assert.NotPanics(t, func() {
			_ = NewServerHandler(WithTracerProvider(nil), WithMeterProvider(nil))
		})
	})
}

func TestNilInstruments(t *testing.T) {
	mp := meterProvider{}
	opts := []Option{WithMeterProvider(mp)}

	ctx := t.Context()

	t.Run("ServerHandler", func(t *testing.T) {
		hIface := NewServerHandler(opts...)
		require.NotNil(t, hIface, "handler")
		require.IsType(t, (*serverHandler)(nil), hIface)

		h := hIface.(*serverHandler)

		assert.NotPanics(t, func() { h.duration.Record(ctx, 0, "") }, "duration")
		assert.NotPanics(t, func() { h.inSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "inSize")
		assert.NotPanics(t, func() { h.outSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "outSize")
	})

	t.Run("ClientHandler", func(t *testing.T) {
		hIface := NewClientHandler(opts...)
		require.NotNil(t, hIface, "handler")
		require.IsType(t, (*clientHandler)(nil), hIface)

		h := hIface.(*clientHandler)

		assert.NotPanics(t, func() { h.duration.Record(ctx, 0, "") }, "duration")
		assert.NotPanics(t, func() { h.inSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "inSize")
		assert.NotPanics(t, func() { h.outSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "outSize")
	})
}

type meterProvider struct {
	embedded.MeterProvider
}

func (meterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return meter{}
}

type meter struct {
	// Panic for non-implemented methods.
	metric.Meter
}

func (meter) Int64Histogram(string, ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	return nil, assert.AnError
}

func (meter) Float64Histogram(string, ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return nil, assert.AnError
}
