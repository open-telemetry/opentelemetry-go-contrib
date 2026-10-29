// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azureappservice_test

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/detectors/azure/azureappservice"
)

func ExampleNewResourceDetector() {
	res, err := resource.New(
		context.Background(),
		resource.WithDetectors(azureappservice.NewResourceDetector()),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
}
