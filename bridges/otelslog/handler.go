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

// NewLogger returns a new [slog.Logger] backed by a new [Handler]. See
// [NewHandler] for details on how the backing Handler is created.
func NewLogger(options ...Option) *slog.Logger {
	return slog.New(NewHandler(options...))
}

type config struct{}

// Option configures a [Handler].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithInstrumentationScope returns an [Option] that configures the scope of
// the [log.Logger] used by a [Handler].
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

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [Handler] to create its [log.Logger].
//
// By default if this Option is not provided, the Handler will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
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

	logger log.Logger
}

// Compile-time check *Handler implements slog.Handler.
var _ slog.Handler = (*Handler)(nil)

// NewHandler returns a new [Handler] to be used as an [slog.Handler].
//
// If [WithLoggerProvider] is not provided, the returned Handler will use the
// global LoggerProvider.
//
// By default the returned Handler will use a [log.Logger] that is identified
// with this bridge package information. [WithInstrumentationScope] should be
// used to override this with details about the package or module the handler
// will instrument.
func NewHandler(options ...Option) *Handler {
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
func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	var record log.Record
	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	record.SetSeverity(log.Severity(l + sevOffset))
	return h.logger.Enabled(ctx, record)
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
