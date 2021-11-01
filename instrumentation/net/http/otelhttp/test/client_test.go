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

package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestConvenienceWrappers(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	orig := otelhttp.DefaultClient
	otelhttp.DefaultClient = &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithTracerProvider(provider),
		),
	}
	defer func() { otelhttp.DefaultClient = orig }()

	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	res, err := otelhttp.Get(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res, err = otelhttp.Head(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res, err = otelhttp.Post(ctx, ts.URL, "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	form := make(url.Values)
	form.Set("foo", "bar")
	res, err = otelhttp.PostForm(ctx, ts.URL, form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	spans := sr.Ended()
	require.Equal(t, 4, len(spans))
	assert.Equal(t, "HTTP GET", spans[0].Name())
	assert.Equal(t, "HTTP HEAD", spans[1].Name())
	assert.Equal(t, "HTTP POST", spans[2].Name())
	assert.Equal(t, "HTTP POST", spans[3].Name())
}

func TestClientWithTraceContext(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "http requests")

	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	res, err := otelhttp.Get(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	span.End()

	spans := sr.Ended()
	require.Equal(t, 2, len(spans))
	assert.Equal(t, "HTTP GET", spans[0].Name())
	assert.Equal(t, "http requests", spans[1].Name())
	assert.NotEmpty(t, spans[0].Parent().SpanID())
	assert.Equal(t, spans[1].SpanContext().SpanID(), spans[0].Parent().SpanID())
}
