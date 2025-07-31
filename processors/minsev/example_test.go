// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/log"
	logsdk "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/contrib/processors/minsev"
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
	// Mock an environmental variable setup that would be done externally.
	_ = os.Setenv(key, "error")

	// Existing processor that emits telemetry.
	var processor logsdk.Processor = logsdk.NewBatchProcessor(nil)

	// Wrap the processor so that it filters by severity level defined
	// via environmental variable.
	processor = minsev.NewLogProcessor(processor, EnvSeverity{})
	lp := logsdk.NewLoggerProvider(
		logsdk.WithProcessor(processor),
	)

	// Show that Logs API respects the minimum severity level processor.
	l := lp.Logger("ExampleSeveritier")

	ctx := context.Background()
	params := log.EnabledParameters{Severity: log.SeverityDebug}
	fmt.Println(l.Enabled(ctx, params))

	params.Severity = log.SeverityError
	fmt.Println(l.Enabled(ctx, params))

	// Output:
	// false
	// true
}
