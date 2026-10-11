// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
)

// RouteFromRequest returns the route attribute for a matched request pattern.
func (n HTTPServer) RouteFromRequest(r *http.Request) (attribute.KeyValue, bool) {
	route := httpRoute(r.Pattern)
	if route == "" {
		return attribute.KeyValue{}, false
	}
	return n.Route(route), true
}
