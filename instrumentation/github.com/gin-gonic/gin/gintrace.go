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

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace.go

package gin

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/otel/codes"

	otelglobal "go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

const (
	tracerKey  = "otel-go-contrib-tracer"
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin"
)

// Middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func Middleware(service string, opts ...Option) gin.HandlerFunc {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Tracer == nil {
		cfg.Tracer = otelglobal.Tracer(tracerName)
	}
	if cfg.Propagators == nil {
		cfg.Propagators = otelglobal.Propagators()
	}
	return func(c *gin.Context) {
		c.Set(tracerKey, cfg.Tracer)
		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()
		ctx := otelpropagation.ExtractHTTP(savedCtx, cfg.Propagators, c.Request.Header)
		opts := []oteltrace.StartOption{
			oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", c.Request)...),
			oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(c.Request)...),
			oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, c.FullPath(), c.Request)...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}
		spanName := c.FullPath()
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Request.Method)
		}
		ctx, span := cfg.Tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		c.Request = c.Request.WithContext(ctx)

		// serve the request to the next middleware
		c.Next()

		status := c.Writer.Status()
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(status)
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(status)
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
		if len(c.Errors) > 0 {
			span.SetAttributes(label.String("gin.errors", c.Errors.String()))
		}
	}
}

// HTML will trace the rendering of the template as a child of the
// span in the given context. This is a replacement for
// gin.Context.HTML function - it invokes the original function after
// setting up the span.
func HTML(c *gin.Context, code int, name string, obj interface{}) {
	var tracer oteltrace.Tracer
	tracerInterface, ok := c.Get(tracerKey)
	if ok {
		tracer, ok = tracerInterface.(oteltrace.Tracer)
	}
	if !ok {
		tracer = otelglobal.Tracer(tracerName)
	}
	savedContext := c.Request.Context()
	defer func() {
		c.Request = c.Request.WithContext(savedContext)
	}()
	opt := oteltrace.WithAttributes(label.String("go.template", name))
	ctx, span := tracer.Start(savedContext, "gin.renderer.html", opt)
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("error rendering template:%s: %s", name, r)
			span.RecordError(ctx, err)
			span.SetStatus(codes.Internal, "template failure")
			span.End()
			panic(r)
		} else {
			span.End()
		}
	}()
	c.HTML(code, name, obj)
}
