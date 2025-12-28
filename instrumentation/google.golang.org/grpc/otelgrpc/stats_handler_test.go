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
	metricSdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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

func TestNilInstruments(t *testing.T) {
	mp := meterProvider{}
	opts := []Option{WithMeterProvider(mp)}

	ctx := t.Context()

	t.Run("ServerHandler", func(t *testing.T) {
		hIface := NewServerHandler(opts...)
		require.NotNil(t, hIface, "handler")
		require.IsType(t, (*serverHandler)(nil), hIface)

		h := hIface.(*serverHandler)

		assert.NotPanics(t, func() { h.duration.Record(ctx, 0) }, "duration")
		assert.NotPanics(t, func() { h.inSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "inSize")
		assert.NotPanics(t, func() { h.outSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "outSize")
		assert.NotPanics(t, func() { h.inMsg.Record(ctx, 0) }, "inMsg")
		assert.NotPanics(t, func() { h.outMsg.Record(ctx, 0) }, "outMsg")
	})

	t.Run("ClientHandler", func(t *testing.T) {
		hIface := NewClientHandler(opts...)
		require.NotNil(t, hIface, "handler")
		require.IsType(t, (*clientHandler)(nil), hIface)

		h := hIface.(*clientHandler)

		assert.NotPanics(t, func() { h.duration.Record(ctx, 0) }, "duration")
		assert.NotPanics(t, func() { h.inSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "inSize")
		assert.NotPanics(t, func() { h.outSize.RecordSet(ctx, 0, *attribute.EmptySet()) }, "outSize")
		assert.NotPanics(t, func() { h.inMsg.Record(ctx, 0) }, "inMsg")
		assert.NotPanics(t, func() { h.outMsg.Record(ctx, 0) }, "outMsg")
	})
}

func TestServerHandler_SpanAttributesFn_And_MetricAttributesFn(t *testing.T) {
	const ctxKey = "test-key"
	const ctxVal = "test-value"

	mr := metricSdk.NewManualReader()
	meterProvider := metricSdk.NewMeterProvider(metricSdk.WithReader(mr))

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	remoteSpan := trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
	}
	prop := propagation.TraceContext{}

	sc := trace.NewSpanContext(remoteSpan)
	ctx := context.WithValue(t.Context(), ctxKey, ctxVal)
	ctx = trace.ContextWithSpanContext(ctx, sc)

	handler := NewServerHandler(
		WithPropagators(prop),
		WithTracerProvider(provider),
		WithMeterProvider(meterProvider),
		WithSpanAttributes(attribute.String("static", "attr")),
		WithSpanAttributesFn(func(ctx context.Context, ri *stats.RPCTagInfo) []attribute.KeyValue {
			val, _ := ctx.Value(ctxKey).(string)
			return []attribute.KeyValue{attribute.String("dynamic", val)}
		}),
		WithMetricAttributes(attribute.Bool("static", true)),
		WithMetricAttributesFn(func(ctx context.Context, ri *stats.RPCTagInfo) []attribute.KeyValue {
			return []attribute.KeyValue{attribute.Bool("dynamic", true)}
		}),
	)

	info := &stats.RPCTagInfo{FullMethodName: "/foo.bar/Baz", FailFast: true}

	ctx = handler.TagRPC(ctx, info)

	handler.HandleRPC(ctx, &stats.Begin{
		Client:                    false,
		BeginTime:                 time.Time{},
		FailFast:                  true,
		IsClientStream:            false,
		IsServerStream:            false,
		IsTransparentRetryAttempt: false,
	})

	handler.HandleRPC(ctx, &stats.End{
		Client:    false,
		BeginTime: time.Time{},
		EndTime:   time.Time{},
		Trailer:   metadata.MD{},
		Error:     nil,
	})

	// Test span attributes
	assert.NoError(t, spanRecorder.ForceFlush(t.Context()))
	spans := spanRecorder.Ended()

	assert.Contains(t, spans[0].Attributes(), attribute.String("static", "attr"))
	assert.Contains(t, spans[0].Attributes(), attribute.String("dynamic", ctxVal))

	// Test metric attributes
	rm := metricdata.ResourceMetrics{}
	assert.NoError(t, mr.Collect(t.Context(), &rm))

	checkAttrs := func(attrs attribute.Set) {
		val, _ := attrs.Value("static")
		assert.True(t, val.AsBool(), "static attribute should be true")

		val, _ = attrs.Value("dynamic")
		assert.True(t, val.AsBool(), "dynamic attribute should be true")
	}

	for _, mm := range rm.ScopeMetrics[0].Metrics {
		switch data := mm.Data.(type) {
		case metricdata.Histogram[float64]:
			for _, dp := range data.DataPoints {
				checkAttrs(dp.Attributes)
			}
		case metricdata.Histogram[int64]:
			for _, dp := range data.DataPoints {
				checkAttrs(dp.Attributes)
			}
		}
	}
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
