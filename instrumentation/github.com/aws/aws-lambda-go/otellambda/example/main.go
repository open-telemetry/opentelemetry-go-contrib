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
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func lambdaHandler(ctx context.Context) error {
	// init aws config
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	// instrument all aws clients
	otelaws.AppendMiddlewares(&cfg.APIOptions)

	// S3
	s3Client := s3.NewFromConfig(cfg)
	input := &s3.ListBucketsInput{}
	result, err := s3Client.ListBuckets(ctx, input)
	if err != nil {
		return err
	}

	log.Println("Buckets:")
	for _, bucket := range result.Buckets {
		log.Println(*bucket.Name + ": " + bucket.CreationDate.Format("2006-01-02 15:04:05 Monday"))
	}

	// HTTP
	client := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		),
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/open-telemetry/opentelemetry-go/releases/latest", nil)
	if err != nil {
		log.Printf("failed to create http request, %v\n", err)
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		log.Printf("failed to do http request, %v\n", err)
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("failed to close http response body, %v\n", err)
		}
	}(res.Body)

	var data map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		log.Printf("failed to read http response body, %v\n", err)
	}
	log.Printf("Latest OTel Go Release is '%s'\n", data["name"])

	return nil
}

func main() {
	ctx := context.Background()

	exp, err := stdouttrace.New()
	if err != nil {
		log.Printf("failed to initialize stdout exporter %v\n", err)
		return
	}

	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(ctx)
	if err != nil {
		log.Fatalf("failed to detect lambda resources: %v\n", err)
		return
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithResource(res),
	)

	// Downstream spans use global tracer provider
	otel.SetTracerProvider(tp)

	lambda.Start(otellambda.InstrumentHandler(lambdaHandler, otellambda.WithTracerProvider(tp)))
}
