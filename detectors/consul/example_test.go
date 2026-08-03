// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consul_test

import (
	"context"
	"log"
	"regexp"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/detectors/consul"
)

func ExampleNewResourceDetector() {
	res, err := resource.New(
		context.Background(),
		resource.WithDetectors(consul.NewResourceDetector()),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
	// Use tp to create tracers ...
}

func ExampleWithMetaKeyFilter() {
	// Of the node meta entries reported by the agent, only "rack" is emitted,
	// as an attribute with the key "consul.meta.rack".
	detector := consul.NewResourceDetector(
		consul.WithMetaKeyFilter(regexp.MustCompile("^rack$").MatchString),
	)

	res, err := resource.New(context.Background(), resource.WithDetectors(detector))
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
	// Use tp to create tracers ...
}
