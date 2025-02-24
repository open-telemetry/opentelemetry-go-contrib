// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev_test // import "go.opentelemetry.io/contrib/processors/minsev"

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/contrib/processors/minsev"
	logapi "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

const key = "OTEL_LOG_LEVEL"

var getSeverity = sync.OnceValue(func() logapi.Severity {
	conv := map[string]logapi.Severity{
		"":      logapi.SeverityInfo, // Default to SeverityInfo for unset.
		"debug": logapi.SeverityDebug,
		"info":  logapi.SeverityInfo,
		"warn":  logapi.SeverityWarn,
		"error": logapi.SeverityError,
	}
	// log.SeverityUndefined for unknown values.
	return conv[strings.ToLower(os.Getenv(key))]
})

type EnvSeverity struct{}

func (EnvSeverity) Severity() logapi.Severity { return getSeverity() }

func ExampleSeveritier() {
	// Mock an environment variable setup that would be done externally.
	_ = os.Setenv(key, "error")

	// Existing processor that emits telemetry.
	var processor log.Processor = log.NewBatchProcessor(nil)

	// Wrap the processor so that it filters by severity level defined
	// via environental variable.
	processor = minsev.NewLogProcessor(processor, EnvSeverity{})
	lp := log.NewLoggerProvider(
		log.WithProcessor(processor),
	)

	// Show that Logs API respects the minimum severity level processor.
	l := lp.Logger("ExampleSeveritier")

	ctx := context.Background()
	params := logapi.EnabledParameters{Severity: logapi.SeverityDebug}
	fmt.Println(l.Enabled(ctx, params))

	params.Severity = logapi.SeverityError
	fmt.Println(l.Enabled(ctx, params))

	// Output:
	// false
	// true
}
