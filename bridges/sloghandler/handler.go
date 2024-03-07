// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package sloghandler provides a bridge between the [log/slog] and
// OpenTelemetry logging.
package sloghandler // import "go.opentelemetry.io/contrib/bridges/sloghandler"

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"go.opentelemetry.io/otel/log"
)

const (
	bridgeName = "go.opentelemetry.io/contrib/bridge/sloghandler"
	// TODO: hook this into the release pipeline.
	bridgeVersion = "0.0.1-alpha"
)

type config struct{}

// Option configures a [Handler].
type Option interface {
	apply(config) config
}

// Handler is a [slog.Handler] that sends all logging records it receives to
// OpenTelemetry.
type Handler struct {
	// Ensure forward compatibility by explicitly making this not comparable.
	noCmp [0]func() //nolint: unused  // This is indeed used.

	attrs  []log.KeyValue
	group  *group
	logger log.Logger
}

// Compile-time check *Handler implements slog.Handler.
var _ slog.Handler = (*Handler)(nil)

// New returns a new [Handler] to be used as an [slog.Handler].
func New(lp log.LoggerProvider, opts ...Option) *Handler {
	return &Handler{
		logger: lp.Logger(
			bridgeName,
			log.WithInstrumentationVersion(bridgeVersion),
		),
	}
}

// Handle handles the Record.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var record log.Record
	record.SetTimestamp(r.Time)
	record.SetBody(log.StringValue(r.Message))

	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	record.SetSeverity(log.Severity(r.Level + sevOffset))

	record.AddAttributes(h.attrs...)
	if h.group != nil {
		n := r.NumAttrs()
		if n > 0 {
			h.group.grow(n)
			r.Attrs(h.group.addAttr)
		}

		curr := h.group
		for curr != nil && len(curr.attrs) == 0 {
			curr = curr.prev
		}
		if curr != nil {
			record.AddAttributes(curr.convert())
		}
	} else {
		r.Attrs(func(attr slog.Attr) bool {
			if attr.Key == "" {
				if attr.Value.Kind() == slog.KindGroup {
					// A Handler should inline the Attrs of a group with an empty key.
					g := attr.Value.Group()
					for _, a := range g {
						record.AddAttributes(convertAttr(a))
					}
					return true
				}

				if attr.Value.Any() == nil {
					// A Handler should ignore an empty Attr.
					return true
				}
			}
			record.AddAttributes(convertAttr(attr))
			return true
		})
	}

	h.logger.Emit(ctx, record)
	return nil
}

// Enable returns true if the Handler is enabled to log for the provided
// context and Level. Otherwise, false is returned if it is not enabled.
func (h *Handler) Enabled(context.Context, slog.Level) bool {
	// TODO (MrAlias): The Logs Bridge API does not provide a way to retrieve
	// the current minimum logging level yet.
	// https://github.com/open-telemetry/opentelemetry-go/issues/4995
	return true
}

// WithAttrs returns a new [slog.Handler] based on h that will log using the
// passed attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	h2 := *h
	if h2.group != nil {
		g := *h2.group

		// Do not alter the orig.
		g.attrs = slices.Clip(g.attrs)
		g.attrs = slices.Grow(g.attrs, len(attrs))
		for _, a := range attrs {
			syncAttr(func(kv ...log.KeyValue) {
				g.attrs = append(g.attrs, kv...)
			}, a)
		}

		h2.group = &g
	} else {
		// Force an append to copy the underlying array.
		h2.attrs = slices.Clip(h2.attrs)
		h2.attrs = slices.Grow(h2.attrs, len(attrs))
		for _, a := range attrs {
			syncAttr(func(kv ...log.KeyValue) {
				h2.attrs = append(h2.attrs, kv...)
			}, a)
		}
	}
	return &h2
}

// WithGroup returns a new [slog.Handler] based on h that will log all messages
// and attributes within a group using name.
func (h *Handler) WithGroup(name string) slog.Handler {
	// Handlers should inline the Attrs of a group with an empty key.
	if name == "" {
		return h
	}

	h2 := *h
	h2.group = &group{name: name, prev: h2.group}
	return &h2
}

func syncAttr(sync func(...log.KeyValue), attr slog.Attr) {
	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			// A Handler should inline the Attrs of a group with an empty key.
			g := attr.Value.Group()
			for _, a := range g {
				sync(convertAttr(a))
			}
			return
		}

		if attr.Value.Any() == nil {
			// A Handler should ignore an empty Attr.
			return
		}
	}

	sync(convertAttr(attr))
}

func convertAttr(attr slog.Attr) log.KeyValue {
	val := convertValue(attr.Value)
	return log.KeyValue{Key: attr.Key, Value: val}
}

func convertValue(v slog.Value) log.Value {
	switch v.Kind() {
	case slog.KindAny:
		return log.StringValue(fmt.Sprintf("%+v", v.Any()))
	case slog.KindBool:
		return log.BoolValue(v.Bool())
	case slog.KindDuration:
		return log.Int64Value(v.Duration().Nanoseconds())
	case slog.KindFloat64:
		return log.Float64Value(v.Float64())
	case slog.KindInt64:
		return log.Int64Value(v.Int64())
	case slog.KindString:
		return log.StringValue(v.String())
	case slog.KindTime:
		return log.Int64Value(v.Time().UnixNano())
	case slog.KindUint64:
		return log.Int64Value(int64(v.Uint64()))
	case slog.KindGroup:
		g := v.Group()
		kvs := make([]log.KeyValue, 0, len(g))
		for _, a := range g {
			syncAttr(func(kv ...log.KeyValue) {
				kvs = append(kvs, kv...)
			}, a)
		}
		return log.MapValue(kvs...)
	case slog.KindLogValuer:
		return convertValue(v.Resolve())
	default:
		panic(fmt.Sprintf("unhandled attribute kind: %s", v.Kind()))
	}
}

type group struct {
	name  string
	attrs []log.KeyValue
	prev  *group
}

func (g *group) grow(n int) {
	g.attrs = slices.Clip(g.attrs)
	g.attrs = slices.Grow(g.attrs, n)
}

func (g *group) addAttr(attr slog.Attr) bool {
	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			// A Handler should inline the Attrs of a group with an empty key.
			for _, a := range attr.Value.Group() {
				g.attrs = append(g.attrs, convertAttr(a))
			}
			return true
		}

		if attr.Value.Any() == nil {
			// A Handler should ignore an empty Attr.
			return true
		}
	}
	g.attrs = append(g.attrs, convertAttr(attr))
	return true
}

func (g *group) convert() log.KeyValue {
	out := log.Map(g.name, g.attrs...)
	g = g.prev
	for g != nil {
		// A Handler should not output groups if there are no attributes.
		if len(g.attrs) > 0 {
			if out.Value.Kind() != log.KindEmpty {
				out = log.Map(g.name, append(g.attrs, out)...)
			} else {
				out = log.Map(g.name, g.attrs...)
			}
		}
		g = g.prev
	}
	return out
}
