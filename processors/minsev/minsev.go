// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package minsev provides an [log.Processor] that will not log any record with
// a severity below a configured threshold.
package minsev // import "go.opentelemetry.io/contrib/processors/minsev"

import (
	"context"

	api "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

// NewLogProcessor returns a new [LogProcessor] that wraps the downstream
// [log.Processor].
//
// If downstream is nil a default No-Op [log.Processor] is used. The returned
// processor will not be enabled for nor emit any records.
func NewLogProcessor(downstream log.Processor, minimum api.Severity) *LogProcessor {
	if downstream == nil {
		downstream = defaultProcessor
	}
	return &LogProcessor{Processor: downstream, Minimum: minimum}
}

// LogProcessor is an [log.Processor] implementation that wraps another
// [log.Processor]. It will pass-through calls to OnEmit and Enabled for
// records with severity greater than or equal to a minimum. All other method
// calls are passed to the wrapped [log.Processor].
//
// If the wrapped [log.Processor] is nil, calls to the LogProcessor methods
// will panic. Use [NewLogProcessor] to create a new LogProcessor that ensures
// no panics.
type LogProcessor struct {
	log.Processor

	Minimum api.Severity
}

// Compile time assertion that LogProcessor implements log.Processor.
var _ log.Processor = (*LogProcessor)(nil)

// OnEmit passes ctx and r to the [log.Processor] that p wraps if the severity
// of record is greater than or equal to p.Minimum. Otherwise, record is
// dropped.
func (p *LogProcessor) OnEmit(ctx context.Context, record log.Record) error {
	if record.Severity() >= p.Minimum {
		return p.Processor.OnEmit(ctx, record)
	}
	return nil
}

// Enabled returns if the [log.Processor] that p wraps is enabled if the
// severity of record is greater than or equal to p.Minimum. Otherwise false is
// returned.
func (p *LogProcessor) Enabled(ctx context.Context, record log.Record) bool {
	return record.Severity() >= p.Minimum && p.Processor.Enabled(ctx, record)
}

var defaultProcessor = noopProcessor{}

type noopProcessor struct{}

func (p noopProcessor) OnEmit(context.Context, log.Record) error { return nil }
func (p noopProcessor) Enabled(context.Context, log.Record) bool { return false }
func (p noopProcessor) Shutdown(context.Context) error           { return nil }
func (p noopProcessor) ForceFlush(context.Context) error         { return nil }
