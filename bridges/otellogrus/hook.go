// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellogrus provides a [Hook], a [logrus.Hook] implementation that
// can be used to bridge between the [github.com/sirupsen/logrus] API and
// [OpenTelemetry].
//
// # Record Conversion
//
// The [logrus.Entry] records are converted to OpenTelemetry [log.Record] in
// the following way:
//
//   - Time is set as the Timestamp.
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is not
//     set.
//   - Fields are transformed and set as the attributes.
//
// The Level is transformed by using the static offset to the OpenTelemetry
// Severity types. For example:
//
//   - [slog.LevelDebug] is transformed to [log.SeverityDebug]
//   - [slog.LevelInfo] is transformed to [log.SeverityInfo]
//   - [slog.LevelWarn] is transformed to [log.SeverityWarn]
//   - [slog.LevelError] is transformed to [log.SeverityError]
//
// Attribute values are transformed based on type:
//
//   - booleans are transformed into [log.Bool]
//   - byte arrays are transformed into [log.Bytes]
//   - float64 are transformed into [log.Float64]
//   - int are transformed into [log.Int]
//   - int64 are transformed into [log.Int64]
//   - string are transformed into [log.String]
//
// Any other type is transformed into a string.
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otellogrus // import "go.opentelemetry.io/contrib/bridges/otellogrus"

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/log"
)

const bridgeName = "go.opentelemetry.io/contrib/bridges/otellogrus"

// NewHook returns a new [Hook] to be used as a [logrus.Hook].
//
// If [WithLoggerProvider] is not provided, the returned Hook will use the
// global LoggerProvider.
func NewHook(options ...Option) logrus.Hook {
	cfg := newConfig(options)
	return &Hook{
		logger: cfg.logger(),
		levels: cfg.levels,
	}
}

// Hook is a [logrus.Hook] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type Hook struct {
	logger log.Logger
	levels []logrus.Level
}

// Levels returns the list of log levels we want to be sent to OpenTelemetry.
func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

// Fire handles the passed record, and sends it to OpenTelemetry.
func (h *Hook) Fire(entry *logrus.Entry) error {
	ctx := entry.Context
	h.logger.Emit(ctx, h.convertEntry(entry))
	return nil
}

func (h *Hook) convertEntry(e *logrus.Entry) log.Record {
	var record log.Record
	record.SetTimestamp(e.Time)
	record.SetBody(log.StringValue(e.Message))

	const sevOffset = logrus.Level(log.SeverityDebug) - logrus.DebugLevel
	record.SetSeverity(log.Severity(e.Level + sevOffset))
	record.AddAttributes(convertFields(e.Data)...)

	return record
}

func convertFields(fields logrus.Fields) []log.KeyValue {
	kvs := make([]log.KeyValue, len(fields))

	i := 0
	for k, v := range fields {
		kvs[i] = convertKeyValue(k, v)
		i++
	}
	return kvs
}

func convertKeyValue(k string, v interface{}) log.KeyValue {
	switch v := v.(type) {
	case bool:
		return log.Bool(k, v)
	case []byte:
		return log.Bytes(k, v)
	case float64:
		return log.Float64(k, v)
	case int:
		return log.Int(k, v)
	case int64:
		return log.Int64(k, v)
	case string:
		return log.String(k, v)
	default:
		return log.String(k, fmt.Sprintf("%s", v))
	}
}
