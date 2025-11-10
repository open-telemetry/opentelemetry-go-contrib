// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetect_test

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/autodetect"
)

// This environment variable is expected to be a comma-separated list of
// detectors the user wants for the purpose of the example. It can take any
// form a user want to parse.
const envVar = "RESOURCE_DETECTORS"

func init() {
	id := autodetect.ID("my.env.var.detector")
	autodetect.Register(id, func() resource.Detector {
		return MyDetector{}
	})

	_ = os.Setenv(envVar, "host,telemetry.sdk,my.env.var.detector")
}

func ExampleDetector_envVar() {
	// This example shows how to parse resource.Detectors from an environment
	// variable.

	names := strings.Split(os.Getenv(envVar), ",")
	ids := make([]autodetect.ID, 0, len(names))
	for _, name := range names {
		ids = append(ids, autodetect.ID(name))
	}

	detector, err := autodetect.Detector(ids...)
	if err != nil {
		// Handle the error if parsing fails.
		panic(err)
	}

	// Use the detector as needed.

	res, err := detector.Detect(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Print(enc.Encode(res.Iter()))
	// Output:
	//   host.name my.key telemetry.sdk.language telemetry.sdk.name telemetry.sdk.version
}
