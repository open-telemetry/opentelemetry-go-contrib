// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"
)

func BenchmarkLogSink(b *testing.B) {
	message := "body"
	keyValues := []any{
		"string", "hello",
		"int", 42,
		"float", 3.14,
		"bool", false,
	}
	err := errors.New("error")

	b.Run("Info", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Info(0, message, keyValues...)
		}
	})

	b.Run("Error", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Error(err, message, keyValues...)
		}
	})

	b.Run("WithValues", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].WithValues(keyValues...)
		}
	})

	b.Run("WithName", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].WithName("name")
		}
	})

	b.Run("WithName.WithValues", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].WithName("name").WithValues(keyValues...)
		}
	})

	b.Run("(WithName.WithValues).Info", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("").WithName("name").WithValues(keyValues...)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Info(0, message)
		}
	})
}
