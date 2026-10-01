// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azureaks_test

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"go.opentelemetry.io/contrib/detectors/azure/azureaks"
)

func ExampleNewResourceDetector() {
	res, err := resource.New(
		context.Background(),
		resource.WithDetectors(azureaks.NewResourceDetector()),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
	// Use tp to create tracers ...
}

func ExampleWithAttributeFilter() {
	// Detect Azure Kubernetes Service without reporting the cluster name.
	detector := azureaks.NewResourceDetector(
		azureaks.WithAttributeFilter(
			attribute.NewDenyKeysFilter(semconv.K8SClusterNameKey),
		),
	)

	res, err := resource.New(context.Background(), resource.WithDetectors(detector))
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
	// Use tp to create tracers ...
}
