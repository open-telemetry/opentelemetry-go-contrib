// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev // import "go.opentelemetry.io/contrib/processors/minsev"

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/log"
)

const key = "OTEL_LOG_LEVEL"

var getSeverity = sync.OnceValue(func() log.Severity {
	conv := map[string]log.Severity{
		"":      log.SeverityInfo, // Default to SeverityInfo for unset.
		"debug": log.SeverityDebug,
		"info":  log.SeverityInfo,
		"warn":  log.SeverityWarn,
		"error": log.SeverityError,
	}
	// log.SeverityUndefined for unknown values.
	return conv[strings.ToLower(os.Getenv(key))]
})

type EnvSeverity struct{}

func (EnvSeverity) Severity() log.Severity { return getSeverity() }

func ExampleSeveritier() {
	// Mock an environment variable setup that would be done externally.
	_ = os.Setenv(key, "error")

	p := NewLogProcessor(&processor{}, EnvSeverity{})

	ctx := context.Background()
	params := log.EnabledParameters{Severity: log.SeverityDebug}
	fmt.Println(p.Enabled(ctx, params))

	params.Severity = log.SeverityError
	fmt.Println(p.Enabled(ctx, params))

	// Output:
	// false
	// true
}
