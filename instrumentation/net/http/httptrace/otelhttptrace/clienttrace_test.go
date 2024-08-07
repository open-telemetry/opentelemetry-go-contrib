// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttptrace

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func ExampleNewClientTrace() {
	client := http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return NewClientTrace(ctx)
			}),
		),
	}

	resp, err := client.Get("https://example.com")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	fmt.Println(resp.Status)
}

func Test_clientTracer_end(t *testing.T) {
	t.Run("end called with no parent clientTracer span", func(t *testing.T) {
		fixture := prepareClientTraceTest()
		fixture.ct.end("http.getconn", nil, HTTPConnectionReused.Bool(true), HTTPConnectionWasIdle.Bool(true))
		assert.Len(t, fixture.spanRecorder.Ended(), 0)
	})

	t.Run("end called with no sub spans, no root span, and no errors", func(t *testing.T) {
		fixture := prepareClientTraceTest()
		WithoutSubSpans().apply(fixture.ct)

		ctx, span := fixture.tracer.Start(fixture.ct.Context, "client request")
		fixture.ct.Context = ctx

		fixture.ct.end("http.getconn", nil, HTTPConnectionReused.Bool(true), HTTPConnectionWasIdle.Bool(true))
		span.End()

		require.Len(t, fixture.spanRecorder.Ended(), 1)
		recSpan := fixture.spanRecorder.Ended()[0]

		require.Len(t, recSpan.Events(), 1)
		gotEvent := recSpan.Events()[0]
		assert.Equal(t, "http.getconn.done", gotEvent.Name)

		assert.Equal(t,
			[]attribute.KeyValue{HTTPConnectionReused.Bool(true), HTTPConnectionWasIdle.Bool(true)},
			gotEvent.Attributes,
		)
	})

	t.Run("end called with no sub spans, root span set, and no errors", func(t *testing.T) {
		fixture := prepareClientTraceTest()
		WithoutSubSpans().apply(fixture.ct)

		ctx, span := fixture.tracer.Start(fixture.ct.Context, "client request")
		fixture.ct.Context = ctx
		fixture.ct.root = span

		fixture.ct.end("http.getconn", nil, HTTPConnectionReused.Bool(true), HTTPConnectionWasIdle.Bool(true))
		span.End()

		require.Len(t, fixture.spanRecorder.Ended(), 1)
		recSpan := fixture.spanRecorder.Ended()[0]

		require.Len(t, recSpan.Events(), 1)
		gotEvent := recSpan.Events()[0]
		assert.Equal(t, "http.getconn.done", gotEvent.Name)

		assert.Equal(t,
			[]attribute.KeyValue{
				HTTPConnectionReused.Bool(true),
				HTTPConnectionWasIdle.Bool(true),
			},
			gotEvent.Attributes,
		)
	})

	t.Run("end called with no sub spans, root span set, and error", func(t *testing.T) {
		fixture := prepareClientTraceTest()
		WithoutSubSpans().apply(fixture.ct)

		ctx, span := fixture.tracer.Start(fixture.ct.Context, "client request")
		fixture.ct.Context = ctx
		fixture.ct.root = span

		fixture.ct.end("http.getconn", errors.New("testError"), HTTPConnectionReused.Bool(true), HTTPConnectionWasIdle.Bool(true))
		span.End()

		require.Len(t, fixture.spanRecorder.Ended(), 1)
		recSpan := fixture.spanRecorder.Ended()[0]

		require.Len(t, recSpan.Events(), 1)
		gotEvent := recSpan.Events()[0]
		assert.Equal(t, "http.getconn.done", gotEvent.Name)

		assert.Equal(t,
			[]attribute.KeyValue{
				HTTPConnectionReused.Bool(true),
				HTTPConnectionWasIdle.Bool(true),
				attribute.Key("http.getconn.error").String("testError"),
			},
			gotEvent.Attributes,
		)
	})
}

type clientTraceTestFixture struct {
	spanRecorder *tracetest.SpanRecorder
	tracer       trace.Tracer
	ct           *clientTracer
}

func prepareClientTraceTest() clientTraceTestFixture {
	fixture := clientTraceTestFixture{}
	fixture.spanRecorder = tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(fixture.spanRecorder))
	otel.SetTracerProvider(provider)

	fixture.tracer = provider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version()))

	fixture.ct = &clientTracer{
		Context:         context.Background(),
		tracerProvider:  otel.GetTracerProvider(),
		root:            nil,
		tr:              fixture.tracer,
		activeHooks:     make(map[string]context.Context),
		redactedHeaders: map[string]struct{}{},
		addHeaders:      true,
		useSpans:        true,
	}

	return fixture
}
