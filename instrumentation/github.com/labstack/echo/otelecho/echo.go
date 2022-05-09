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
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

const (
	ctxEchoPath = "ctxEchoPath"
)

// Middleware returns echo middleware which will trace incoming requests.
func Middleware(service string, opts ...otelhttp.Option) echo.MiddlewareFunc {
	return wrapMiddleware(func(handlerFunc http.Handler) http.Handler {
		return otelhttp.NewHandler(handlerFunc, service, opts...)
	})
}

// WithRouteTag wraps otelhttp.WithRouteTag into an echo middleware
func WithRouteTag(route string) echo.MiddlewareFunc {
	return wrapMiddleware(func(handler http.Handler) http.Handler {
		return otelhttp.WithRouteTag(route, handler)
	})
}

// PathSpanNameFormatter formats span names with the name of the path for the routed handler
// The PathSpanNameFormatter requires that the server has the instrumentation middleware inserted before it
func PathSpanNameFormatter(operation string, r *http.Request) string {
	path := r.Context().Value(ctxEchoPath)
	return path.(string)
}

func wrapMiddleware(middleware func(http.Handler) http.Handler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.WithValue(c.Request().Context(), ctxEchoPath, c.Path())
			middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			})).ServeHTTP(c.Response(), c.Request().WithContext(ctx))
			return nil
		}
	}
}
