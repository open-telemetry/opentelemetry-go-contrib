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

	//"github.com/tommy-muehle/go-mnd/config"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/helper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

var instrumentationName = "github.com/aws/aws-sdk-go/aws/service/s3"

type instrumentedS3 struct {
	s3iface.S3API
	tracer          trace.Tracer
	meter           metric.Meter
	propagators     otel.TextMapPropagator
	counters        *counters
	recorders       *recorders
	spanCorrelation bool
}

type counters struct {
	operation metric.Int64Counter
}

type recorders struct {
	operationDuration metric.Float64ValueRecorder
}

// PutObjectWithContext invokes the PutObjectWithContext function with tracing instrumented.
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
	defer span.End()

	output, err := s.S3API.PutObjectWithContext(ctx, input, opts...)

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
	} else {
		attrs = append(attrs, labelStatusSuccess)
	}

	if s.spanCorrelation {
		attrs = helper.AppendSpanAndTraceIDFromSpan(attrs, span)
	}

	s.recorders.operationDuration.Record(
		spanCtx,
		float64(time.Since(startTime).Microseconds()),
		attrs...,
	)
	s.counters.operation.Add(ctx, 1, attrs...)

	return output, err
}

// GetObjectWithContext invokes the GetObjectWithContext function with tracing instrumented
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
	defer span.End()

	output, err := s.S3API.GetObjectWithContext(ctx, input, opts...)

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
	} else {
		attrs = append(attrs, labelStatusSuccess)
	}

	if s.spanCorrelation {
		attrs = helper.AppendSpanAndTraceIDFromSpan(attrs, span)
	}

	s.recorders.operationDuration.Record(
		spanCtx,
		float64(time.Since(startTime).Microseconds()),
		attrs...,
	)
	s.counters.operation.Add(ctx, 1, attrs...)

	return output, err
}

// DeleteObjectWithContext invokes the DeleteObjectWithContext function with tracing instrumented
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
	defer span.End()

	output, err := s.S3API.DeleteObjectWithContext(ctx, input, opts...)

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
	} else {
		attrs = append(attrs, labelStatusSuccess)
	}

	if s.spanCorrelation {
		attrs = helper.AppendSpanAndTraceIDFromSpan(attrs, span)
	}

	s.recorders.operationDuration.Record(
		spanCtx,
		float64(time.Since(startTime).Microseconds()),
		attrs...,
	)
	s.counters.operation.Add(ctx, 1, attrs...)

	return output, err
}

func createCounters(meter metric.Meter) *counters {
	operationCounter, _ := meter.NewInt64Counter("aws.s3.operation")
	return &counters{operation: operationCounter}
}

func createRecorders(meter metric.Meter) *recorders {
	execTimeRecorder, _ := meter.NewFloat64ValueRecorder("aws.s3.operation.duration")
	return &recorders{operationDuration: execTimeRecorder}
}

// NewInstrumentedS3Client returns an instrumentedS3 object
// containing configuration options and an S3 interface
func NewInstrumentedS3Client(s s3iface.S3API, opts ...Option) (s3iface.S3API, error) {
	if s == nil || reflect.ValueOf(s).IsNil() {
		return &instrumentedS3{}, errors.New("interface must be set")
	}

	cfg := config{
		TracerProvider: global.TracerProvider(),
		MetricProvider: global.MeterProvider(),
		Propagators:    global.TextMapPropagator(),
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	tracer := cfg.TracerProvider.Tracer(instrumentationName)
	meter := cfg.MetricProvider.Meter(instrumentationName)

	return &instrumentedS3{
		S3API:           s,
		meter:           meter,
		tracer:          tracer,
		propagators:     cfg.Propagators,
		counters:        createCounters(meter),
		recorders:       createRecorders(meter),
		spanCorrelation: cfg.SpanCorrelation,
	}, nil
}
