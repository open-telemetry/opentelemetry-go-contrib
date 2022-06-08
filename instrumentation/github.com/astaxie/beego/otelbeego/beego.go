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

package otelbeego // import "go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego"

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego/internal"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/astaxie/beego"
)

// Handler implements the http.Handler interface and provides
// trace and metrics to beego web apps.
type Handler struct {
	http.Handler
}

// ServerHTTP calls the configured handler to serve HTTP for req to rr.
func (o *Handler) ServeHTTP(rr http.ResponseWriter, req *http.Request) {
	ctx := beego.BeeApp.Handlers.GetContext()
	defer beego.BeeApp.Handlers.GiveBackContext(ctx)
	ctx.Reset(rr, req)
	// use the beego context to try to find a route template
	if router, found := beego.BeeApp.Handlers.FindRouter(ctx); found {
		// if found, save it to the context
		reqCtx := context.WithValue(req.Context(), internal.CtxRouteTemplateKey, router.GetPattern())
		req = req.WithContext(reqCtx)
	}
	o.Handler.ServeHTTP(rr, req)
}

// defaultSpanNameFormatter is the default formatter for spans created with the beego
// integration. Returns the route path template, or the URL path if the current path
// is not associated with a router.
func defaultSpanNameFormatter(operation string, req *http.Request) string {
	if val := req.Context().Value(internal.CtxRouteTemplateKey); val != nil {
		str, ok := val.(string)
		if ok {
			return str
		}
	}
	return req.Method
}

// NewOTelBeegoMiddleWare creates a MiddleWare that provides OpenTelemetry
// tracing and metrics to a Beego web app.
// Parameter service should describe the name of the (virtual) server handling the request.
// The OTelBeegoMiddleWare can be configured using the provided Options.
func NewOTelBeegoMiddleWare(service string, options ...Option) beego.MiddleWare {
	cfg := newConfig(options...)

	httpOptions := []otelhttp.Option{
		otelhttp.WithTracerProvider(cfg.tracerProvider),
		otelhttp.WithMeterProvider(cfg.meterProvider),
		otelhttp.WithPropagators(cfg.propagators),
	}

	for _, f := range cfg.filters {
		httpOptions = append(
			httpOptions,
			otelhttp.WithFilter(otelhttp.Filter(f)),
		)
	}

	if cfg.formatter != nil {
		httpOptions = append(httpOptions, otelhttp.WithSpanNameFormatter(cfg.formatter))
	}

	return func(handler http.Handler) http.Handler {
		return &Handler{
			otelhttp.NewHandler(
				handler,
				service,
				httpOptions...,
			),
		}
	}
}

// Render traces beego.Controller.Render. Use this function
// if you want to add a child span for the rendering of a template file.
// Disable autorender before use, and call this function explicitly.
func Render(c *beego.Controller) error {
	_, span := span(c, internal.RenderTemplateSpanName)
	defer span.End()
	err := c.Render()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "template failure")
	}
	return err
}

// RenderString traces beego.Controller.RenderString. Use this function
// if you want to add a child span for the rendering of a template file to
// its string representation.
// Disable autorender before use, and call this function explicitly.
func RenderString(c *beego.Controller) (string, error) {
	_, span := span(c, internal.RenderStringSpanName)
	defer span.End()
	str, err := c.RenderString()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "render string failure")
	}
	return str, err
}

// RenderBytes traces beego.Controller.RenderBytes. Use this function if
// you want to add a child span for the rendering of a template file to its
// byte representation.
// Disable autorender before use, and call this function explicitly.
func RenderBytes(c *beego.Controller) ([]byte, error) {
	_, span := span(c, internal.RenderBytesSpanName)
	defer span.End()
	bytes, err := c.RenderBytes()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "render bytes failure")
	}
	return bytes, err
}

func span(c *beego.Controller, spanName string) (context.Context, trace.Span) {
	ctx := c.Ctx.Request.Context()
	span := trace.SpanFromContext(ctx)
	tracer := span.TracerProvider().Tracer("go.opentelemetry.io/contrib/instrumentation/github/astaxie/beego/otelbeego")
	return tracer.Start(
		ctx,
		spanName,
		trace.WithAttributes(
			Template(c.TplName),
		),
	)
}
