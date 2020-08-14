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

package beego

import (
	"context"
	"net/http"

	"google.golang.org/grpc/codes"

	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http"
	"go.opentelemetry.io/otel/api/trace"

	"github.com/astaxie/beego"
)

// OTelBeegoHandler implements the http.Handler interface and provides
// trace and metrics to beego web apps.
type OTelBeegoHandler struct {
	http.Handler
}

func (o *OTelBeegoHandler) ServeHTTP(rr http.ResponseWriter, req *http.Request) {
	ctx := beego.BeeApp.Handlers.GetContext()
	defer beego.BeeApp.Handlers.GiveBackContext(ctx)
	ctx.Reset(rr, req)
	// use the beego context to try to find a route template
	if router, found := beego.BeeApp.Handlers.FindRouter(ctx); found {
		// if found, save it to the context
		reqCtx := context.WithValue(req.Context(), ctxRouteTemplateKey, router.GetPattern())
		req = req.WithContext(reqCtx)
	}
	o.Handler.ServeHTTP(rr, req)
}

// defaultSpanNameFormatter is the default formatter for spans created with the beego
// integration. Returns the route path template, or the URL path if the current path
// is not associated with a router.
func defaultSpanNameFormatter(operation string, req *http.Request) string {
	if val := req.Context().Value(ctxRouteTemplateKey); val != nil {
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
	cfg := configure(options...)

	httpOptions := []otelhttp.Option{
		otelhttp.WithTracer(cfg.traceProvider.Tracer(packageName)),
		otelhttp.WithMeter(cfg.meterProvider.Meter(packageName)),
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
		return &OTelBeegoHandler{
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
	ctx, span := span(c, renderTemplateSpanName)
	defer span.End()
	err := c.Render()
	if err != nil {
		span.RecordError(ctx, err)
		span.SetStatus(codes.Internal, "template failure")
	}
	return err
}

// RenderString traces beego.Controller.RenderString. Use this function
// if you want to add a child span for the rendering of a template file to
// its string representation.
// Disable autorender before use, and call this function explicitly.
func RenderString(c *beego.Controller) (string, error) {
	ctx, span := span(c, renderStringSpanName)
	defer span.End()
	str, err := c.RenderString()
	if err != nil {
		span.RecordError(ctx, err)
		span.SetStatus(codes.Internal, "render string failure")
	}
	return str, err
}

// RenderBytes traces beego.Controller.RenderBytes. Use this function if
// you want to add a child span for the rendering of a template file to its
// byte representation.
// Disable autorender before use, and call this function explicitly.
func RenderBytes(c *beego.Controller) ([]byte, error) {
	ctx, span := span(c, renderBytesSpanName)
	defer span.End()
	bytes, err := c.RenderBytes()
	if err != nil {
		span.RecordError(ctx, err)
		span.SetStatus(codes.Internal, "render bytes failure")
	}
	return bytes, err
}

func span(c *beego.Controller, spanName string) (context.Context, trace.Span) {
	ctx := c.Ctx.Request.Context()
	span := trace.SpanFromContext(ctx)
	tracer := span.Tracer()
	return tracer.Start(
		ctx,
		spanName,
		trace.WithAttributes(
			Template(c.TplName),
		),
	)

}
