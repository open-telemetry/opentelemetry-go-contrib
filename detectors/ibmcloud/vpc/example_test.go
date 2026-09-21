// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpc_test

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"
)

func ExampleNewResourceDetector() {
	res, err := resource.New(
		context.Background(),
		resource.WithDetectors(vpc.NewResourceDetector()),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
	// Use tp to create tracers ...
}
