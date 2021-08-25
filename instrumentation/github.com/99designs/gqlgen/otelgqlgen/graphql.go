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

package otelgqlgen

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen"
)

// Tracer implements graphql HandlerExtension , OperationInterceptor and FieldInterceptor.
type Tracer struct {
	config *config
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationInterceptor
	graphql.FieldInterceptor
} = &Tracer{}

// NewTracer returns 99designs graphql HandlerExtension which will trace incoming requests.
func NewTracer(opts ...Option) *Tracer {
	cfg := &config{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	if cfg.tracerProvider == nil {
		cfg.tracerProvider = otel.GetTracerProvider()
	}
	if cfg.propagators == nil {
		cfg.propagators = otel.GetTextMapPropagator()
	}

	return &Tracer{config: cfg}
}

//ExtensionName which may be shown in stats and logging.
func (t *Tracer) ExtensionName() string {
	return "OpenTelemetry"
}

//Validate is called when adding an extension to the server, opentelemetry should not verify the GQL schema .
func (t *Tracer) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

// InterceptOperation is called for each incoming query, for basic requests the writer will be invoked once,
// for subscriptions it will be invoked multiple times.
func (t *Tracer) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	oc := graphql.GetOperationContext(ctx)

	ctx, span := t.config.tracerProvider.Tracer(tracerName).Start(ctx, oc.Operation.Name)
	ctx = trace.ContextWithSpan(ctx, span)
	defer span.End()
	attr := []attribute.KeyValue{
		attribute.Any("resolver.rawQuery", oc.RawQuery),
		attribute.Any("kind", "server"),
		attribute.Any("component", "gqlgen"),
	}
	span.SetAttributes(attr...)

	return next(ctx)
}

// InterceptField called around each field. expand the GQL schema if it encounters a Expand the GQL schema if it encounters a separate resolver,
//it will process opentelemetry span.
func (t *Tracer) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	fc := graphql.GetFieldContext(ctx)
	ctx, span := t.config.tracerProvider.Tracer(tracerName).Start(ctx, fc.Object+"_"+fc.Field.Name)
	defer span.End()

	attr := []attribute.KeyValue{
		attribute.Any("resolver.object", fc.Object),
		attribute.Any("resolver.field", fc.Field.Name),
	}
	span.SetAttributes(attr...)

	res, err := next(ctx)
	errList := graphql.GetFieldErrors(ctx, fc)
	if len(errList) != 0 {

		span.SetStatus(codes.Error, errList.Error())
		for idx, err := range errList {
			attr := []attribute.KeyValue{
				attribute.Any(fmt.Sprintf("error.%d.message", idx), err.Error()),
				attribute.Any(fmt.Sprintf("error.%d.kind", idx), fmt.Sprintf("%T", err)),
			}
			span.SetAttributes(attr...)
		}
	}

	return res, err
}
