// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagecopy

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/log"
)

// LogProcessor is a [log.Processor] implementation that adds baggage
// members onto a log as attributes.
type LogProcessor struct {
	filter Filter
	config config
}

var _ log.Processor = (*LogProcessor)(nil)

// NewLogProcessor returns a new [LogProcessor].
//
// The Baggage log processor adds attributes to a log record that are found
// in Baggage in the parent context at the moment the log is emitted.
// The passed filter determines which baggage members are added to the span.
//
// If filter is nil, all baggage members will be added.
//
// A baggage member overwrites an attribute of the same key already set on the
// log record unless [WithOverwriteExistingAttributes] is passed false.
func NewLogProcessor(filter Filter, opts ...Option) *LogProcessor {
	return &LogProcessor{
		filter: filter,
		config: newConfig(opts),
	}
}

// Enabled reports whether the Processor will process.
func (LogProcessor) Enabled(context.Context, log.EnabledParameters) bool {
	return true
}

// OnEmit adds Baggage member to a log record as attributes that are pulled from
// the Baggage found in ctx. Baggage members are filtered by the filter passed
// to NewLogProcessor.
func (processor LogProcessor) OnEmit(ctx context.Context, record *log.Record) error {
	filter := processor.filter
	if filter == nil {
		filter = AllowAllMembers
	}

	for _, member := range baggage.FromContext(ctx).Members() {
		if !filter(member) {
			continue
		}
		if processor.config.keepExisting && hasAttribute(record, attribute.Key(member.Key())) {
			continue
		}
		record.AddAttributes(attribute.String(member.Key(), member.Value()))
	}

	return nil
}

func hasAttribute(record *log.Record, key attribute.Key) bool {
	var found bool
	record.WalkAttributes(func(kv attribute.KeyValue) bool {
		found = kv.Key == key
		return !found
	})
	return found
}

// Shutdown is called when the [log.Processor] is shutting down and is a no-op for this processor.
func (LogProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush is called to ensure all logs are flushed to the output and is a no-op for this processor.
func (LogProcessor) ForceFlush(context.Context) error { return nil }
