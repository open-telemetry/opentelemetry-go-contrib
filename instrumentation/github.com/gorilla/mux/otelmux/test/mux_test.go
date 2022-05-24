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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func ok(w http.ResponseWriter, _ *http.Request) {}
func notfound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "not found", http.StatusNotFound)
}

func TestSDKIntegration(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar", otelmux.WithTracerProvider(provider)))
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	r1 := httptest.NewRequest("GET", "/book/foo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)
	router.ServeHTTP(w, r1)

	require.Len(t, sr.Ended(), 2)
	assertSpan(t, sr.Ended()[0],
		"/user/{id:[0-9]+}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/user/123"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)
	assertSpan(t, sr.Ended()[1],
		"/book/{title}",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/book/foo"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestNotFoundIsNotError(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar", otelmux.WithTracerProvider(provider)))
	router.HandleFunc("/does/not/exist", notfound)

	r0 := httptest.NewRequest("GET", "/does/not/exist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)

	require.Len(t, sr.Ended(), 1)
	assertSpan(t, sr.Ended()[0],
		"/does/not/exist",
		trace.SpanKindServer,
		attribute.String("http.server_name", "foobar"),
		attribute.Int("http.status_code", http.StatusNotFound),
		attribute.String("http.method", "GET"),
		attribute.String("http.target", "/does/not/exist"),
		attribute.String("http.route", "/does/not/exist"),
	)
	assert.Equal(t, sr.Ended()[0].Status().Code, codes.Unset)
}

func assertSpan(t *testing.T, span sdktrace.ReadOnlySpan, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) {
	assert.Equal(t, name, span.Name())
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())

	got := make(map[attribute.Key]attribute.Value, len(span.Attributes()))
	for _, a := range span.Attributes() {
		got[a.Key] = a.Value
	}
	for _, want := range attrs {
		if !assert.Contains(t, got, want.Key) {
			continue
		}
		assert.Equal(t, got[want.Key], want.Value)
	}
}
