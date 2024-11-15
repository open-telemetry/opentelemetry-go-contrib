// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func getSpanFromRecorder(sr *tracetest.SpanRecorder, name string) (trace.ReadOnlySpan, bool) {
	for _, s := range sr.Ended() {
		if s.Name() == name {
			return s, true
		}
	}
	return nil, false
}

func getSpansFromRecorder(sr *tracetest.SpanRecorder, name string) []trace.ReadOnlySpan {
	var ret []trace.ReadOnlySpan
	for _, s := range sr.Ended() {
		if s.Name() == name {
			ret = append(ret, s)
		}
	}
	return ret
}

func TestHTTPRequestWithClientTrace(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	tr := tp.Tracer("httptrace/client")

	// Mock http server
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		}),
	)
	defer ts.Close()
	address := ts.Listener.Addr()

	client := ts.Client()
	err := func(ctx context.Context) error {
		ctx, span := tr.Start(ctx, "test")
		defer span.End()
		req, _ := http.NewRequest("GET", ts.URL, nil)
		_, req = otelhttptrace.W3C(ctx, req)

		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %s", err.Error())
		}
		_ = res.Body.Close()

		return nil
	}(context.Background())
	if err != nil {
		panic("unexpected error in http request: " + err.Error())
	}

	testLen := []struct {
		name       string
		attributes []attribute.KeyValue
		parent     string
	}{
		{
			name: "http.connect",
			attributes: []attribute.KeyValue{
				attribute.Key("http.conn.done.addr").String(address.String()),
				attribute.Key("http.conn.done.network").String("tcp"),
				attribute.Key("http.conn.start.network").String("tcp"),
				attribute.Key("http.remote").String(address.String()),
			},
			parent: "http.getconn",
		},
		{
			name: "http.getconn",
			attributes: []attribute.KeyValue{
				attribute.Key("http.remote").String(address.String()),
				attribute.Key("net.host.name").String(address.String()),
				attribute.Key("http.conn.reused").Bool(false),
				attribute.Key("http.conn.wasidle").Bool(false),
			},
			parent: "test",
		},
		{
			name:   "http.receive",
			parent: "test",
		},
		{
			name:   "http.headers",
			parent: "test",
		},
		{
			name:   "http.send",
			parent: "test",
		},
		{
			name: "test",
		},
	}
	for _, tl := range testLen {
		span, ok := getSpanFromRecorder(sr, tl.name)
		if !assert.True(t, ok) {
			continue
		}

		if tl.parent != "" {
			parent, ok := getSpanFromRecorder(sr, tl.parent)
			if assert.True(t, ok) {
				assert.Equal(t, span.Parent().SpanID(), parent.SpanContext().SpanID())
			}
		}
		if len(tl.attributes) > 0 {
			attrs := span.Attributes()
			if tl.name == "http.getconn" {
				// http.local attribute uses a non-deterministic port.
				local := attribute.Key("http.local")
				var contains bool
				for i, a := range attrs {
					if a.Key == local {
						attrs = append(attrs[:i], attrs[i+1:]...)
						contains = true
						break
					}
				}
				assert.True(t, contains, "missing http.local attribute")
			}
			assert.ElementsMatch(t, tl.attributes, attrs)
		}
	}
}

func TestConcurrentConnectionStart(t *testing.T) {
	tts := []struct {
		name string
		run  func(*httptrace.ClientTrace)
	}{
		{
			name: "Open1Close1Open2Close2",
			run: func(ct *httptrace.ClientTrace) {
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
			},
		},
		{
			name: "Open2Close2Open1Close1",
			run: func(ct *httptrace.ClientTrace) {
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
			},
		},
		{
			name: "Open1Open2Close1Close2",
			run: func(ct *httptrace.ClientTrace) {
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
				ct.ConnectDone("tcp", "[::1]:3000", nil)
			},
		},
		{
			name: "Open1Open2Close2Close1",
			run: func(ct *httptrace.ClientTrace) {
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
			},
		},
		{
			name: "Open2Open1Close1Close2",
			run: func(ct *httptrace.ClientTrace) {
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
				ct.ConnectDone("tcp", "[::1]:3000", nil)
			},
		},
		{
			name: "Open2Open1Close2Close1",
			run: func(ct *httptrace.ClientTrace) {
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
			},
		},
	}

	expectedRemotes := []attribute.KeyValue{
		attribute.String("http.remote", "127.0.0.1:3000"),
		attribute.String("http.conn.start.network", "tcp"),
		attribute.String("http.conn.done.addr", "127.0.0.1:3000"),
		attribute.String("http.conn.done.network", "tcp"),
		attribute.String("http.remote", "[::1]:3000"),
		attribute.String("http.conn.start.network", "tcp"),
		attribute.String("http.conn.done.addr", "[::1]:3000"),
		attribute.String("http.conn.done.network", "tcp"),
	}
	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
			otel.SetTracerProvider(tp)
			tt.run(otelhttptrace.NewClientTrace(context.Background()))
			spans := getSpansFromRecorder(sr, "http.connect")
			require.Len(t, spans, 2)

			var gotRemotes []attribute.KeyValue
			for _, span := range spans {
				gotRemotes = append(gotRemotes, span.Attributes()...)
			}
			assert.ElementsMatch(t, expectedRemotes, gotRemotes)
		})
	}
}

func TestEndBeforeStartCreatesSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)

	ct := otelhttptrace.NewClientTrace(context.Background())
	ct.DNSDone(httptrace.DNSDoneInfo{})
	ct.DNSStart(httptrace.DNSStartInfo{Host: "example.com"})

	name := "http.dns"
	spans := getSpansFromRecorder(sr, name)
	require.Len(t, spans, 1)
}

func TestEndBeforeStartWithoutSubSpansDoesNotPanic(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)

	ct := otelhttptrace.NewClientTrace(context.Background(), otelhttptrace.WithoutSubSpans())

	require.NotPanics(t, func() {
		ct.DNSDone(httptrace.DNSDoneInfo{})
	})

	// no spans created because we were just using background context without span
	// and Start wasn't called which would have started a span
	require.Empty(t, sr.Ended())
}

type clientTraceTestFixture struct {
	Address      string
	URL          string
	Client       *http.Client
	SpanRecorder *tracetest.SpanRecorder
}

func prepareClientTraceTest(t *testing.T) clientTraceTestFixture {
	fixture := clientTraceTestFixture{}
	fixture.SpanRecorder = tracetest.NewSpanRecorder()
	otel.SetTracerProvider(
		trace.NewTracerProvider(trace.WithSpanProcessor(fixture.SpanRecorder)),
	)

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		}),
	)
	t.Cleanup(ts.Close)
	fixture.Client = ts.Client()
	fixture.URL = ts.URL
	fixture.Address = ts.Listener.Addr().String()
	return fixture
}

func TestWithoutSubSpans(t *testing.T) {
	fixture := prepareClientTraceTest(t)

	ctx := context.Background()
	ctx = httptrace.WithClientTrace(ctx,
		otelhttptrace.NewClientTrace(ctx,
			otelhttptrace.WithoutSubSpans(),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, nil)
	require.NoError(t, err)
	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	// no spans created because we were just using background context without span
	require.Empty(t, fixture.SpanRecorder.Ended())

	// Start again with a "real" span in the context, now tracing should add
	// events and annotations.
	ctx, span := otel.Tracer("oteltest").Start(context.Background(), "root")
	ctx = httptrace.WithClientTrace(ctx,
		otelhttptrace.NewClientTrace(ctx,
			otelhttptrace.WithoutSubSpans(),
		),
	)
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, nil)
	req.Header.Set("User-Agent", "oteltest/1.1")
	req.Header.Set("Authorization", "Bearer token123")
	require.NoError(t, err)
	resp, err = fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	span.End()
	// we just have the one span we created
	require.Len(t, fixture.SpanRecorder.Ended(), 1)
	recSpan := fixture.SpanRecorder.Ended()[0]

	gotAttributes := recSpan.Attributes()
	require.Len(t, gotAttributes, 4)
	assert.Equal(t,
		[]attribute.KeyValue{
			attribute.Key("http.request.header.host").String(fixture.Address),
			attribute.Key("http.request.header.user-agent").String("oteltest/1.1"),
			attribute.Key("http.request.header.authorization").String("****"),
			attribute.Key("http.request.header.accept-encoding").String("gzip"),
		},
		gotAttributes,
	)

	type attrMap = map[attribute.Key]attribute.Value
	expectedEvents := []struct {
		Event       string
		VerifyAttrs func(t *testing.T, got attrMap)
	}{
		{"http.getconn.start", func(t *testing.T, got attrMap) {
			assert.Equal(t,
				attribute.StringValue(fixture.Address),
				got[attribute.Key("net.host.name")],
			)
		}},
		{"http.getconn.done", func(t *testing.T, got attrMap) {
			// value is dynamic, just verify we have the attribute
			assert.Contains(t, got, attribute.Key("http.conn.idletime"))
			assert.Equal(t,
				attribute.BoolValue(true),
				got[attribute.Key("http.conn.reused")],
			)
			assert.Equal(t,
				attribute.BoolValue(true),
				got[attribute.Key("http.conn.wasidle")],
			)
			assert.Equal(t,
				attribute.StringValue(fixture.Address),
				got[attribute.Key("http.remote")],
			)
			// value is dynamic, just verify we have the attribute
			assert.Contains(t, got, attribute.Key("http.local"))
		}},
		{"http.send.start", nil},
		{"http.send.done", nil},
		{"http.receive.start", nil},
		{"http.receive.done", nil},
	}
	require.Len(t, recSpan.Events(), len(expectedEvents))
	for i, e := range recSpan.Events() {
		attrs := attrMap{}
		for _, a := range e.Attributes {
			attrs[a.Key] = a.Value
		}
		expected := expectedEvents[i]
		assert.Equal(t, expected.Event, e.Name)
		if expected.VerifyAttrs == nil {
			assert.Nil(t, e.Attributes, "Event %q has no attributes", e.Name)
		} else {
			e := e // make loop var lexical
			t.Run(e.Name, func(t *testing.T) {
				expected.VerifyAttrs(t, attrs)
			})
		}
	}
}

