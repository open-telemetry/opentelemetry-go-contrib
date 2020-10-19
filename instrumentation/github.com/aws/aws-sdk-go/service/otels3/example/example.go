package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	otelmetric "go.opentelemetry.io/otel/api/metric"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	oteltracestdout "go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	obsvsS3 "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/otels3"
)

type mockS3Client struct {
	s3iface.S3API
}

func (s *mockS3Client) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (s *mockS3Client) GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

func (s *mockS3Client) DeleteObjectWithContext(ctx aws.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}

func (s *mockS3Client) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	fmt.Printf("UnInstrumentedMethod `DeleteObject` called")
	return &s3.DeleteObjectOutput{}, nil
}

func main() {
	tracerProvider := initTracer()
	meterProvider := initMeter()

	client := obsvsS3.NewInstrumentedS3Client(
		&mockS3Client{},
		config.WithTracerProvider(tracerProvider),
		config.WithMetricProvider(meterProvider),
		config.WithSpanCorrelationInMetrics(true),
	)
	tracer := tracerProvider.Tracer("http-tracer")

	outerSpanCtx, span := tracer.Start(
		context.Background(),
		"http_request_served",
	)
	span.End()

	_, _ = client.PutObjectWithContext(outerSpanCtx, &s3.PutObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("010101"),
		Body:   bytes.NewReader([]byte("foo")),
	})
	_, _ = client.GetObjectWithContext(outerSpanCtx, &s3.GetObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("bar"),
	})
	_, _ = client.DeleteObjectWithContext(outerSpanCtx, &s3.DeleteObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("010101"),
	})
	_, _ = client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("010101"),
	})

	time.Sleep(time.Second * 15)
}

func initTracer() oteltrace.TracerProvider {
	exporter, _ := oteltracestdout.NewExporter(oteltracestdout.WithPrettyPrint())
	cfg := sdktrace.Config{
		DefaultSampler: sdktrace.AlwaysSample(),
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithConfig(cfg),
		sdktrace.WithSyncer(exporter),
	)

	return tp
}

func initMeter() otelmetric.MeterProvider {
	selector := simple.NewWithExactDistribution()
	exporter, _ := oteltracestdout.NewExporter(oteltracestdout.WithPrettyPrint())
	pusher := push.New(
		processor.New(selector, metric.PassThroughExporter),
		exporter,
	)
	pusher.Start()

	return pusher.MeterProvider()
}
