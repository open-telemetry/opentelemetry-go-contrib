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

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
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
				if attr.Key == semconv.NetSockPeerPortKey {
					// Peer port will be non-deterministic
					continue
				}
				actualAttrs[attr.Key] = attr.Value.Emit()
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
		semconv.NetHostNameKey:              hp[0],
		semconv.NetHostPortKey:              hp[1],
		semconv.NetProtocolVersionKey:       "1.1",
		semconv.HTTPMethodKey:               "GET",
		semconv.HTTPSchemeKey:               "http",
		semconv.HTTPTargetKey:               "/",
		semconv.HTTPRequestContentLengthKey: "3",
		semconv.NetSockPeerAddrKey:          hp[0],
		semconv.NetTransportKey:             "ip_tcp",
		semconv.UserAgentOriginalKey:        "Go-http-client/1.1",
	}

	client := ts.Client()
	ctx := context.Background()
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
		req, _ := http.NewRequest("GET", ts.URL, nil)
		otelhttptrace.Inject(ctx, req, otelhttptrace.WithPropagators(propagation.Baggage{}))

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
}
