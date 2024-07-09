// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package isolate_test

import (
	"go.opentelemetry.io/contrib/processors/isolate"
	"go.opentelemetry.io/otel/sdk/log"
)

func Example() {
	// Log processing pipelines that process and emit telemetry.
	var p1 log.Processor
	var p2 log.Processor
	var p3 log.Processor

	// Register the processors using
	// isolate.NewLogProcessor and the log.WithProcessor option
	// so that the log records are not shared between pipelines.
	_ = log.NewLoggerProvider(
		log.WithProcessor(isolate.NewLogProcessor(p1)),
		log.WithProcessor(isolate.NewLogProcessor(p2)),
		log.WithProcessor(isolate.NewLogProcessor(p3)),
	)
}
