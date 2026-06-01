// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package env_test

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/detectors/env"
)

func ExampleNewResourceDetector() {
	envResourceDetector := env.NewResourceDetector()
	resource, err := envResourceDetector.Detect(context.Background())
	if err != nil {
		panic(err)
	}

	// Now, you can use the resource (e.g. pass it to a tracer or meter provider).
	fmt.Println(resource.SchemaURL())
}
