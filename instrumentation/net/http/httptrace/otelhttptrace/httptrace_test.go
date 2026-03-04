// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttptrace_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
)

func TestRoundtrip(t *testing.T) {
	tr := noop.NewTracerProvider().Tracer("httptrace/client")

	var expectedAttrs map[attribute.Key]string
	expectedCorrs := map[string]string{"foo": "bar"}

	props := otelhttptrace.WithPropagators(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Mock http server
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attrs, corrs, span := otelhttptrace.Extract(r.Context(), r, props)

			actualAttrs := make(map[attribute.Key]string)
			for _, attr := range attrs {
				if expectedAttrs[attr.Key] == "any" {
					actualAttrs[attr.Key] = expectedAttrs[attr.Key]
				} else {
					actualAttrs[attr.Key] = attr.Value.Emit()
				}
			}

			if diff := cmp.Diff(actualAttrs, expectedAttrs); diff != "" {
				t.Fatalf("[TestRoundtrip] Attributes are different: %v", diff)
			}

			actualCorrs := make(map[string]string)
			for _, corr := range corrs.Members() {
				actualCorrs[corr.Key()] = corr.Value()
			}

			if diff := cmp.Diff(actualCorrs, expectedCorrs); diff != "" {
				t.Fatalf("[TestRoundtrip] Correlations are different: %v", diff)
			}

			if !span.IsValid() {
				t.Fatalf("[TestRoundtrip] Invalid span extracted: %v", span)
			}

			_, err := w.Write([]byte("OK"))
			if err != nil {
				t.Fatal(err)
			}
		}),
	)
	defer ts.Close()

	address := ts.Listener.Addr()
	hp := strings.Split(address.String(), ":")
	expectedAttrs = map[attribute.Key]string{
		"client.address":           hp[0],
		"http.request.body.size":   "3",
		"http.request.method":      "GET",
		"network.peer.address":     hp[0],
		"network.peer.port":        "any",
		"network.protocol.version": "1.1",
		"network.transport":        "tcp",
		"server.address":           "127.0.0.1",
		"server.port":              hp[1],
		"url.path":                 "/",
		"url.scheme":               "http",
		"user_agent.original":      "Go-http-client/1.1",
	}

	client := ts.Client()
	ctx := t.Context()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	err := func(ctx context.Context) error {
		ctx, span := tr.Start(ctx, "test")
		defer span.End()
		bag, _ := baggage.Parse("foo=bar")
		ctx = baggage.ContextWithBaggage(ctx, bag)
		req, _ := http.NewRequest("GET", ts.URL, strings.NewReader("foo"))
		otelhttptrace.Inject(ctx, req, props)

		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %s", err.Error())
		}
		_ = res.Body.Close()

		return nil
	}(ctx)
	if err != nil {
		panic("unexpected error in http request: " + err.Error())
	}
}

func TestSpecifyPropagators(t *testing.T) {
	tr := noop.NewTracerProvider().Tracer("httptrace/client")

	expectedCorrs := map[string]string{"foo": "bar"}

	// Mock http server
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, corrs, span := otelhttptrace.Extract(r.Context(), r, otelhttptrace.WithPropagators(propagation.Baggage{}))

			actualCorrs := make(map[string]string)
			for _, corr := range corrs.Members() {
				actualCorrs[corr.Key()] = corr.Value()
			}

			if diff := cmp.Diff(actualCorrs, expectedCorrs); diff != "" {
				t.Fatalf("[TestRoundtrip] Correlations are different: %v", diff)
			}

			if span.IsValid() {
				t.Fatalf("[TestRoundtrip] valid span extracted, expected none: %v", span)
			}

			_, err := w.Write([]byte("OK"))
			if err != nil {
				t.Fatal(err)
			}
		}),
	)
	defer ts.Close()

	client := ts.Client()
	err := func(ctx context.Context) error {
		ctx, span := tr.Start(ctx, "test")
		defer span.End()
		bag, _ := baggage.Parse("foo=bar")
		ctx = baggage.ContextWithBaggage(ctx, bag)
		req, _ := http.NewRequest("GET", ts.URL, http.NoBody)
		otelhttptrace.Inject(ctx, req, otelhttptrace.WithPropagators(propagation.Baggage{}))

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
}
