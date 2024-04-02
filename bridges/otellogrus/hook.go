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

func (h *Hook) Fire(*logrus.Entry) error {
	return nil
}