func TestWithRedactedHeaders(t *testing.T) {
	fixture := prepareClientTraceTest(t)

	ctx, span := otel.Tracer("oteltest").Start(context.Background(), "root")
	ctx = httptrace.WithClientTrace(ctx,
		otelhttptrace.NewClientTrace(ctx,
			otelhttptrace.WithoutSubSpans(),
			otelhttptrace.WithRedactedHeaders("user-agent"),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, nil)
	require.NoError(t, err)
	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	span.End()
	require.Len(t, fixture.SpanRecorder.Ended(), 1)
	recSpan := fixture.SpanRecorder.Ended()[0]

	gotAttributes := recSpan.Attributes()
	assert.Equal(t,
		[]attribute.KeyValue{
			attribute.Key("http.request.header.host").String(fixture.Address),
			attribute.Key("http.request.header.user-agent").String("****"),
			attribute.Key("http.request.header.accept-encoding").String("gzip"),
		},
		gotAttributes,
	)
}

func TestWithoutHeaders(t *testing.T) {
	fixture := prepareClientTraceTest(t)

	ctx, span := otel.Tracer("oteltest").Start(context.Background(), "root")
	ctx = httptrace.WithClientTrace(ctx,
		otelhttptrace.NewClientTrace(ctx,
			otelhttptrace.WithoutSubSpans(),
			otelhttptrace.WithoutHeaders(),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, nil)
	require.NoError(t, err)
	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	span.End()
	require.Len(t, fixture.SpanRecorder.Ended(), 1)
	recSpan := fixture.SpanRecorder.Ended()[0]

	gotAttributes := recSpan.Attributes()
	require.Empty(t, gotAttributes)
}

func TestWithInsecureHeaders(t *testing.T) {
	fixture := prepareClientTraceTest(t)

	ctx, span := otel.Tracer("oteltest").Start(context.Background(), "root")
	ctx = httptrace.WithClientTrace(ctx,
		otelhttptrace.NewClientTrace(ctx,
			otelhttptrace.WithoutSubSpans(),
			otelhttptrace.WithInsecureHeaders(),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, nil)
	req.Header.Set("User-Agent", "oteltest/1.1")
	req.Header.Set("Authorization", "Bearer token123")
	require.NoError(t, err)
	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	span.End()
	require.Len(t, fixture.SpanRecorder.Ended(), 1)
	recSpan := fixture.SpanRecorder.Ended()[0]

	gotAttributes := recSpan.Attributes()
	assert.Equal(t,
		[]attribute.KeyValue{
			attribute.Key("http.request.header.host").String(fixture.Address),
			attribute.Key("http.request.header.user-agent").String("oteltest/1.1"),
			attribute.Key("http.request.header.authorization").String("Bearer token123"),
			attribute.Key("http.request.header.accept-encoding").String("gzip"),
		},
		gotAttributes,
	)
}

func TestHTTPRequestWithTraceContext(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	// Mock http server
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		}),
	)
	defer ts.Close()

	ctx, span := tp.Tracer("").Start(context.Background(), "parent_span")

	req, _ := http.NewRequest("GET", ts.URL, nil)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), otelhttptrace.NewClientTrace(ctx)))

	client := ts.Client()
	res, err := client.Do(req)
	require.NoError(t, err)
	_ = res.Body.Close()

	span.End()

	parent, ok := getSpanFromRecorder(sr, "parent_span")
	require.True(t, ok)

	getconn, ok := getSpanFromRecorder(sr, "http.getconn")
	require.True(t, ok)

	require.Equal(t, parent.SpanContext().TraceID(), getconn.SpanContext().TraceID())
	require.Equal(t, parent.SpanContext().SpanID(), getconn.Parent().SpanID())
}

func TestHTTPRequestWithExpect100Continue(t *testing.T) {
	fixture := prepareClientTraceTest(t)

	ctx, span := otel.Tracer("oteltest").Start(context.Background(), "root")
	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fixture.URL, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	// Set Expect: 100-continue
	req.Header.Set("Expect", "100-continue")
	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	span.End()

	// Wait for http.send span as per https://pkg.go.dev/net/http/httptrace#ClientTrace:
	// Functions may be called concurrently from different goroutines and some may be called
	// after the request has completed
	var httpSendSpan trace.ReadOnlySpan
	require.Eventually(t, func() bool {
		var ok bool
		httpSendSpan, ok = getSpanFromRecorder(fixture.SpanRecorder, "http.send")
		return ok
	}, 5*time.Second, 10*time.Millisecond)

	// Found http.send span must contain "GOT 100 - Wait" event
	found := false
	for _, v := range httpSendSpan.Events() {
		if v.Name == "GOT 100 - Wait" {
			found = true
			break
		}
	}
	require.True(t, found)
}
