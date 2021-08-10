package graphql

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
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/99designs/graphql"
)

type tracer struct {
	config *config
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationInterceptor
	graphql.FieldInterceptor
} = &tracer{}

func NewTracer(opts ...Option) *tracer {
	cfg := &config{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}
	if cfg.Propagators == nil {
		cfg.Propagators = otel.GetTextMapPropagator()
	}

	return &tracer{config: cfg}
}

func (t *tracer) ExtensionName() string {
	return "OpenTelemetry"
}

func (t *tracer) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (t *tracer) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	oc := graphql.GetOperationContext(ctx)

	ctx, span := t.config.TracerProvider.Tracer(tracerName).Start(ctx, oc.Operation.Name)
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

func (t *tracer) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	fc := graphql.GetFieldContext(ctx)
	ctx, span := t.config.TracerProvider.Tracer(tracerName).Start(ctx, fc.Object+"_"+fc.Field.Name)
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
