// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelrestful

import (
	"strconv"
	"time"

	"github.com/emicklei/go-restful/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful/internal/semconv"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"

// OTelFilter returns a restful.FilterFunction which will trace an incoming request.
//
// The service parameter should describe the name of the (virtual) server handling
// the request.  Options can be applied to configure the tracer and propagators
// used for this filter.
func OTelFilter(service string, opts ...Option) restful.FilterFunction {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		ScopeName,
		oteltrace.WithInstrumentationVersion(Version),
	)
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = otel.GetMeterProvider()
	}
	meter := cfg.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version),
	)
	semconvServer := semconv.NewHTTPServer(meter)

	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		requestStartTime := time.Now()
		r := req.Request
		ctx := cfg.Propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		route := req.SelectedRoutePath()
		rSemconv := *r
		rSemconv.Pattern = route
		spanName := semconvServer.SpanName(&rSemconv)

		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(semconvServer.RequestTraceAttrs(service, &rSemconv, semconv.RequestTraceAttrsOpts{})...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		}

		if cfg.PublicEndpointFn != nil && cfg.PublicEndpointFn(r.WithContext(ctx)) {
			opts = append(opts, oteltrace.WithNewRoot())
			// Linking incoming span context if any for public endpoint.
			if s := oteltrace.SpanContextFromContext(ctx); s.IsValid() && s.IsRemote() {
				opts = append(opts, oteltrace.WithLinks(oteltrace.Link{SpanContext: s}))
			}
		}

		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		req.Request = req.Request.WithContext(ctx)
		// Copy MultipartForm back to the original request so net/http's
		// finishRequest can find and remove any temp files ParseMultipartForm
		// created on the context-derived request copy.
		defer func() {
			if req.Request != nil && req.Request.MultipartForm != nil {
				r.MultipartForm = req.Request.MultipartForm
			}
		}()

		chain.ProcessFilter(req, resp)

		status := resp.StatusCode()
		var additionalAttributes []attribute.KeyValue
		err := ctx.Err()
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			errAttr := otelsemconv.ErrorType(err)
			span.SetAttributes(errAttr)
			additionalAttributes = append(additionalAttributes, errAttr)
		} else {
			stCode, stMsg := semconvServer.Status(status)
			span.SetStatus(stCode, stMsg)
			if status >= 500 && status < 600 {
				errAttr := otelsemconv.ErrorTypeKey.String(strconv.Itoa(status))
				span.SetAttributes(errAttr)
				additionalAttributes = append(additionalAttributes, errAttr)
			}
		}

		respSize := int64(max(resp.ContentLength(), 0))
		span.SetAttributes(semconvServer.ResponseTraceAttrs(semconv.ResponseTelemetry{
			StatusCode: status,
			WriteBytes: respSize,
		})...)

		semconvServer.RecordMetrics(ctx, semconv.ServerMetricData{
			ServerName:   service,
			ResponseSize: respSize,
			MetricAttributes: semconv.MetricAttributes{
				Req:                  &rSemconv,
				StatusCode:           status,
				Route:                route,
				AdditionalAttributes: additionalAttributes,
			},
			MetricData: semconv.MetricData{
				RequestSize:     r.ContentLength,
				RequestDuration: time.Since(requestStartTime),
			},
		})
	}
}
