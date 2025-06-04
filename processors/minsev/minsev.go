// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package minsev provides an [log.Processor] that will not log any record with
// a severity below a configured threshold.
package minsev // import "go.opentelemetry.io/contrib/processors/minsev"

import (
	"context"

	logapi "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

// NewLogProcessor returns a new [LogProcessor] that wraps the downstream
// [log.Processor].
//
// Severitier reports the minimum record severity that will be logged. The
// LogProcessor discards records with lower severities. If severity is nil,
// SeverityInfo is used as a default. The LogProcessor calls severitier.Severity
// for each record processed or queried; to adjust the minimum level
// dynamically, use a [SeverityVar].
//
// If downstream is nil a default No-Op [log.Processor] is used. The returned
// processor will not be enabled for nor emit any records.
func NewLogProcessor(downstream log.Processor, severitier Severitier) *LogProcessor {
	if downstream == nil {
		downstream = defaultProcessor
	}
	if severitier == nil {
		severitier = SeverityInfo
	}
	p := &LogProcessor{Processor: downstream, sev: severitier}
	if fp, ok := downstream.(log.FilterProcessor); ok {
		p.filter = fp
	}
	return p
}

// LogProcessor is an [log.Processor] implementation that wraps another
// [log.Processor]. It filters out log records with severity below a minimum
// severity level, which is provided by a [Severitier] interface, that are
// within the [logapi.SeverityTrace1]..[logapi.SeverityFatal4] range.
//
// If the wrapped [log.Processor] is nil, calls to the LogProcessor methods
// will panic. Use [NewLogProcessor] to create a new LogProcessor that ensures
// no panics.
type LogProcessor struct {
	log.Processor

	filter log.FilterProcessor
	sev    Severitier
}

// Compile time assertion that LogProcessor implements log.Processor and log.FilterProcessor.
var (
	_ log.Processor       = (*LogProcessor)(nil)
	_ log.FilterProcessor = (*LogProcessor)(nil)
)

// OnEmit drops records with severity less than the one returned by [Severitier]
// and inside the [logapi.SeverityTrace1]..[logapi.SeverityFatal4] range.
// If the severity of record is greater than or equal to he one returned by [Severitier],
// it calls the wrapped [log.Processor] with ctx and record.
func (p *LogProcessor) OnEmit(ctx context.Context, record *log.Record) error {
	sev := record.Severity()
	if sev >= logapi.SeverityTrace1 && sev <= logapi.SeverityFatal4 && sev < p.sev.Severity() {
		return nil
	}
	return p.Processor.OnEmit(ctx, record)
}

// Enabled returns false if the severity of param is inside the
// [logapi.SeverityTrace1]..[logapi.SeverityFatal4] range and less than
// the one returned by [Severitier].
// Otherwise, it returns the result of calling Enabled on the wrapped
// [log.Processor] if it implements [log.FilterProcessor].
// If the wrapped [log.Processor] does not implement [log.FilterProcessor], it returns true.
func (p *LogProcessor) Enabled(ctx context.Context, param log.EnabledParameters) bool {
	sev := param.Severity
	if sev >= logapi.SeverityTrace1 && sev <= logapi.SeverityFatal4 && sev < p.sev.Severity() {
		return false
	}
	if p.filter != nil {
		return p.filter.Enabled(ctx, param)
	}
	return true
}

var defaultProcessor = noopProcessor{}

type noopProcessor struct{}

func (p noopProcessor) OnEmit(context.Context, *log.Record) error           { return nil }
func (p noopProcessor) Enabled(context.Context, log.EnabledParameters) bool { return false }
func (p noopProcessor) Shutdown(context.Context) error                      { return nil }
func (p noopProcessor) ForceFlush(context.Context) error                    { return nil }
