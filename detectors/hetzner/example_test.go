// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hetzner_test

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/detectors/hetzner"
)

func ExampleNewResourceDetector() {
	detector := hetzner.NewResourceDetector()
	res, err := detector.Detect(context.Background())
	if err != nil {
		panic(err)
	}

	// Pass the resource to a tracer or meter provider.
	fmt.Println(res.SchemaURL())
}
