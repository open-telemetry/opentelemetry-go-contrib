// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttptrace_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
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
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		}),
	)
	defer ts.Close()
	address := ts.Listener.Addr()

	client := ts.Client()
	err := func(ctx context.Context) error {
		ctx, span := tr.Start(ctx, "test")
		defer span.End()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, http.NoBody)
		_, req = otelhttptrace.W3C(ctx, req)

		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %s", err.Error())
		}
		_ = res.Body.Close()

		return nil
	}(t.Context())
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
				attribute.Key("server.address").String(address.String()),
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
			tt.run(otelhttptrace.NewClientTrace(t.Context()))
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

	ct := otelhttptrace.NewClientTrace(t.Context())
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

	ct := otelhttptrace.NewClientTrace(t.Context(), otelhttptrace.WithoutSubSpans())

	require.NotPanics(t, func() {
		ct.DNSDone(httptrace.DNSDoneInfo{})
	})

	// no spans created because we were just using background context without span
	// and Start wasn't called which would have started a span
	require.Empty(t, sr.Ended())
}

func TestNoClientTraceCallGuarantee(t *testing.T) {
	t.Run("Got100Continue", func(t *testing.T) {
		// It is possible that Got100Continue is called before GotFirstResponseByte.
		// Also as there is no guarantee provided in the ClientTrace docs that GotFirstResponseByte should be called before
		// Got100Continue this edge case should be covered.
		assert.NotPanics(t, func() {
			clientTrace := otelhttptrace.NewClientTrace(t.Context())
			clientTrace.Got100Continue()
		})
	})
	t.Run("Got1xxResponse", func(t *testing.T) {
		clientTrace := otelhttptrace.NewClientTrace(t.Context())
		err := clientTrace.Got1xxResponse(http.StatusNoContent, nil)
		assert.NoError(t, err)
	})
	t.Run("Wait100Continue", func(t *testing.T) {
		assert.NotPanics(t, func() {
			clientTrace := otelhttptrace.NewClientTrace(t.Context())
			clientTrace.Wait100Continue()
		})
	})
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
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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

	ctx := t.Context()
	ctx = httptrace.WithClientTrace(
		ctx,
		otelhttptrace.NewClientTrace(
			ctx,
			otelhttptrace.WithoutSubSpans(),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, http.NoBody)
	require.NoError(t, err)
	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	// no spans created because we were just using background context without span
	require.Empty(t, fixture.SpanRecorder.Ended())

	// Start again with a "real" span in the context, now tracing should add
	// events and annotations.
	ctx, span := otel.Tracer("oteltest").Start(t.Context(), "root")
	ctx = httptrace.WithClientTrace(
		ctx,
		otelhttptrace.NewClientTrace(
			ctx,
			otelhttptrace.WithoutSubSpans(),
		),
	)
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, http.NoBody)
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
	assert.Equal(
		t,
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
			assert.Equal(
				t,
				attribute.StringValue(fixture.Address),
				got[attribute.Key("server.address")],
			)
		}},
		{"http.getconn.done", func(t *testing.T, got attrMap) {
			// value is dynamic, just verify we have the attribute
			assert.Contains(t, got, attribute.Key("http.conn.idletime"))
			assert.Equal(
				t,
				attribute.BoolValue(true),
				got[attribute.Key("http.conn.reused")],
			)
			assert.Equal(
				t,
				attribute.BoolValue(true),
				got[attribute.Key("http.conn.wasidle")],
			)
			assert.Equal(
				t,
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

	ctx, span := otel.Tracer("oteltest").Start(t.Context(), "root")
	ctx = httptrace.WithClientTrace(
		ctx,
		otelhttptrace.NewClientTrace(
			ctx,
			otelhttptrace.WithoutSubSpans(),
			otelhttptrace.WithRedactedHeaders("user-agent"),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, http.NoBody)
	require.NoError(t, err)
	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	span.End()
	require.Len(t, fixture.SpanRecorder.Ended(), 1)
	recSpan := fixture.SpanRecorder.Ended()[0]

	gotAttributes := recSpan.Attributes()
	assert.Equal(
		t,
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

	ctx, span := otel.Tracer("oteltest").Start(t.Context(), "root")
	ctx = httptrace.WithClientTrace(
		ctx,
		otelhttptrace.NewClientTrace(
			ctx,
			otelhttptrace.WithoutSubSpans(),
			otelhttptrace.WithoutHeaders(),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, http.NoBody)
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

	ctx, span := otel.Tracer("oteltest").Start(t.Context(), "root")
	ctx = httptrace.WithClientTrace(
		ctx,
		otelhttptrace.NewClientTrace(
			ctx,
			otelhttptrace.WithoutSubSpans(),
			otelhttptrace.WithInsecureHeaders(),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, http.NoBody)
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
	assert.Equal(
		t,
		[]attribute.KeyValue{
			attribute.Key("http.request.header.host").String(fixture.Address),
			attribute.Key("http.request.header.user-agent").String("oteltest/1.1"),
			attribute.Key("http.request.header.authorization").String("Bearer token123"),
			attribute.Key("http.request.header.accept-encoding").String("gzip"),
		},
		gotAttributes,
	)
}

func TestSubSpansHeaderAttributes(t *testing.T) {
	fixture := prepareClientTraceTest(t)

	ctx, span := otel.Tracer("oteltest").Start(t.Context(), "root")
	ctx = httptrace.WithClientTrace(
		ctx,
		otelhttptrace.NewClientTrace(ctx), // default: with sub-spans
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.URL, http.NoBody)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "oteltest/1.1")
	req.Header.Set("Authorization", "Bearer token123")

	resp, err := fixture.Client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	span.End()

	// wroteHeaderField() sets header attributes via ct.root.SetAttributes().
	// Collect all attributes across all ended spans to verify they are recorded.
	allAttrs := make(map[attribute.Key]attribute.Value)
	for _, s := range fixture.SpanRecorder.Ended() {
		for _, a := range s.Attributes() {
			allAttrs[a.Key] = a.Value
		}
	}

	assert.Contains(t, allAttrs, attribute.Key("http.request.header.host"),
		"header attribute should be recorded on a span")
	assert.Contains(t, allAttrs, attribute.Key("http.request.header.user-agent"),
		"header attribute should be recorded on a span")
	assert.Contains(t, allAttrs, attribute.Key("http.request.header.authorization"),
		"header attribute should be recorded on a span (redacted)")
}

func TestHTTPRequestWithTraceContext(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	// Mock http server
	ts := httptest.NewServer(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		}),
	)
	defer ts.Close()

	ctx, span := tp.Tracer("").Start(t.Context(), "parent_span")

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL, http.NoBody)
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

	ctx, span := otel.Tracer("oteltest").Start(t.Context(), "root")
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

func TestWithTracerProvider(t *testing.T) {
	t.Run("CustomTracerProvider", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

		globalSR := tracetest.NewSpanRecorder()
		globalTP := trace.NewTracerProvider(trace.WithSpanProcessor(globalSR))
		otel.SetTracerProvider(globalTP)

		ct := otelhttptrace.NewClientTrace(
			t.Context(),
			otelhttptrace.WithTracerProvider(tp),
		)
		ct.DNSStart(httptrace.DNSStartInfo{Host: "example.com"})
		ct.DNSDone(httptrace.DNSDoneInfo{})

		spans := getSpansFromRecorder(sr, "http.dns")
		require.Len(t, spans, 1, "expected span in custom tracer provider")
		require.Empty(t, globalSR.Ended(), "expected no spans in global tracer provider")
	})

	t.Run("NilTracerProviderIgnored", func(t *testing.T) {
		globalSR := tracetest.NewSpanRecorder()
		globalTP := trace.NewTracerProvider(trace.WithSpanProcessor(globalSR))
		otel.SetTracerProvider(globalTP)

		ct := otelhttptrace.NewClientTrace(
			t.Context(),
			otelhttptrace.WithTracerProvider(nil),
		)
		ct.DNSStart(httptrace.DNSStartInfo{Host: "example.com"})
		ct.DNSDone(httptrace.DNSDoneInfo{})

		spans := getSpansFromRecorder(globalSR, "http.dns")
		require.Len(t, spans, 1, "nil provider should fall back to global provider")
	})
}

func TestTLSHandshake(t *testing.T) {
	testCases := []struct {
		name          string
		withoutSpans  bool
		handshakeErr  error
		expectedError bool
	}{
		{
			name:          "SuccessWithSubSpans",
			withoutSpans:  false,
			handshakeErr:  nil,
			expectedError: false,
		},
		{
			name:          "ErrorWithSubSpans",
			withoutSpans:  false,
			handshakeErr:  errors.New("tls: handshake failed"),
			expectedError: true,
		},
		{
			name:          "SuccessWithoutSubSpans",
			withoutSpans:  true,
			handshakeErr:  nil,
			expectedError: false,
		},
		{
			name:          "ErrorWithoutSubSpans",
			withoutSpans:  true,
			handshakeErr:  errors.New("tls: bad certificate"),
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
			otel.SetTracerProvider(tp)

			opts := []otelhttptrace.ClientTraceOption{}
			if tc.withoutSpans {
				opts = append(opts, otelhttptrace.WithoutSubSpans())
			}

			ctx, root := tp.Tracer("test").Start(t.Context(), "root")
			ct := otelhttptrace.NewClientTrace(ctx, opts...)

			ct.TLSHandshakeStart()
			ct.TLSHandshakeDone(tls.ConnectionState{}, tc.handshakeErr)

			root.End()

			if !tc.withoutSpans {
				span, ok := getSpanFromRecorder(sr, "http.tls")
				require.True(t, ok, "http.tls span should be created")
				if tc.expectedError {
					assert.Equal(t, codes.Error, span.Status().Code)
					assert.Equal(t, tc.handshakeErr.Error(), span.Status().Description)
				} else {
					assert.Equal(t, codes.Unset, span.Status().Code)
				}
			} else {
				// In withoutSubSpans mode, events should be recorded on root span.
				rootSpan, ok := getSpanFromRecorder(sr, "root")
				require.True(t, ok, "root span must exist")

				eventNames := make([]string, 0, len(rootSpan.Events()))
				for _, e := range rootSpan.Events() {
					eventNames = append(eventNames, e.Name)
				}
				assert.Contains(t, eventNames, "http.tls.start")
				assert.Contains(t, eventNames, "http.tls.done")

				if tc.expectedError {
					for _, e := range rootSpan.Events() {
						if e.Name != "http.tls.done" {
							continue
						}
						var hasErrMsg bool
						for _, attr := range e.Attributes {
							if attr.Key == attribute.Key("http.tls.error") && attr.Value.AsString() == tc.handshakeErr.Error() {
								hasErrMsg = true
								break
							}
						}
						assert.True(t, hasErrMsg, "expected error attribute on http.tls.done event")
					}
				}
			}
		})
	}
}

func TestHTTPSRequestWithClientTrace(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, root := tp.Tracer("test").Start(t.Context(), "root")
	client := ts.Client()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), otelhttptrace.NewClientTrace(ctx)))

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	root.End()

	tlsSpan, ok := getSpanFromRecorder(sr, "http.tls")
	require.True(t, ok, "expected http.tls span for HTTPS request")
	assert.Equal(t, codes.Unset, tlsSpan.Status().Code)
}

