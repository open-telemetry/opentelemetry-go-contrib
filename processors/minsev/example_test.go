// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/log"
	logsdk "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/contrib/processors/minsev"
)

type EnvSeverity struct {
	Var string
}

func (s EnvSeverity) Severity() log.Severity {
	var sev minsev.Severity
	_ = sev.UnmarshalText([]byte(os.Getenv(s.Var)))
	return sev.Severity() // Default to SeverityInfo if not set or error.
}

// This example demonstrates how to use a Severitier that reads from
// an environment variable.
func ExampleSeveritier_environment() {
	const key = "LOG_LEVEL"
	// Mock an environmental variable setup that would be done externally.
	_ = os.Setenv(key, "error")

	// Existing processor that emits telemetry.
	var processor logsdk.Processor = logsdk.NewBatchProcessor(nil)

	// Wrap the processor so that it filters by severity level defined
	// via environmental variable.
	processor = minsev.NewLogProcessor(processor, EnvSeverity{key})
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

// This example demonstrates how to use a Severitier that reads from a JSON
// configuration.
func ExampleSeveritier_json() {
	// Example JSON configuration that specifies the minimum severity level.
	// This would be provided by the application user.
	const jsonConfig = `{"log_level":"error"}`

	var config struct {
		Severity minsev.Severity `json:"log_level"`
	}
	if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
		panic(err)
	}

	// Existing processor that emits telemetry.
	var processor logsdk.Processor = logsdk.NewBatchProcessor(nil)

	// Wrap the processor so that it filters by severity level defined
	// in the JSON configuration. Note that the severity level itself is a
	// Severitier implementation.
	processor = minsev.NewLogProcessor(processor, config.Severity)
	lp := logsdk.NewLoggerProvider(logsdk.WithProcessor(processor))

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
