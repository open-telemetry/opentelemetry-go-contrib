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

package main

import (
	"bytes"
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/config"
	obsvsS3 "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/otels3"
	mocks "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/otels3/mocks"
	otelmetric "go.opentelemetry.io/otel/api/metric"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	oteltracestdout "go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	tracerProvider := initTracer()
	meterProvider := initMeter()

	client, err := obsvsS3.NewInstrumentedS3Client(
		&mocks.MockS3Client{},
		config.WithTracerProvider(tracerProvider),
		config.WithMetricProvider(meterProvider),
		config.WithSpanCorrelationInMetrics(true),
	)

	if err != nil {
		panic(err)
	}

	tracer := tracerProvider.Tracer("http-tracer")

	outerSpanCtx, span := tracer.Start(
		context.Background(),
		"http_request_served",
	)
	defer span.End()

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
