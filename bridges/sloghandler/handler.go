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
	"go.opentelemetry.io/otel/log/noop"
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

// Handler is an [slog.Handler] that sends all logging records it receives to
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
func New(lp log.LoggerProvider, _ ...Option) *Handler {
	if lp == nil {
		// Do not panic.
		lp = noop.NewLoggerProvider()
	}
	return &Handler{
		logger: lp.Logger(
			bridgeName,
			log.WithInstrumentationVersion(bridgeVersion),
		),
	}
}

// Handle handles the passed record.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	h.logger.Emit(ctx, h.convertRecord(record))
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
	h2 := *h
	if h2.group != nil {
		h2.group = h2.group.Clone()
		h2.group.AddAttrs(attrs)
	} else {
		if h2.attrs == nil {
			h2.attrs = newKVBuffer(len(attrs))
		} else {
			h2.attrs = h2.attrs.Clone()
		}
		h2.attrs.AddAttrs(attrs)
	}
	return &h2
}

// WithGroup returns a new [slog.Handler] based on h that will log all messages
// and attributes within a group of the provided name.
func (h *Handler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.group = &group{name: name, next: h2.group}
	return &h2
}

// group represents a group received from slog.
type group struct {
	// name is the name of the group.
	name string
	// attrs are the attributes associated with the group.
	attrs *kvBuffer
	// next points to the next group that holds this group.
	//
	// Groups are represented as map value types in OpenTelemetry. This means
	// that for an slog group hierarchy like the following ...
	//
	//   WithGroup("G").WithGroup("H").WithGroup("I")
	//
	// the corresponding OpenTelemetry log value types will have the following
	// hierarchy ...
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{
	//       Key: "H",
	//       Value: []KeyValue{{
	//         Key: "I",
	//         Value: []KeyValue{},
	//       }},
	//     }},
	//   }
	//
	// When attributes are recorded (i.e. Info("msg", "key", "value") or
	// WithAttrs("key", "value")) they need to be added to the "leaf" group. In
	// the above example, that would be group "I":
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{
	//       Key: "H",
	//       Value: []KeyValue{{
	//         Key: "I",
	//         Value: []KeyValue{
	//           String("key", "value"),
	//         },
	//       }},
	//     }},
	//   }
	//
	// Therefore, groups are structured as a linked-list with the "leaf" node
	// being the head of the list. Following the above example, the group data
	// representation would be ...
	//
	//   *group{"I", next: *group{"H", next: *group{"G"}}}
	next *group
}

// NextNonEmpty returns the next group within g's linked-list that has
// attributes (including g itself). If no group is found, nil is returned.
func (g *group) NextNonEmpty() *group {
	if g == nil || g.attrs.Len() > 0 {
		return g
	}
	return g.next.NextNonEmpty()
}

// KeyValue returns group g containing kvs as a [log.KeyValue]. The value of
// the returned KeyValue will be of type [log.KindMap].
//
// The passed kvs are rendered in the returned value, but are not added to the
// group.
//
// This does not check g. It is the callers responsibility to ensure g is
// non-empty or kvs is non-empty so as to return a valid group representation
// (according to slog).
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

// Clone returns a copy of g.
func (g *group) Clone() *group {
	if g == nil {
		return g
	}
	g2 := *g
	g2.attrs = g2.attrs.Clone()
	return &g2
}

// AddAttrs add attrs to g.
func (g *group) AddAttrs(attrs []slog.Attr) {
	if g.attrs == nil {
		g.attrs = newKVBuffer(len(attrs))
	}
	g.attrs.AddAttrs(attrs)
}

var kvBufferPool = sync.Pool{
	New: func() any {
		// Based on slog research (https://go.dev/blog/slog#performance), 95%
		// of use-cases will use 5 or less attributes.
		return newKVBuffer(5)
	},
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

// Len returns the number of [log.KeyValue] held by b.
func (b *kvBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.data)
}

// Clone returns a copy of b.
func (b *kvBuffer) Clone() *kvBuffer {
	if b == nil {
		return nil
	}
	return &kvBuffer{data: slices.Clone(b.data)}
}

// KeyValues returns kvs appended to the [log.KeyValue] held by b.
func (b *kvBuffer) KeyValues(kvs ...log.KeyValue) []log.KeyValue {
	if b == nil {
		return kvs
	}
	return append(b.data, kvs...)
}

// AddAttrs adds attrs to b.
func (b *kvBuffer) AddAttrs(attrs []slog.Attr) {
	b.data = slices.Grow(b.data, len(attrs))
	for _, a := range attrs {
		_ = b.AddAttr(a)
	}
}

// AddAttrs adds attr to b and returns true.
//
// This is designed to be passed to the AddAttributes method of an
// [slog.Record].
//
// If attr is a group with an empty key, its values will be flattened.
//
// If attr is empty, it will be dropped.
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
		// Try to handle this as gracefully as possible.
		//
		// Don't panic here. The goal here is to have developers find this
		// first if a new slog.Kind is added. A test on the new kind will find
		// this malformed attribute as well as a panic. However, it is
		// preferable to have user's open issue asking why their attributes
		// have a "unhandled: " prefix than say that their code is panicking.
		return log.StringValue(fmt.Sprintf("unhandled: %+v", v.Any()))
	}
}
