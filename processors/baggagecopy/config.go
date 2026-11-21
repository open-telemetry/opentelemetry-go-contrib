// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagecopy

// config is the configuration of a [SpanProcessor] or a [LogProcessor].
type config struct {
	// keepExisting is the negation of OverwriteExistingAttributes so that the
	// zero config overwrites, as both processors always have.
	keepExisting bool
}

func newConfig(opts []Option) config {
	var c config
	for _, opt := range opts {
		c = opt.apply(c)
	}
	return c
}

// Option configures a [SpanProcessor] or a [LogProcessor].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithOverwriteExistingAttributes returns an [Option] that configures whether a
// baggage member replaces an attribute of the same key already set on the span
// it is copied onto, or on the log record.
//
// By default, if this Option is not provided, a baggage member overwrites the
// existing attribute. Pass false to keep the existing attribute and drop the
// baggage member instead. Baggage is commonly extracted from an incoming
// request, so overwriting lets the caller rewrite attributes set by the
// instrumentation or by the application itself.
func WithOverwriteExistingAttributes(overwrite bool) Option {
	return optFunc(func(c config) config {
		c.keepExisting = !overwrite
		return c
	})
}
