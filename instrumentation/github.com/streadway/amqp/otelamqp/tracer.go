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

package otelamqp

import (
	"context"
	"github.com/streadway/amqp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("amqp")

func StartProducerSpan(ctx context.Context, headers amqp.Table) trace.Span {
	c := amqpHeadersCarrier(headers)
	extCtx := otel.GetTextMapPropagator().Extract(ctx, c)

	attrs := []attribute.KeyValue{
		semconv.MessagingSystemKey.String("amqp"),
	}
	opts := []trace.SpanOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	}

	spanCtx, span := tracer.Start(extCtx, "amqp.producer", opts...)

	if spanCtx != nil {
		// Inject current span context, so consumers can use it to propagate span.
		otel.GetTextMapPropagator().Inject(ctx, c)
	}
	return span
}

func EndProducerSpan(span trace.Span, err error) {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func StartConsumerSpan(ctx context.Context, headers amqp.Table) (trace.Span, context.Context) {
	c := amqpHeadersCarrier(headers)

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
