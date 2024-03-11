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
	"sync"

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

	attrs  *kvBuffer
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
	h.logger.Emit(ctx, h.convertRecord(r))
	return nil
}

func (h *Handler) convertRecord(r slog.Record) log.Record {
	var record log.Record
	record.SetTimestamp(r.Time)
	record.SetBody(log.StringValue(r.Message))

	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	record.SetSeverity(log.Severity(r.Level + sevOffset))

	if h.attrs.Len() > 0 {
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
	}

	return record
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
		h2.group.AddAttrs(attrs)
	} else {
		if h2.attrs == nil {
			h2.attrs = newKVBuffer(len(attrs))
		}
		h2.attrs.AddAttrs(attrs)
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
	h2.group = &group{name: name, next: h2.group}
	return &h2
}

type group struct {
	name  string
	attrs *kvBuffer
	next  *group
}

func (g *group) NextNonEmpty() *group {
	if g == nil || g.attrs.Len() > 0 {
		return g
	}
	return g.next.NextNonEmpty()
}

func (g *group) KeyValue(kvs ...log.KeyValue) log.KeyValue {
	// Assumes checking of group g already performed (i.e. non-empty).
	out := log.Map(g.name, g.attrs.KeyValues(kvs...)...)
	g = g.next
	for g != nil {
		// A Handler should not output groups if there are no attributes.
		if g.attrs.Len() > 0 {
			out = log.Map(g.name, g.attrs.KeyValues(out)...)
		}
		g = g.next
	}
	return out
}

func (g *group) AddAttrs(attrs []slog.Attr) {
	if g.attrs == nil {
		g.attrs = newKVBuffer(len(attrs))
	}
	g.attrs.AddAttrs(attrs)
}

var kvBufferPool = sync.Pool{
	New: func() any { return newKVBuffer(10) },
}

func getKVBuffer() (buf *kvBuffer, free func()) {
	buf = kvBufferPool.Get().(*kvBuffer)
	return buf, func() {
		// TODO: limit returned size so the pool doesn't hold on to very large
		// buffers.

		// Do not modify any previously held data.
		buf.data = buf.data[:0:0]
		kvBufferPool.Put(buf)
	}
}

type kvBuffer struct {
	data []log.KeyValue
}

func newKVBuffer(n int) *kvBuffer {
	return &kvBuffer{data: make([]log.KeyValue, 0, n)}
}

func (b *kvBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.data)
}

func (b *kvBuffer) KeyValues(kvs ...log.KeyValue) []log.KeyValue {
	if b == nil {
		return kvs
	}
	return append(b.data, kvs...)
}

func (b *kvBuffer) AddAttrs(attrs []slog.Attr) {
	b.data = slices.Grow(b.data, len(attrs))
	for _, a := range attrs {
		_ = b.AddAttr(a)
	}
}

func (b *kvBuffer) AddAttr(attr slog.Attr) bool {
	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			// A Handler should inline the Attrs of a group with an empty key.
			for _, a := range attr.Value.Group() {
				b.data = append(b.data, log.KeyValue{
					Key:   a.Key,
					Value: convertValue(a.Value),
				})
			}
			return true
		}

		if attr.Value.Any() == nil {
			// A Handler should ignore an empty Attr.
			return true
		}
	}
	b.data = append(b.data, log.KeyValue{
		Key:   attr.Key,
		Value: convertValue(attr.Value),
	})
	return true
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
		buf, free := getKVBuffer()
		defer free()
		buf.AddAttrs(v.Group())
		return log.MapValue(buf.data...)
	case slog.KindLogValuer:
		return convertValue(v.Resolve())
	default:
		panic(fmt.Sprintf("unhandled attribute kind: %s", v.Kind()))
	}
}
