// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"time"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

type spanTimestampKey struct{}

// AttributeSetter returns an array of KeyValue pairs, it can be used to set custom attributes.
//
// Deprecated: Use AttributeBuilder instead. This will be removed in a future release.
type AttributeSetter func(context.Context, middleware.InitializeInput) []attribute.KeyValue

// AttributeBuilder returns an array of KeyValue pairs, it can be used to set custom attributes.
type AttributeBuilder func(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) []attribute.KeyValue

type otelMiddlewares struct {
	tracer            trace.Tracer
	propagator        propagation.TextMapPropagator
	attributeBuilders []AttributeBuilder
}

func (m otelMiddlewares) initializeMiddlewareBefore(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OTelInitializeMiddlewareBefore", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error,
	) {
		ctx = context.WithValue(ctx, spanTimestampKey{}, time.Now())
		return next.HandleInitialize(ctx, in)
	}),
		middleware.Before)
}

func (m otelMiddlewares) initializeMiddlewareAfter(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OTelInitializeMiddlewareAfter", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error,
	) {
		serviceID := v2Middleware.GetServiceID(ctx)
		operation := v2Middleware.GetOperationName(ctx)
		region := v2Middleware.GetRegion(ctx)

		attributes := []attribute.KeyValue{
			SystemAttr(),
			ServiceAttr(serviceID),
			RegionAttr(region),
			OperationAttr(operation),
		}

		ctx, span := m.tracer.Start(ctx, spanName(serviceID, operation),
			trace.WithTimestamp(ctx.Value(spanTimestampKey{}).(time.Time)),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attributes...),
		)
		defer span.End()

		out, metadata, err = next.HandleInitialize(ctx, in)
		span.SetAttributes(m.buildAttributes(ctx, in, out)...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		return out, metadata, err
	}),
		middleware.After)
}

func (m otelMiddlewares) finalizeMiddlewareAfter(stack *middleware.Stack) error {
	return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("OTelFinalizeMiddleware", func(
		ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (
		out middleware.FinalizeOutput, metadata middleware.Metadata, err error,
	) {
		// Propagate the Trace information by injecting it into the HTTP request.
		switch req := in.Request.(type) {
		case *smithyhttp.Request:
			m.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
		default:
		}

		return next.HandleFinalize(ctx, in)
	}),
		middleware.After)
}

func (m otelMiddlewares) deserializeMiddleware(stack *middleware.Stack) error {
	return stack.Deserialize.Add(middleware.DeserializeMiddlewareFunc("OTelDeserializeMiddleware", func(
		ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (
		out middleware.DeserializeOutput, metadata middleware.Metadata, err error,
	) {
		out, metadata, err = next.HandleDeserialize(ctx, in)
		resp, ok := out.RawResponse.(*smithyhttp.Response)
		if !ok {
			// No raw response to wrap with.
			return out, metadata, err
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(semconv.HTTPStatusCode(resp.StatusCode))

		requestID, ok := v2Middleware.GetRequestIDMetadata(metadata)
		if ok {
			span.SetAttributes(RequestIDAttr(requestID))
		}

		return out, metadata, err
	}),
		middleware.Before)
}

func (m otelMiddlewares) buildAttributes(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) (attributes []attribute.KeyValue) {
	for _, builder := range m.attributeBuilders {
		attributes = append(attributes, builder(ctx, in, out)...)
	}

	return attributes
}

func spanName(serviceID, operation string) string {
	spanName := serviceID
	if operation != "" {
		spanName += "." + operation
	}
	return spanName
}

// AppendMiddlewares attaches OTel middlewares to the AWS Go SDK V2 for instrumentation.
// OTel middlewares can be appended to either all aws clients or a specific operation.
// Please see more details in https://aws.github.io/aws-sdk-go-v2/docs/middleware/
func AppendMiddlewares(apiOptions *[]func(*middleware.Stack) error, opts ...Option) {
	cfg := config{
		TracerProvider:    otel.GetTracerProvider(),
		TextMapPropagator: otel.GetTextMapPropagator(),
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.AttributeBuilders == nil {
		cfg.AttributeBuilders = []AttributeBuilder{DefaultAttributeBuilder}
	}

	m := otelMiddlewares{
		tracer: cfg.TracerProvider.Tracer(ScopeName,
			trace.WithInstrumentationVersion(Version())),
		propagator:        cfg.TextMapPropagator,
		attributeBuilders: cfg.AttributeBuilders,
	}
	*apiOptions = append(*apiOptions, m.initializeMiddlewareBefore, m.initializeMiddlewareAfter, m.finalizeMiddlewareAfter, m.deserializeMiddleware)
}
