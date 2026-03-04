// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func BenchmarkCoreWrite(b *testing.B) {
	benchmarks := []struct {
		name   string
		fields []zapcore.Field
	}{
		{
			name: "10 fields",
			fields: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.ByteString("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.Array("k", loggable{true}),
				zap.String("k", "a"),
				zap.Ints("k", []int{1, 2}),
			},
		},
		{
			name: "20 fields",
			fields: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.ByteString("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.String("k", "a"),
				zap.Array("k", loggable{true}),
				zap.Ints("k", []int{1, 2}),
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.ByteString("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.Array("k", loggable{true}),
				zap.String("k", "a"),
				zap.Ints("k", []int{1, 2}),
			},
		},
		{ // Benchmark with nested namespace
			name: "Namespace",
			fields: []zapcore.Field{
				zap.Namespace("a"),
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.Namespace("b"),
				zap.Binary("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.String("k", "a"),
				zap.Array("k", loggable{true}),
				zap.Ints("k", []int{1, 2}),
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			zc := NewCore(loggerName)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc.Write(testEntry, bm.fields)
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
				}
			})
		})
	}

	for _, bm := range benchmarks {
		b.Run(fmt.Sprint("With", bm.name), func(b *testing.B) {
			zc := NewCore(loggerName)
			zc1 := zc.With(bm.fields)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc1.Write(testEntry, []zapcore.Field{})
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
				}
			})
		})
	}
}
