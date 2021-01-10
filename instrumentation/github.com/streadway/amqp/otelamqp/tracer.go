package otelamqp

import (
	"context"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("amqp")

func StartProducerSpan(hdrs amqp.Table, ctx context.Context) trace.Span {
	c := amqpHeadersCarrier(hdrs)
	otel.GetTextMapPropagator().Extract(ctx, c)

	attrs := []label.KeyValue{
		semconv.MessagingSystemKey.String("amqp"),
	}
	opts := []trace.SpanOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	}

	ctx, span := tracer.Start(ctx, "amqp.producer", opts...)

	return span
}

func EndProducerSpan(span trace.Span, err error) {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func StartConsumerSpan(hdrs amqp.Table, ctx context.Context) (trace.Span, context.Context) {
	c := amqpHeadersCarrier(hdrs)

	otel.GetTextMapPropagator().Extract(ctx, c)
	opts := []trace.SpanOption{
		trace.WithSpanKind(trace.SpanKindConsumer),
	}

	ctx, span := tracer.Start(ctx, "amqp.consumer", opts...)

	return span, ctx
}

func EndConsumerSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
	}
	defer span.End()
}
