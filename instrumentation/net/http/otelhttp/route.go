// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"net/http"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

type routeContextKey struct{}

type routeContext struct {
	sync.RWMutex
	middleware *middleware
	span       trace.Span
	route      string
	disabled   bool
}

// RouteMiddleware records the matched route on the server span and metrics
// created by [NewHandler] or [NewMiddleware] configured with [WithRouteTracking].
// Place it directly around an [http.ServeMux], inside any middleware that copies
// the request with [http.Request.WithContext] or [http.Request.Clone].
//
// It uses the enclosing instrumentation's span name formatter. The innermost
// matched route wins when multiple route middlewares wrap nested muxes. The route
// is added to the server span before a handler panic propagates. Requests with no
// matched pattern are left unchanged. If the nearest enclosing instrumented
// handler does not enable route tracking, it simply calls the next handler.
func RouteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := r.Context().Value(routeContextKey{}).(*routeContext)
		if !ok || route == nil || route.disabled {
			next.ServeHTTP(w, r)
			return
		}
		defer func() {
			routeStr := httpRoute(r.Pattern)
			if routeStr == "" {
				return
			}
			attr := route.middleware.semconv.Route(routeStr)

			// A route middleware closer to the handler already recorded the
			// innermost matched route, so there is nothing left to do.
			route.RLock()
			recorded := route.route != ""
			route.RUnlock()
			if recorded {
				return
			}

			// spanNameFormatter is user code and may be slow. Format the name
			// before taking the write lock so it never blocks a concurrent
			// serveHTTP waiting on route.RLock.
			name := route.middleware.spanNameFormatter(route.middleware.operation, r)

			route.Lock()
			defer route.Unlock()
			if route.route == "" {
				route.route = attr.Value.AsString()
				route.span.SetName(name)
				route.span.SetAttributes(attr)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func httpRoute(pattern string) string {
	if idx := strings.IndexByte(pattern, '/'); idx >= 0 {
		return pattern[idx:]
	}
	return ""
}
