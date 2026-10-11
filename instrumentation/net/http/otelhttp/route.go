// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

type routeContextKey struct{}

type routeContext struct {
	middleware *middleware
	span       trace.Span
	request    *http.Request
}

// RouteMiddleware records the matched route on the server span and metrics
// created by [NewHandler] or [NewMiddleware]. Place it directly around an
// [http.ServeMux], inside any middleware that copies the request with
// [http.Request.WithContext] or [http.Request.Clone].
//
// It uses the enclosing instrumentation's span name formatter. The innermost
// matched route wins when multiple route middlewares wrap nested muxes. The route
// is added to the server span before a handler panic propagates. Requests with no
// matched pattern are left unchanged. Without enclosing otelhttp instrumentation, it
// simply calls the next handler.
func RouteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := r.Context().Value(routeContextKey{}).(*routeContext)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		defer func() {
			if route.request != nil {
				return
			}
			attr, matched := route.middleware.semconv.RouteFromRequest(r)
			if !matched {
				return
			}
			// Capture the routed request before an outer mux or middleware can
			// change its pattern, and keep it local to this server request.
			route.request = r.WithContext(r.Context())
			route.span.SetName(route.middleware.spanNameFormatter(route.middleware.operation, route.request))
			route.span.SetAttributes(attr)
		}()
		next.ServeHTTP(w, r)
	})
}
