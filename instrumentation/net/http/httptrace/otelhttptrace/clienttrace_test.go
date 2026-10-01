// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttptrace

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace/noop"
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", http.NoBody)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	fmt.Println(resp.Status)
}

// BenchmarkWroteHeaderFieldWithoutSubSpans isolates header attribute encoding
// on the parent span; it does not measure the default subspan path.
func BenchmarkWroteHeaderFieldWithoutSubSpans(b *testing.B) {
	for _, tc := range []struct {
		name   string
		values []string
	}{
		{name: "single-value", values: []string{"value"}},
		{name: "multiple-values", values: []string{"first", "second"}},
	} {
		// The scalar case preserves the pre-change value encoding as a baseline.
		b.Run(tc.name+"/scalar", func(b *testing.B) {
			provider := noop.NewTracerProvider()
			_, span := provider.Tracer(ScopeName).Start(b.Context(), "root")
			ct := &clientTracer{root: span, addHeaders: true}
			redactedHeaders := map[string]struct{}{
				"authorization":       {},
				"www-authenticate":    {},
				"proxy-authenticate":  {},
				"proxy-authorization": {},
				"cookie":              {},
				"set-cookie":          {},
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if ct.useSpans && ct.span("http.headers") == nil {
					ct.start("http.headers", "http.headers")
				}
				if !ct.addHeaders {
					continue
				}
				name := strings.ToLower("X-Custom-Header")
				value := sliceToString(tc.values)
				if _, ok := redactedHeaders[name]; ok {
					value = "****"
				}
				span.SetAttributes(attribute.String(
					"http.request.header."+name,
					value,
				))
				ct.start("http.send", "http.send")
			}
			b.StopTimer()
			span.End()
		})
		b.Run(tc.name+"/string-slice", func(b *testing.B) {
			provider := noop.NewTracerProvider()
			_, span := provider.Tracer(ScopeName).Start(b.Context(), "root")
			ct := &clientTracer{
				root:       span,
				addHeaders: true,
				headerAttributes: map[string]headerAttribute{
					"authorization":       {redacted: true},
					"www-authenticate":    {redacted: true},
					"proxy-authenticate":  {redacted: true},
					"proxy-authorization": {redacted: true},
					"cookie":              {redacted: true},
					"set-cookie":          {redacted: true},
				},
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				ct.wroteHeaderField("X-Custom-Header", tc.values)
				ct.wroteHeaders()
			}
			b.StopTimer()
			span.End()
		})
	}
}
