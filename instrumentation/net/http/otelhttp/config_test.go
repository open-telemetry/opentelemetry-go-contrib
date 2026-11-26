// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestBasicFilter(t *testing.T) {
	rr := httptest.NewRecorder()

	spanRecorder := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))

	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), "test_handler",
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithFilter(func(*http.Request) bool {
			return false
		}),
	)

	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)
	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if got := rr.Header().Get("Traceparent"); got != "" {
		t.Fatal("expected empty trace header")
	}
	if got, expected := len(spanRecorder.Ended()), 0; got != expected {
		t.Fatalf("got %d recorded spans, expected %d", got, expected)
	}
	d, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, expected := string(d), "hello world"; got != expected {
		t.Fatalf("got %q, expected %q", got, expected)
	}
}

func TestSpanNameFormatter(t *testing.T) {
	testCases := []struct {
		name      string
		formatter func(s string, r *http.Request) string
		operation string
		expected  string
	}{
		{
			name: "default handler formatter",
			formatter: func(operation string, _ *http.Request) string {
				return operation
			},
			operation: "test_operation",
			expected:  "test_operation",
		},
		{
			name: "default transport formatter",
			formatter: func(_ string, r *http.Request) string {
				return "HTTP " + r.Method
			},
			expected: "HTTP GET",
		},
		{
			name: "custom formatter",
			formatter: func(_ string, r *http.Request) string {
				return r.URL.Path
			},
			operation: "",
			expected:  "/hello",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			spanRecorder := tracetest.NewSpanRecorder()
			provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, err := io.WriteString(w, "hello world"); err != nil {
					t.Fatal(err)
				}
			})
			h := otelhttp.NewHandler(
				handler,
				tc.operation,
				otelhttp.WithTracerProvider(provider),
				otelhttp.WithSpanNameFormatter(tc.formatter),
			)
			r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/hello", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			h.ServeHTTP(rr, r)
			if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
				t.Fatalf("got %d, expected %d", got, expected)
			}

			spans := spanRecorder.Ended()
			if assert.Len(t, spans, 1) {
				assert.Equal(t, tc.expected, spans[0].Name())
			}
		})
	}
}

func TestEvent(t *testing.T) {
	t.Run("constant values", func(t *testing.T) {
		assert.Equal(t, otelhttp.ReadEvents, otelhttp.Event(1), "ReadEvents should be 1")
		assert.Equal(t, otelhttp.WriteEvents, otelhttp.Event(2), "WriteEvents should be 2")
	})

	t.Run("unspecifiedEvent", func(t *testing.T) {
		var unspecified otelhttp.Event // zero-value
		assert.Equal(t, otelhttp.Event(0), unspecified, "unspecifiedEvent should be zero-value Event")

		// Validate that unspecifiedEvent is different from defined events
		assert.NotEqual(t, otelhttp.ReadEvents, unspecified, "unspecifiedEvent should not equal ReadEvents")
		assert.NotEqual(t, otelhttp.WriteEvents, unspecified, "unspecifiedEvent should not equal WriteEvents")

		// Validate WithMessageEvents accepts unspecifiedEvent
		opt := otelhttp.WithMessageEvents(unspecified)
		assert.NotNil(t, opt, "WithMessageEvents(unspecifiedEvent) should not return nil")

		// Additional validation: test behavior with unspecified events
		optMultiple := otelhttp.WithMessageEvents(unspecified, otelhttp.ReadEvents)
		assert.NotNil(t, optMultiple, "WithMessageEvents with unspecified and valid events should not return nil")
	})

	t.Run("WithMessageEvents", func(t *testing.T) {
		tests := []struct {
			name   string
			events []otelhttp.Event
		}{
			{name: "ReadEvents", events: []otelhttp.Event{otelhttp.ReadEvents}},
			{name: "WriteEvents", events: []otelhttp.Event{otelhttp.WriteEvents}},
			{name: "multiple events", events: []otelhttp.Event{otelhttp.ReadEvents, otelhttp.WriteEvents}},
			{name: "no events", events: []otelhttp.Event{}},
			{name: "unspecified event only", events: []otelhttp.Event{otelhttp.Event(0)}},
			{name: "mixed with unspecified", events: []otelhttp.Event{otelhttp.Event(0), otelhttp.ReadEvents}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := otelhttp.WithMessageEvents(tt.events...)
				assert.NotNil(t, got, "WithMessageEvents(%v) should not return nil", tt.events)
			})
		}
	})
}

