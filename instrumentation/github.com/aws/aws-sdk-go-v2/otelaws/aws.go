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

package otelaws

import (
	"context"
	"time"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

type spanTimestampKey struct{}

type otelMiddlewares struct {
	tracer trace.Tracer
}

func (m otelMiddlewares) initializeMiddlewareBefore(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OTelInitializeMiddlewareBefore", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

		ctx = context.WithValue(ctx, spanTimestampKey{}, time.Now())
		return next.HandleInitialize(ctx, in)
	}),
		middleware.Before)
}

func (m otelMiddlewares) initializeMiddlewareAfter(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OTelInitializeMiddlewareAfter", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

		serviceID := v2Middleware.GetServiceID(ctx)
		ctx, span := m.tracer.Start(ctx, serviceID,
			trace.WithTimestamp(ctx.Value(spanTimestampKey{}).(time.Time)),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(ServiceAttr(serviceID),
				RegionAttr(v2Middleware.GetRegion(ctx)),
				OperationAttr(v2Middleware.GetOperationName(ctx))),
		)
		defer span.End()

		out, metadata, err = next.HandleInitialize(ctx, in)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		return out, metadata, err
	}),
		middleware.After)
}

func (m otelMiddlewares) deserializeMiddleware(stack *middleware.Stack) error {
	return stack.Deserialize.Add(middleware.DeserializeMiddlewareFunc("OTelDeserializeMiddleware", func(
		ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (
		out middleware.DeserializeOutput, metadata middleware.Metadata, err error) {
		out, metadata, err = next.HandleDeserialize(ctx, in)
		resp, ok := out.RawResponse.(*smithyhttp.Response)
		if !ok {
			// No raw response to wrap with.
			return out, metadata, err
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(resp.StatusCode))

		requestID, ok := v2Middleware.GetRequestIDMetadata(metadata)
		if ok {
			span.SetAttributes(RequestIDAttr(requestID))
		}

		return out, metadata, err
	}),
		middleware.Before)
}

// AppendMiddlewares attaches OTel middlewares to the AWS Go SDK V2 for instrumentation.
// OTel middlewares can be appended to either all aws clients or a specific operation.
// Please see more details in https://aws.github.io/aws-sdk-go-v2/docs/middleware/
func AppendMiddlewares(apiOptions *[]func(*middleware.Stack) error, opts ...Option) {
	cfg := config{
		TracerProvider: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	m := otelMiddlewares{tracer: cfg.TracerProvider.Tracer(tracerName,
		trace.WithInstrumentationVersion(SemVersion()))}
	*apiOptions = append(*apiOptions, m.initializeMiddlewareBefore, m.initializeMiddlewareAfter, m.deserializeMiddleware)
}
