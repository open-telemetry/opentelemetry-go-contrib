// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurecontainerapps_test

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"
)

func ExampleNewResourceDetector() {
	azureContainerAppsResourceDetector := azurecontainerapps.NewResourceDetector()
	resource, err := azureContainerAppsResourceDetector.Detect(context.Background())
	if err != nil {
		panic(err)
	}

	// Now, you can use the resource (e.g. pass it to a tracer or meter provider).
	fmt.Println(resource.SchemaURL())
}
