// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellogrus provides a [Hook], a [logrus.Hook] implementation that
// can be used to bridge between the [github.com/sirupsen/logrus] API and
// [OpenTelemetry].
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

type Hook struct {
	logger log.Logger
	levels []logrus.Level
}

func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

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
