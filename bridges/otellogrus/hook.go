// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellogrus provides a [Hook], a [logrus.Hook] implementation that
// can be used to bridge between the [github.com/sirupsen/logrus] API and
// [OpenTelemetry].
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otellogrus // import "go.opentelemetry.io/contrib/bridges/otellogrus"

import (
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

	/*if h.attrs.Len() > 0 {
		record.AddAttributes(h.attrs.KeyValues()...)
	}

	n := r.NumAttrs()
	if h.group != nil {
		if n > 0 {
			buf, free := getKVBuffer()
			defer free()
			r.Attrs(buf.AddAttr)
			record.AddAttributes(h.group.KeyValue(buf.KeyValues()...))
		} else {
			// A Handler should not output groups if there are no attributes.
			g := h.group.NextNonEmpty()
			if g != nil {
				record.AddAttributes(g.KeyValue())
			}
		}
	} else if n > 0 {
		buf, free := getKVBuffer()
		defer free()
		r.Attrs(buf.AddAttr)
		record.AddAttributes(buf.KeyValues()...)
	}*/

	return record
}
