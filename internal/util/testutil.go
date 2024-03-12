// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package util // import "go.opentelemetry.io/contrib/internal/util"

import (
	"fmt"
	"os"
)

func IntegrationShouldRun(name string) {
	if val, ok := os.LookupEnv("INTEGRATION"); !ok || val != name {
		fmt.Println(
			"--- SKIP: to enable integration test, set the INTEGRATION environment variable",
			"to",
			fmt.Sprintf("\"%s\"", name),
		)
		os.Exit(0) //nolint revive  // Signal test was successfully skipped.
	}
}
