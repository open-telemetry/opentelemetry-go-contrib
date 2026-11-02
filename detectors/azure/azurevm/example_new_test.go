// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurevm_test

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
)

func ExampleNew() {
	azureVMResourceDetector := azurevm.New()
	res, err := azureVMResourceDetector.Detect(context.Background())
	if err != nil {
		panic(err)
	}

	// Now, you can use the resource (e.g. pass it to a tracer or meter provider).
	fmt.Println(res.SchemaURL())
}

func ExampleNewResourceDetector() {
	res, err := resource.New(
		context.Background(),
		resource.WithDetectors(azurevm.NewResourceDetector()),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
}

func ExampleWithTagKeyFilter() {
	// VM tags are not collected unless a tag key filter is configured. Here
	// only the "env" tag is emitted, as azure.tag.env.
	detector := azurevm.NewResourceDetector(
		azurevm.WithTagKeyFilter(regexp.MustCompile("^env$").MatchString),
	)

	res, err := resource.New(context.Background(), resource.WithDetectors(detector))
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
}

func ExampleWithAttributeFilter() {
	// Only the cloud.* attributes are included in the detected resource.
	detector := azurevm.NewResourceDetector(
		azurevm.WithAttributeFilter(func(kv attribute.KeyValue) bool {
			return strings.HasPrefix(string(kv.Key), "cloud.")
		}),
	)

	res, err := resource.New(context.Background(), resource.WithDetectors(detector))
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	_ = tp.Shutdown(context.Background())
}