func TestWithServerName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid server name",
			input:    "test-server-name",
			expected: "test-server-name",
		},
		{
			name:     "Empty server name",
			input:    "",
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := otelhttp.WithServerName(tt.input)
			assert.NotNil(t, opt)

			rr := httptest.NewRecorder()
			spanRecorder := tracetest.NewSpanRecorder()
			provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))

			h := otelhttp.NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
				"test_operation",
				otelhttp.WithTracerProvider(provider),
				opt,
			)

			r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/", http.NoBody)
			assert.NoError(t, err)

			h.ServeHTTP(rr, r)

			spans := spanRecorder.Ended()
			if !assert.Len(t, spans, 1) {
				return
			}
			var found bool
			for _, attr := range spans[0].Attributes() {
				if attr.Key == "server.address" {
					assert.Equal(t, tt.expected, attr.Value.AsString())
					found = true
					break
				}
			}
			assert.True(t, found, "server.address attribute not found in span attributes")
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestOptionTypes(t *testing.T) {
	t.Run("type exclusivity", func(t *testing.T) {
		testCases := []struct {
			name               string
			opt                any
			expectHandlerOpt   bool
			expectTransportOpt bool
		}{
			{
				name:               "WithPublicEndpointFn",
				opt:                otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return false }),
				expectHandlerOpt:   true,
				expectTransportOpt: false,
			},
			{
				name:               "WithMessageEvents",
				opt:                otelhttp.WithMessageEvents(otelhttp.ReadEvents),
				expectHandlerOpt:   true,
				expectTransportOpt: false,
			},
			{
				name:               "WithServerName",
				opt:                otelhttp.WithServerName("example.com"),
				expectHandlerOpt:   true,
				expectTransportOpt: false,
			},
			{
				name:               "WithRouteTracking",
				opt:                otelhttp.WithRouteTracking(),
				expectHandlerOpt:   true,
				expectTransportOpt: false,
			},
			{
				name:               "WithClientTrace",
				opt:                otelhttp.WithClientTrace(func(context.Context) *httptrace.ClientTrace { return nil }),
				expectHandlerOpt:   false,
				expectTransportOpt: true,
			},
			{
				name:               "WithTracerProvider",
				opt:                otelhttp.WithTracerProvider(trace.NewTracerProvider()),
				expectHandlerOpt:   true,
				expectTransportOpt: true,
			},
			{
				name:               "WithMeterProvider",
				opt:                otelhttp.WithMeterProvider(noop.NewMeterProvider()),
				expectHandlerOpt:   true,
				expectTransportOpt: true,
			},
			{
				name:               "WithPropagators",
				opt:                otelhttp.WithPropagators(propagation.TraceContext{}),
				expectHandlerOpt:   true,
				expectTransportOpt: true,
			},
			{
				name:               "WithSpanOptions",
				opt:                otelhttp.WithSpanOptions(oteltrace.WithAttributes(attribute.String("custom", "val"))),
				expectHandlerOpt:   true,
				expectTransportOpt: true,
			},
			{
				name:               "WithFilter",
				opt:                otelhttp.WithFilter(func(*http.Request) bool { return true }),
				expectHandlerOpt:   true,
				expectTransportOpt: true,
			},
			{
				name:               "WithSpanNameFormatter",
				opt:                otelhttp.WithSpanNameFormatter(func(op string, _ *http.Request) string { return op }),
				expectHandlerOpt:   true,
				expectTransportOpt: true,
			},
			{
				name:               "WithMetricAttributesFn",
				opt:                otelhttp.WithMetricAttributesFn(func(*http.Request) []attribute.KeyValue { return nil }),
				expectHandlerOpt:   true,
				expectTransportOpt: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, isHandler := tc.opt.(otelhttp.HandlerOption)
				_, isTransport := tc.opt.(otelhttp.TransportOption)
				assert.Equal(t, tc.expectHandlerOpt, isHandler)
				assert.Equal(t, tc.expectTransportOpt, isTransport)
			})
		}
	})

	t.Run("options take effect", func(t *testing.T) {
		const incomingTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		const incomingTraceparent = "00-" + incomingTraceID + "-00f067aa0ba902b7-01"

		testCases := []struct {
			name string
			run  func(t *testing.T, sharedOpts []otelhttp.Option) (traceID string)
		}{
			{
				name: "NewHandler",
				run: func(t *testing.T, sharedOpts []otelhttp.Option) string {
					hOpts := []otelhttp.HandlerOption{
						otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return false }),
						otelhttp.WithMessageEvents(otelhttp.ReadEvents),
						otelhttp.WithServerName("example.com"),
						otelhttp.WithRouteTracking(),
					}
					for _, opt := range sharedOpts {
						hOpts = append(hOpts, opt)
					}
					h := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusOK)
					}), "op", hOpts...)
					rr := httptest.NewRecorder()
					req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/", http.NoBody)
					req.Header.Set("Traceparent", incomingTraceparent)
					h.ServeHTTP(rr, req)
					return incomingTraceID
				},
			},
			{
				name: "NewMiddleware",
				run: func(t *testing.T, sharedOpts []otelhttp.Option) string {
					hOpts := []otelhttp.HandlerOption{
						otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return false }),
						otelhttp.WithMessageEvents(otelhttp.ReadEvents),
						otelhttp.WithServerName("example.com"),
						otelhttp.WithRouteTracking(),
					}
					for _, opt := range sharedOpts {
						hOpts = append(hOpts, opt)
					}
					mw := otelhttp.NewMiddleware("op", hOpts...)
					h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusOK)
					}))
					rr := httptest.NewRecorder()
					req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/", http.NoBody)
					req.Header.Set("Traceparent", incomingTraceparent)
					h.ServeHTTP(rr, req)
					return incomingTraceID
				},
			},
			{
				name: "NewTransport",
				run: func(t *testing.T, sharedOpts []otelhttp.Option) string {
					var clientTraceCalled bool
					tOpts := []otelhttp.TransportOption{
						otelhttp.WithClientTrace(func(context.Context) *httptrace.ClientTrace {
							clientTraceCalled = true
							return &httptrace.ClientTrace{}
						}),
					}
					for _, opt := range sharedOpts {
						tOpts = append(tOpts, opt)
					}
					var injectedTraceparent string
					base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
						injectedTraceparent = r.Header.Get("Traceparent")
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       http.NoBody,
							Request:    r,
						}, nil
					})
					tr := otelhttp.NewTransport(base, tOpts...)
					req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/", http.NoBody)
					resp, err := tr.RoundTrip(req)
					assert.NoError(t, err)
					if resp != nil && resp.Body != nil {
						_ = resp.Body.Close()
					}
					assert.True(t, clientTraceCalled, "expected WithClientTrace to take effect")
					assert.NotEmpty(t, injectedTraceparent, "expected WithPropagators to inject Traceparent")
					return ""
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				spanRecorder := tracetest.NewSpanRecorder()
				provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
				var filterCalled bool

				sharedOpts := []otelhttp.Option{
					otelhttp.WithTracerProvider(provider),
					otelhttp.WithMeterProvider(noop.NewMeterProvider()),
					otelhttp.WithPropagators(propagation.TraceContext{}),
					otelhttp.WithSpanOptions(oteltrace.WithAttributes(attribute.String("custom.opt", "applied"))),
					otelhttp.WithFilter(func(*http.Request) bool {
						filterCalled = true
						return true
					}),
					otelhttp.WithSpanNameFormatter(func(_ string, _ *http.Request) string {
						return "custom-span-name"
					}),
					otelhttp.WithMetricAttributesFn(func(*http.Request) []attribute.KeyValue { return nil }),
				}

				expectedTraceID := tc.run(t, sharedOpts)
				assert.True(t, filterCalled, "expected WithFilter to take effect")

				spans := spanRecorder.Ended()
				if assert.Len(t, spans, 1) {
					assert.Equal(t, "custom-span-name", spans[0].Name())
					assert.Contains(t, spans[0].Attributes(), attribute.String("custom.opt", "applied"))
					if expectedTraceID != "" {
						assert.Equal(t, expectedTraceID, spans[0].SpanContext().TraceID().String())
					}
				}
			})
		}
	})
}
