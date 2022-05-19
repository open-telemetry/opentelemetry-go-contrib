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

package otelecho // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type echoCtxKey int

const (
	echoContextCtxKey echoCtxKey = iota
)

// Middleware returns echo middleware which will trace incoming requests.
func Middleware(service string, opts ...Option) echo.MiddlewareFunc {
	conf := newConfig(opts...)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var handlerFunc http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.SetRequest(r)
				c.SetResponse(echo.NewResponse(w, c.Echo()))
				err := next(c)
				if err != nil {
					span := trace.SpanFromContext(r.Context())
					if span != nil {
						span.SetAttributes(attribute.String("echo.error", err.Error()))
					}
					// invokes the registered HTTP error handler
					c.Error(err)
				}
			})

			if conf.routeTagFromPath {
				handlerFunc = otelhttp.WithRouteTag(c.Path(), handlerFunc)
			}

			ctx := context.WithValue(c.Request().Context(), echoContextCtxKey, c)
			otelhttp.NewHandler(handlerFunc, service, conf.otelhttpOptions...).
				ServeHTTP(c.Response(), c.Request().WithContext(ctx))
			return nil
		}
	}
}

// WithRouteTag wraps otelhttp.WithRouteTag into an echo middleware
func WithRouteTag(route string) echo.MiddlewareFunc {
	return echo.WrapMiddleware(func(handler http.Handler) http.Handler {
		return otelhttp.WithRouteTag(route, handler)
	})
}

// PathSpanNameFormatter formats span names with the name of the path for the routed handler
// The PathSpanNameFormatter requires that the server has the instrumentation middleware inserted before it
func PathSpanNameFormatter(operation string, r *http.Request) string {
	ctx := r.Context().Value(echoContextCtxKey).(echo.Context)
	return ctx.Path()
}
