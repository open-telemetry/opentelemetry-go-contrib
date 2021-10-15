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

package otelconfluent

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	// tracerName is the technical name of the tracer.
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go"

	// traceIdentifierHeaderName is the name of the header attach to Kafka message
	// that contains the parent trace identifier.
	traceIdentifierHeaderName = "otel-confluent-kafka-trace-id"

	// spanIdentifierHeaderName is the name of the header attach to Kafka message
	// that contains the parent span identifier.
	spanIdentifierHeaderName = "otel-confluent-kafka-span-id"
)

func endSpan(s oteltrace.Span, err error) {
	if err != nil {
		s.SetStatus(codes.Error, err.Error())
	}
	s.End()
}

func contextFromMessageHeaders(ctx context.Context, msg *kafka.Message) context.Context {
	if msg == nil {
		return ctx
	}

	traceHeaderValue := getValueForHeader(msg.Headers, traceIdentifierHeaderName)
	traceID, _ := oteltrace.TraceIDFromHex(traceHeaderValue)

	spanHeaderValue := getValueForHeader(msg.Headers, spanIdentifierHeaderName)
	spanID, _ := oteltrace.SpanIDFromHex(spanHeaderValue)

	if traceID.IsValid() && spanID.IsValid() {
		parent := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		})
		ctx = oteltrace.ContextWithSpanContext(ctx, parent)
	}

	return ctx
}

func replaceOrAddSpanContextToMessageHeaders(spanContext oteltrace.SpanContext, msg *kafka.Message) {
	if msg == nil {
		return
	}

	msg.Headers = replaceOrAddHeaderValue(msg.Headers, traceIdentifierHeaderName, []byte(spanContext.TraceID().String()))
	msg.Headers = replaceOrAddHeaderValue(msg.Headers, spanIdentifierHeaderName, []byte(spanContext.SpanID().String()))
}

func getValueForHeader(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}

	return ""
}

func replaceOrAddHeaderValue(headers []kafka.Header, key string, value []byte) []kafka.Header {
	for i, header := range headers {
		if header.Key == key {
			headers[i].Value = value
			return headers
		}
	}

	headers = append(headers, kafka.Header{
		Key:   key,
		Value: value,
	})

	return headers
}
