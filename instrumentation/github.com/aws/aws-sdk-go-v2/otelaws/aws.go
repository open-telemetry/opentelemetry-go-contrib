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
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

type config struct {
	TracerProvider trace.TracerProvider
	Propagators    propagation.TextMapPropagator
}

// Option specifies instrumentation configuration options.
type Option func(*config)

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return func(cfg *config) {
		cfg.Propagators = propagators
	}
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(cfg *config) {
		cfg.TracerProvider = provider
	}
}

// AppendOtelMiddlewares attaches otel middlewares to aws go sdk v2 for instrumentation.
// Otel middlewares can be appended to either all aws clients or a specific operation.
// Please see more details in https://aws.github.io/aws-sdk-go-v2/docs/middleware/
func AppendOtelMiddlewares(apiOptions *[]func(*middleware.Stack) error, opts ...Option) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}

	awsTracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(contrib.SemVersion()),
	)

	*apiOptions = append(*apiOptions,
		func(stack *middleware.Stack) error {
			return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OtelSpanCreator", func(
				ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
				out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

				commonLabels := []label.KeyValue{
					label.String("aws.operation", v2Middleware.GetOperationName(ctx)),
					label.String("aws.region", v2Middleware.GetRegion(ctx)),
					label.String("aws.service", v2Middleware.GetServiceID(ctx)),
				}
				opts := []trace.SpanOption{
					trace.WithAttributes(commonLabels...),
					trace.WithSpanKind(trace.SpanKindClient),
				}
				ctx, span := awsTracer.Start(ctx, v2Middleware.GetServiceID(ctx), opts...)
				defer span.End()

				out, metadata, err = next.HandleInitialize(ctx, in)
				if err != nil {
					span.RecordError(err)
				}

				return out, metadata, err
			}),
				middleware.After)
		},
		func(stack *middleware.Stack) error {
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
				statusCode := resp.StatusCode
				span.SetAttributes(label.Int("http.status_code", statusCode))

				requestID, ok := v2Middleware.GetRequestIDMetadata(metadata)
				if ok {
					span.SetAttributes(label.String("aws.request_id", requestID))
				}

				return out, metadata, err
			}),
				middleware.Before)
		})
}
