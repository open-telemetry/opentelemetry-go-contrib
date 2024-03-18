// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelslog provides [Handler], an [slog.Handler] implementation, that
// can be used to bridge between the [log/slog] API and [OpenTelemetry].
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otelslog // import "go.opentelemetry.io/contrib/bridges/otelslog"

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

type config struct{}

// Option configures a [Handler].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithInstrumentationScope returns an option that configures the scope of the
// [log.Logger] used by a [Handler].
//
// By default if this Option is not provided, the Handler will use a default
// instrumentation scope describing this bridge package. It is recommended to
// provide this so log data can be associated with its source package or
// module.
func WithInstrumentationScope(scope instrumentation.Scope) Option {
	return optFunc(func(c config) config {
		// TODO: implement.
		return c
	})
}

// Handler is an [slog.Handler] that sends all logging records it receives to
// OpenTelemetry.
type Handler struct {
	// Ensure forward compatibility by explicitly making this not comparable.
	noCmp [0]func() //nolint: unused  // This is indeed used.

	// TODO: implement.
}

// Compile-time check *Handler implements slog.Handler.
var _ slog.Handler = (*Handler)(nil)

// New returns a new [Handler] to be used as an [slog.Handler].
//
// If lp is nil, [noop.LoggerProvider] will be used as the default.
//
// By default the returned Handler will use a [log.Logger] that is identified
// with this bridge package information. [WithInstrumentationScope] should be
// used to override this with details about the package or module the handler
// will instrument.
func New(lp log.LoggerProvider, options ...Option) *Handler {
	// TODO: implement.
	return &Handler{}
}

// Handle handles the passed record.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	// TODO: implement.
	return nil
}

// Enable returns true if the Handler is enabled to log for the provided
// context and Level. Otherwise, false is returned if it is not enabled.
func (h *Handler) Enabled(context.Context, slog.Level) bool {
	// TODO: implement.
	return true
}

// WithAttrs returns a new [slog.Handler] based on h that will log using the
// passed attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// TODO: implement.
	return h
}

// WithGroup returns a new [slog.Handler] based on h that will log all messages
// and attributes within a group of the provided name.
func (h *Handler) WithGroup(name string) slog.Handler {
	// TODO: implement.
	return h
}
