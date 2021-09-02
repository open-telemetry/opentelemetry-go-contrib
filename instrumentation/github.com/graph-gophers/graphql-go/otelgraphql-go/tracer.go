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

package otelgraphqlgo

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/introspection"
	"github.com/graph-gophers/graphql-go/trace"

	otelcontrib "go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type OpenTelemetryTracer struct {
	Tracer oteltrace.Tracer
}

const tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo"

func NewOpenTelemetryTracer(opts ...Option) OpenTelemetryTracer {
	cfg := config{}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion(otelcontrib.SemVersion()),
	)
	return OpenTelemetryTracer{Tracer: tracer}
}

func (t OpenTelemetryTracer) TraceQuery(ctx context.Context,
	queryString string, operationName string,
	variables map[string]interface{},
	varTypes map[string]*introspection.Type) (context.Context, trace.TraceQueryFinishFunc) {
	spanCtx, span := t.Tracer.Start(ctx, "GraphQL request",
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)
	span.SetAttributes(attribute.String("trace.operation", "request"))
	span.SetAttributes(attribute.String("graphql.query", queryString))

	if operationName != "" {
		span.SetAttributes(attribute.String("graphql.operationName", operationName))
	}

	if len(variables) != 0 {
		for name, value := range variables {
			span.SetAttributes(attribute.Any("graphql.variables."+name, value))
		}
	}

	return spanCtx, func(errs []*errors.QueryError) {
		if len(errs) > 0 {
			msg := errs[0].Error()
			if len(errs) > 1 {
				msg += fmt.Sprintf(" (and %d more errors)", len(errs)-1)
			}
			span.RecordError(errs[0])
			span.SetStatus(codes.Error, msg)
		}
		span.End()
	}
}

func (t OpenTelemetryTracer) TraceField(ctx context.Context,
	label,
	typeName,
	fieldName string,
	trivial bool,
	args map[string]interface{}) (context.Context, trace.TraceFieldFinishFunc) {
	if trivial {
		return ctx, func(*errors.QueryError) {}
	}

	spanCtx, span := t.Tracer.Start(ctx, label)
	span.SetAttributes(attribute.String("trace.operation", "field"))
	span.SetAttributes(attribute.String("graphql.type", typeName))
	span.SetAttributes(attribute.String("graphql.field", fieldName))
	for name, value := range args {
		span.SetAttributes(attribute.Any("graphql.args."+name, value))
	}

	return spanCtx, func(err *errors.QueryError) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

func (t OpenTelemetryTracer) TraceValidation(ctx context.Context) trace.TraceValidationFinishFunc {

	_, span := t.Tracer.Start(ctx, "Validate query")
	span.SetAttributes(attribute.String("trace.operation", "validation"))

	return func(errs []*errors.QueryError) {
		if len(errs) > 0 {
			msg := errs[0].Error()
			if len(errs) > 1 {
				msg += fmt.Sprintf(" (and %d more errors)", len(errs)-1)
			}
			span.RecordError(errs[0])
			span.SetStatus(codes.Error, msg)
		}
		span.End()
	}
}
