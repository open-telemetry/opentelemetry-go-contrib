// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nova_test

import (
	"context"
	"log"
	"regexp"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/detectors/openstack/nova"
)

func ExampleNewResourceDetector() {
	res, err := resource.New(
		context.Background(),
		resource.WithDetectors(nova.NewResourceDetector()),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
	// Use tp to create tracers ...
}

func ExampleWithMetaKeyFilter() {
	// Emit every instance metadata key starting with "otel_" as an
	// openstack.nova.meta.<key> attribute.
	keys := regexp.MustCompile(`^otel_`)

	res, err := resource.New(
		context.Background(),
		resource.WithDetectors(nova.NewResourceDetector(
			nova.WithMetaKeyFilter(keys.MatchString),
		)),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
	// Use tp to create tracers ...
}
