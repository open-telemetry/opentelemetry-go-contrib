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

package otels3

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var instrumentationName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/s3/otels3"

type instrumentedS3 struct {
	s3iface.S3API
	tracer            trace.Tracer
	meter             metric.Meter
	textMapPropagator propagation.TextMapPropagator
	counters          *counters
	recorders         *recorders
	spanCorrelation   bool
}

type counters struct {
	operation metric.Int64Counter
}

type recorders struct {
	operationDuration metric.Float64ValueRecorder
}

// appendSpanAndTraceIDFromSpan extracts the trace id and span id from a span using the context field.
// It returns a list of attributes with the span id and trace id appended to the original slice.
// Any modification on the original slice will modify the returned slice.
func appendSpanAndTraceIDFromSpan(attrs []label.KeyValue, span trace.Span) []label.KeyValue {
	return append(attrs,
		label.String("event.spanId", span.SpanContext().SpanID.String()),
		label.String("event.traceId", span.SpanContext().TraceID.String()),
	)
}

// PutObjectWithContext wraps the embedded PutObjectWithContext method with telemetry signal instrumentation.
func (s *instrumentedS3) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	startTime := time.Now()
	destination := aws.StringValue(input.Bucket)
	attrs := createAttributes(destination, operationPutObject)

	spanCtx, span := s.tracer.Start(
		ctx,
		fmt.Sprintf("%s.%s", destination, operationPutObject),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	output, err := s.S3API.PutObjectWithContext(ctx, input, opts...)
	callReturnTime := time.Now()
	defer span.End(trace.WithTimestamp(callReturnTime))

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
		span.SetAttributes(labelStatusFailure)
		span.SetStatus(codes.Error, err.Error())
	} else {
		attrs = append(attrs, labelStatusSuccess)
		span.SetAttributes(labelStatusSuccess)
		span.SetStatus(codes.Ok, "")
	}

	if s.spanCorrelation {
		attrs = appendSpanAndTraceIDFromSpan(attrs, span)
	}

	s.recorders.operationDuration.Record(
		spanCtx,
		float64(callReturnTime.Sub(startTime).Microseconds()),
		attrs...,
	)
	s.counters.operation.Add(ctx, 1, attrs...)

	return output, err
}

// GetObjectWithContext wraps the embedded GetObjectWithContext method with telemetry signal instrumentation.
func (s *instrumentedS3) GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	startTime := time.Now()
	destination := aws.StringValue(input.Bucket)
	attrs := createAttributes(destination, operationGetObject)

	spanCtx, span := s.tracer.Start(
		ctx,
		fmt.Sprintf("%s.%s", destination, operationGetObject),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	output, err := s.S3API.GetObjectWithContext(ctx, input, opts...)
	callReturnTime := time.Now()
	defer span.End(trace.WithTimestamp(callReturnTime))

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
		span.SetAttributes(labelStatusFailure)
		span.SetStatus(codes.Error, err.Error())
	} else {
		attrs = append(attrs, labelStatusSuccess)
		span.SetAttributes(labelStatusSuccess)
		span.SetStatus(codes.Ok, "")
	}

	if s.spanCorrelation {
		attrs = appendSpanAndTraceIDFromSpan(attrs, span)
	}

	s.recorders.operationDuration.Record(
		spanCtx,
		float64(callReturnTime.Sub(startTime).Microseconds()),
		attrs...,
	)
	s.counters.operation.Add(ctx, 1, attrs...)

	return output, err
}

// DeleteObjectWithContext wraps the embedded DeleteObjectWithContext method with telemetry signal instrumentation.
func (s *instrumentedS3) DeleteObjectWithContext(ctx aws.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	startTime := time.Now()
	destination := aws.StringValue(input.Bucket)
	attrs := createAttributes(destination, operationDeleteObject)

	spanCtx, span := s.tracer.Start(
		ctx,
		fmt.Sprintf("%s.%s", destination, operationDeleteObject),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	output, err := s.S3API.DeleteObjectWithContext(ctx, input, opts...)
	callReturnTime := time.Now()
	defer span.End(trace.WithTimestamp(callReturnTime))

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
		span.SetAttributes(labelStatusFailure)
		span.SetStatus(codes.Error, err.Error())
	} else {
		attrs = append(attrs, labelStatusSuccess)
		span.SetAttributes(labelStatusSuccess)
		span.SetStatus(codes.Ok, "")
	}

	if s.spanCorrelation {
		attrs = appendSpanAndTraceIDFromSpan(attrs, span)
	}

	s.recorders.operationDuration.Record(
		spanCtx,
		float64(callReturnTime.Sub(startTime).Microseconds()),
		attrs...,
	)
	s.counters.operation.Add(ctx, 1, attrs...)

	return output, err
}

func createCounters(meter metric.Meter) *counters {
	operationCounter, err := meter.NewInt64Counter("aws.s3.operation")
	if err != nil {
		otel.Handle(err)
	}
	return &counters{operation: operationCounter}
}

func createRecorders(meter metric.Meter) *recorders {
	execTimeRecorder, err := meter.NewFloat64ValueRecorder("aws.s3.operation.duration", metric.WithUnit("μs"))
	if err != nil {
		otel.Handle(err)
	}
	return &recorders{operationDuration: execTimeRecorder}
}

// NewInstrumentedS3Client returns an instrumentation wrapped s.
func NewInstrumentedS3Client(s s3iface.S3API, opts ...Option) (s3iface.S3API, error) {
	if s == nil || reflect.ValueOf(s).IsNil() {
		return &instrumentedS3{}, errors.New("interface must be set")
	}

	cfg := config{
		TracerProvider:    otel.GetTracerProvider(),
		MeterProvider:     otel.GetMeterProvider(),
		TextMapPropagator: otel.GetTextMapPropagator(),
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	tracer := cfg.TracerProvider.Tracer(instrumentationName)
	meter := cfg.MeterProvider.Meter(instrumentationName)

	return &instrumentedS3{
		S3API:             s,
		meter:             meter,
		tracer:            tracer,
		textMapPropagator: cfg.TextMapPropagator,
		counters:          createCounters(meter),
		recorders:         createRecorders(meter),
		spanCorrelation:   cfg.SpanCorrelation,
	}, nil
}
