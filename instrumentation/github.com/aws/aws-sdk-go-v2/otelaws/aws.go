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

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

type oTelMiddlewares struct {
	tracer trace.Tracer
}

func (m oTelMiddlewares) initializeMiddleware(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OtelSpanCreator", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

		opts := []trace.SpanOption{
			trace.WithSpanKind(trace.SpanKindClient),
		}
		ctx, span := m.tracer.Start(ctx, v2Middleware.GetServiceID(ctx), opts...)
		defer span.End()

		out, metadata, err = next.HandleInitialize(ctx, in)
		if err != nil {
			span.RecordError(err)
		}

		return out, metadata, err
	}),
		middleware.Before)
}

func (m oTelMiddlewares) deserializeMiddleware(stack *middleware.Stack) error {
	return stack.Deserialize.Add(middleware.DeserializeMiddlewareFunc("OtelSpanDecorator", func(
		ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (
		out middleware.DeserializeOutput, metadata middleware.Metadata, err error) {
		out, metadata, err = next.HandleDeserialize(ctx, in)
		resp, ok := out.RawResponse.(*smithyhttp.Response)
		if !ok {
			// No raw response to wrap with.
			return out, metadata, err
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(resp.StatusCode),
			ServiceAttr(v2Middleware.GetServiceID(ctx)),
			RegionAttr(v2Middleware.GetRegion(ctx)),
			OperationAttr(v2Middleware.GetOperationName(ctx)))

		requestID, ok := v2Middleware.GetRequestIDMetadata(metadata)
		if ok {
			span.SetAttributes(RequestIDAttr(requestID))
		}

		return out, metadata, err
	}),
		middleware.Before)
}

// AppendOtelMiddlewares attaches otel middlewares to aws go sdk v2 for instrumentation.
// Otel middlewares can be appended to either all aws clients or a specific operation.
// Please see more details in https://aws.github.io/aws-sdk-go-v2/docs/middleware/
func AppendMiddlewares(apiOptions *[]func(*middleware.Stack) error, opts ...Option) {
	cfg := config{
		Propagators:    otel.GetTextMapPropagator(),
		TracerProvider: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt.Apply(&cfg)
	}

	m := oTelMiddlewares{tracer: cfg.TracerProvider.Tracer(tracerName,
		trace.WithInstrumentationVersion(contrib.SemVersion()))}
	*apiOptions = append(*apiOptions, m.initializeMiddleware, m.deserializeMiddleware)
}
