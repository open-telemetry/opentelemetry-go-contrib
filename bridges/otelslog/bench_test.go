// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog

import (
	"log/slog"
	"testing"
	"time"
)

func BenchmarkHandler(b *testing.B) {
	var (
		h   slog.Handler
		err error
	)

	attrs10 := []slog.Attr{
		slog.String("1", "1"),
		slog.Int64("2", 2),
		slog.Int("3", 3),
		slog.Uint64("4", 4),
		slog.Float64("5", 5.),
		slog.Bool("6", true),
		slog.Time("7", time.Now()),
		slog.Duration("8", time.Second),
		slog.Any("9", 9),
		slog.Any("10", "10"),
	}
	attrs5 := attrs10[:5]
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "body", 0)
	ctx := b.Context()

	b.Run("Handle", func(b *testing.B) {
		handlers := make([]*Handler, b.N)
		for i := range handlers {
			handlers[i] = NewHandler("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			err = handlers[n].Handle(ctx, record)
		}
	})

	b.Run("WithAttrs", func(b *testing.B) {
		b.Run("5", func(b *testing.B) {
			handlers := make([]*Handler, b.N)
			for i := range handlers {
				handlers[i] = NewHandler("")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for n := range b.N {
				h = handlers[n].WithAttrs(attrs5)
			}
		})
		b.Run("10", func(b *testing.B) {
			handlers := make([]*Handler, b.N)
			for i := range handlers {
				handlers[i] = NewHandler("")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for n := range b.N {
				h = handlers[n].WithAttrs(attrs10)
			}
		})
	})

	b.Run("WithGroup", func(b *testing.B) {
		handlers := make([]*Handler, b.N)
		for i := range handlers {
			handlers[i] = NewHandler("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			h = handlers[n].WithGroup("group")
		}
	})

	b.Run("WithGroup.WithAttrs", func(b *testing.B) {
		b.Run("5", func(b *testing.B) {
			handlers := make([]*Handler, b.N)
			for i := range handlers {
				handlers[i] = NewHandler("")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for n := range b.N {
				h = handlers[n].WithGroup("group").WithAttrs(attrs5)
			}
		})
		b.Run("10", func(b *testing.B) {
			handlers := make([]*Handler, b.N)
			for i := range handlers {
				handlers[i] = NewHandler("")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for n := range b.N {
				h = handlers[n].WithGroup("group").WithAttrs(attrs10)
			}
		})
	})

	b.Run("(WithGroup.WithAttrs).Handle", func(b *testing.B) {
		b.Run("5", func(b *testing.B) {
			handlers := make([]slog.Handler, b.N)
			for i := range handlers {
				handlers[i] = NewHandler("").WithGroup("group").WithAttrs(attrs5)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for n := range b.N {
				err = handlers[n].Handle(ctx, record)
			}
		})
		b.Run("10", func(b *testing.B) {
			handlers := make([]slog.Handler, b.N)
			for i := range handlers {
				handlers[i] = NewHandler("").WithGroup("group").WithAttrs(attrs10)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for n := range b.N {
				err = handlers[n].Handle(ctx, record)
			}
		})
	})

	b.Run("(WithSource).Handle", func(b *testing.B) {
		handlers := make([]*Handler, b.N)
		for i := range handlers {
			handlers[i] = NewHandler("", WithSource(true))
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			err = handlers[n].Handle(ctx, record)
		}
	})

	_, _ = h, err
}
