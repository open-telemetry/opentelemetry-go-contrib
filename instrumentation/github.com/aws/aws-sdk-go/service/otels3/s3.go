package otels3

import (
	"fmt"
	"reflect"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/config"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/helper"
	"go.opentelemetry.io/otel"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

var instrumentationName = "github.com/aws/aws-sdk-go/aws/service/s3"

type instrumentedS3 struct {
	s3iface.S3API
	tracer                   trace.Tracer
	meter                    metric.Meter
	propagators              otel.TextMapPropagator
	counters                 *counters
	recorders                *recorders
	spanCorrelationInMetrics bool
}

type counters struct {
	operation metric.Int64Counter
}

type recorders struct {
	operationDuration metric.Float64ValueRecorder
}

func (s *instrumentedS3) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	startTime := time.Now()
	destination := aws.StringValue(input.Bucket)
	attrs := createAttributes(destination, operationPutObject)

	spanCtx, span := s.tracer.Start(
		ctx,
		fmt.Sprintf("%s.%s", destination, operationPutObject),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithTimestamp(startTime),
		trace.WithAttributes(attrs...),
	)
	defer span.End()

	output, err := s.S3API.PutObjectWithContext(ctx, input, opts...)

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
	} else {
		attrs = append(attrs, labelStatusSuccess)
	}

	if s.spanCorrelationInMetrics {
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

func (s *instrumentedS3) GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	startTime := time.Now()
	destination := aws.StringValue(input.Bucket)
	attrs := createAttributes(destination, operationGetObject)

	spanCtx, span := s.tracer.Start(
		ctx,
		fmt.Sprintf("%s.%s", destination, operationGetObject),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithTimestamp(startTime),
		trace.WithAttributes(attrs...),
	)
	defer span.End()

	output, err := s.S3API.GetObjectWithContext(ctx, input, opts...)

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
	} else {
		attrs = append(attrs, labelStatusSuccess)
	}

	if s.spanCorrelationInMetrics {
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

func (s *instrumentedS3) DeleteObjectWithContext(ctx aws.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	startTime := time.Now()
	destination := aws.StringValue(input.Bucket)
	attrs := createAttributes(destination, operationDeleteObject)

	spanCtx, span := s.tracer.Start(
		ctx,
		fmt.Sprintf("%s.%s", destination, operationDeleteObject),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithTimestamp(startTime),
		trace.WithAttributes(attrs...),
	)
	defer span.End()

	output, err := s.S3API.DeleteObjectWithContext(ctx, input, opts...)

	if err != nil {
		attrs = append(attrs, labelStatusFailure)
	} else {
		attrs = append(attrs, labelStatusSuccess)
	}

	if s.spanCorrelationInMetrics {
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
	operationCounter, _ := meter.NewInt64Counter("storage.s3.operation")
	return &counters{operation: operationCounter}
}

func createRecorders(meter metric.Meter) *recorders {
	execTimeRecorder, _ := meter.NewFloat64ValueRecorder("storage.operation.duration_μs")
	return &recorders{operationDuration: execTimeRecorder}
}

func NewInstrumentedS3Client(s s3iface.S3API, opts ...config.Option) (s3iface.S3API, error) {
	if s == nil || reflect.ValueOf(s).IsNil() {
		return &instrumentedS3{}, fmt.Errorf("interface must be set")
	}

	cfg := config.Config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = global.TracerProvider()
	}
	tracer := cfg.TracerProvider.Tracer(instrumentationName)
	if cfg.MetricProvider == nil {
		cfg.MetricProvider = global.MeterProvider()
	}
	meter := cfg.MetricProvider.Meter(instrumentationName)

	if cfg.Propagators == nil {
		cfg.Propagators = global.TextMapPropagator()
	}

	return &instrumentedS3{
		S3API:                    s,
		meter:                    meter,
		tracer:                   tracer,
		propagators:              cfg.Propagators,
		counters:                 createCounters(meter),
		recorders:                createRecorders(meter),
		spanCorrelationInMetrics: cfg.SpanCorrelationInMetrics,
	}, nil
}
