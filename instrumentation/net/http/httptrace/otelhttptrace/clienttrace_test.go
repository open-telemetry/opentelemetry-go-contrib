// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package otelhttptrace_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/oteltest"
)

func getSpanFromRecorder(sr *oteltest.SpanRecorder, name string) (*oteltest.Span, bool) {
	for _, s := range sr.Completed() {
		if s.Name() == name {
			return s, true
		}
	}
	return nil, false
}

func getSpansFromRecorder(sr *oteltest.SpanRecorder, name string) []*oteltest.Span {
	ret := []*oteltest.Span{}
	for _, s := range sr.Completed() {
		if s.Name() == name {
			ret = append(ret, s)
		}
	}
	return ret
}

func TestHTTPRequestWithClientTrace(t *testing.T) {
	sr := &oteltest.SpanRecorder{}
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
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
		attributes map[attribute.Key]attribute.Value
		parent     string
	}{
		{
			name: "http.connect",
			attributes: map[attribute.Key]attribute.Value{
				attribute.Key("http.remote"): attribute.StringValue(address.String()),
			},
			parent: "http.getconn",
		},
		{
			name: "http.getconn",
			attributes: map[attribute.Key]attribute.Value{
				attribute.Key("http.remote"): attribute.StringValue(address.String()),
				attribute.Key("http.host"):   attribute.StringValue(address.String()),
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
				assert.Equal(t, span.ParentSpanID(), parent.SpanContext().SpanID())
			}
		}
		if len(tl.attributes) > 0 {
			attrs := span.Attributes()
			if tl.name == "http.getconn" {
				// http.local attribute uses a non-deterministic port.
				local := attribute.Key("http.local")
				assert.Contains(t, attrs, local)
				delete(attrs, local)
			}
			assert.Equal(t, tl.attributes, attrs)
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
		attribute.String("http.remote", "[::1]:3000"),
	}
	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			// sr.Reset()
			sr := &oteltest.SpanRecorder{}
			otel.SetTracerProvider(
				oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr)),
			)
			tt.run(otelhttptrace.NewClientTrace(context.Background()))
			spans := getSpansFromRecorder(sr, "http.connect")
			require.Len(t, spans, 2)

			var gotRemotes []attribute.KeyValue
			for _, span := range spans {
				for k, v := range span.Attributes() {
					gotRemotes = append(gotRemotes, attribute.Any(string(k), v.AsInterface()))
				}
			}
			assert.ElementsMatch(t, expectedRemotes, gotRemotes)
		})
	}
}

func TestEndBeforeStartCreatesSpan(t *testing.T) {
	sr := &oteltest.SpanRecorder{}
	otel.SetTracerProvider(
		oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr)),
	)

	ct := otelhttptrace.NewClientTrace(context.Background())
	ct.DNSDone(httptrace.DNSDoneInfo{})
	ct.DNSStart(httptrace.DNSStartInfo{Host: "example.com"})

	name := "http.dns"
	spans := getSpansFromRecorder(sr, name)
	require.Len(t, spans, 1)
}

func TestHTTPRequestWithTraceContext(t *testing.T) {
	sr := &oteltest.SpanRecorder{}
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

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
	require.Equal(t, parent.SpanContext().SpanID(), getconn.ParentSpanID())
}
