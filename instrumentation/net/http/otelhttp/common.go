// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

// Filter is a predicate used to determine whether a given http.request should
// be traced. A Filter must return true if the request should be traced.
type Filter func(*http.Request) bool

func newTracer(tp trace.TracerProvider) trace.Tracer {
	return tp.Tracer(ScopeName, trace.WithInstrumentationVersion(Version))
}
