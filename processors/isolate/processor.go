// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package isolate provides an isolating processor that can be used to
// configure independent processing pipelines.
package isolate // import "go.opentelemetry.io/contrib/processors/isolate"

import (
	"context"

	"go.opentelemetry.io/otel/sdk/log"
)

// NewLogProcessor returns a new [LogProcessor] that wraps the downstream
// [log.Processor].
//
// If downstream is nil a default No-Op [log.Processor] is used. The returned
// processor will not be enabled for nor emit any records.
func NewLogProcessor(downstream log.Processor) *LogProcessor {
	if downstream == nil {
		downstream = defaultProcessor
	}
	return &LogProcessor{Processor: downstream}
}

// LogProcessor is an [log.Processor] implementation clones the received log
// records in order to no share mutable data with subsequent registered processors.
//
// If the wrapped [log.Processor] is nil, calls to the LogProcessor methods
// will panic.
//
// Use [NewLogProcessor] to create a new LogProcessor that ensures
// no panics.
type LogProcessor struct {
	log.Processor
}

// Compile time assertion that LogProcessor implements log.Processor.
var _ log.Processor = (*LogProcessor)(nil)

// OnEmit clones the record and calls the wrapped downstream processor.
func (p *LogProcessor) OnEmit(ctx context.Context, record log.Record) error {
	record = record.Clone()
	return p.Processor.OnEmit(ctx, record)
}

// Enabled clones the record and calls the wrapped downstream processor.
func (p *LogProcessor) Enabled(ctx context.Context, record log.Record) bool {
	record = record.Clone()
	return p.Processor.Enabled(ctx, record)
}

var defaultProcessor = noopProcessor{}

type noopProcessor struct{}

func (p noopProcessor) OnEmit(context.Context, log.Record) error { return nil }
func (p noopProcessor) Enabled(context.Context, log.Record) bool { return false }
func (p noopProcessor) Shutdown(context.Context) error           { return nil }
func (p noopProcessor) ForceFlush(context.Context) error         { return nil }