func TestClientTraceErrorAndInformationalBranches(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T, ct *httptrace.ClientTrace, sr *tracetest.SpanRecorder)
	}{
		{
			name: "DNSDoneWithError",
			run: func(t *testing.T, ct *httptrace.ClientTrace, sr *tracetest.SpanRecorder) {
				dnsErr := errors.New("dns lookup failed: no such host")
				ct.DNSStart(httptrace.DNSStartInfo{Host: "invalid.domain"})
				ct.DNSDone(httptrace.DNSDoneInfo{Err: dnsErr})

				span, ok := getSpanFromRecorder(sr, "http.dns")
				require.True(t, ok)
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, dnsErr.Error(), span.Status().Description)
			},
		},
		{
			name: "WroteRequestWithError",
			run: func(t *testing.T, ct *httptrace.ClientTrace, sr *tracetest.SpanRecorder) {
				writeErr := errors.New("broken pipe writing body")
				ct.WroteHeaders()
				ct.WroteRequest(httptrace.WroteRequestInfo{Err: writeErr})

				span, ok := getSpanFromRecorder(sr, "http.send")
				require.True(t, ok)
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, writeErr.Error(), span.Status().Description)
			},
		},
		{
			name: "Got1xxResponseWithHeaders",
			run: func(t *testing.T, ct *httptrace.ClientTrace, sr *tracetest.SpanRecorder) {
				ct.GotFirstResponseByte()

				headers := textproto.MIMEHeader{
					"Link": []string{"</style.css>; rel=preload"},
				}
				err := ct.Got1xxResponse(103, headers)
				require.NoError(t, err)

				ct.PutIdleConn(nil)

				span, ok := getSpanFromRecorder(sr, "http.receive")
				require.True(t, ok)

				var foundEvent bool
				for _, ev := range span.Events() {
					if ev.Name != "GOT 1xx" {
						continue
					}
					foundEvent = true
					var hasStatus, hasHeader bool
					for _, attr := range ev.Attributes {
						if attr.Key == attribute.Key("http.status") && attr.Value.AsInt64() == 103 {
							hasStatus = true
						}
						if attr.Key == attribute.Key("http.mime") && attr.Value.AsString() == "Link=</style.css>; rel=preload" {
							hasHeader = true
						}
					}
					assert.True(t, hasStatus, "expected http.status=103 in GOT 1xx event")
					assert.True(t, hasHeader, "expected parsed MIME header in GOT 1xx event")
					break
				}
				assert.True(t, foundEvent, "expected 'GOT 1xx' event on http.receive span")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
			otel.SetTracerProvider(tp)

			ctx, root := tp.Tracer("test").Start(t.Context(), "root")
			ct := otelhttptrace.NewClientTrace(ctx)

			tc.run(t, ct, sr)
			root.End()
		})
	}
}
