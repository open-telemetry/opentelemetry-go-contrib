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
package httptrace_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	nhtrace "net/http/httptrace"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace/tracetest"
	"go.opentelemetry.io/otel/label"
)

type SpanRecorder map[string]*tracetest.Span

func (sr *SpanRecorder) OnStart(span *tracetest.Span) {}
func (sr *SpanRecorder) OnEnd(span *tracetest.Span)   { (*sr)[span.Name()] = span }

func TestHTTPRequestWithClientTrace(t *testing.T) {
	sr := SpanRecorder{}
	tp := tracetest.NewProvider(tracetest.WithSpanRecorder(&sr))
	global.SetTraceProvider(tp)
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
		_, req = httptrace.W3C(ctx, req)

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
		attributes map[label.Key]label.Value
		parent     string
	}{
		{
			name: "http.connect",
			attributes: map[label.Key]label.Value{
				label.Key("http.remote"): label.StringValue(address.String()),
			},
			parent: "http.getconn",
		},
		{
			name: "http.getconn",
			attributes: map[label.Key]label.Value{
				label.Key("http.remote"): label.StringValue(address.String()),
				label.Key("http.host"):   label.StringValue(address.String()),
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
		if !assert.Contains(t, sr, tl.name) {
			continue
		}
		span := sr[tl.name]
		if tl.parent != "" {
			if assert.Contains(t, sr, tl.parent) {
				assert.Equal(t, span.ParentSpanID(), sr[tl.parent].SpanContext().SpanID)
			}
		}
		if len(tl.attributes) > 0 {
			attrs := span.Attributes()
			if tl.name == "http.getconn" {
				// http.local attribute uses a non-deterministic port.
				local := label.Key("http.local")
				assert.Contains(t, attrs, local)
				delete(attrs, local)
			}
			assert.Equal(t, tl.attributes, attrs)
		}
	}
}

type MultiSpanRecorder map[string][]*tracetest.Span

func (sr *MultiSpanRecorder) Reset()                       { (*sr) = MultiSpanRecorder{} }
func (sr *MultiSpanRecorder) OnStart(span *tracetest.Span) {}
func (sr *MultiSpanRecorder) OnEnd(span *tracetest.Span) {
	(*sr)[span.Name()] = append((*sr)[span.Name()], span)
}

func TestConcurrentConnectionStart(t *testing.T) {
	sr := MultiSpanRecorder{}
	global.SetTraceProvider(
		tracetest.NewProvider(tracetest.WithSpanRecorder(&sr)),
	)
	ct := httptrace.NewClientTrace(context.Background())
	tts := []struct {
		name string
		run  func()
	}{
		{
			name: "Open1Close1Open2Close2",
			run: func() {
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
			},
		},
		{
			name: "Open2Close2Open1Close1",
			run: func() {
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
			},
		},
		{
			name: "Open1Open2Close1Close2",
			run: func() {
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
				ct.ConnectDone("tcp", "[::1]:3000", nil)
			},
		},
		{
			name: "Open1Open2Close2Close1",
			run: func() {
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
			},
		},
		{
			name: "Open2Open1Close1Close2",
			run: func() {
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
				ct.ConnectDone("tcp", "[::1]:3000", nil)
			},
		},
		{
			name: "Open2Open1Close2Close1",
			run: func() {
				ct.ConnectStart("tcp", "[::1]:3000")
				ct.ConnectStart("tcp", "127.0.0.1:3000")
				ct.ConnectDone("tcp", "[::1]:3000", nil)
				ct.ConnectDone("tcp", "127.0.0.1:3000", nil)
			},
		},
	}

	expectedRemotes := []label.KeyValue{
		label.String("http.remote", "127.0.0.1:3000"),
		label.String("http.remote", "[::1]:3000"),
	}
	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			sr.Reset()
			tt.run()
			spans := sr["http.connect"]
			require.Len(t, spans, 2)

			var gotRemotes []label.KeyValue
			for _, span := range spans {
				for k, v := range span.Attributes() {
					gotRemotes = append(gotRemotes, label.Any(string(k), v.AsInterface()))
				}
			}
			assert.ElementsMatch(t, expectedRemotes, gotRemotes)
		})
	}
}

func TestEndBeforeStartCreatesSpan(t *testing.T) {
	sr := MultiSpanRecorder{}
	global.SetTraceProvider(
		tracetest.NewProvider(tracetest.WithSpanRecorder(&sr)),
	)

	ct := httptrace.NewClientTrace(context.Background())
	ct.DNSDone(nhtrace.DNSDoneInfo{})
	ct.DNSStart(nhtrace.DNSStartInfo{Host: "example.com"})

	name := "http.dns"
	require.Contains(t, sr, name)
	spans := sr[name]
	require.Len(t, spans, 1)
}
