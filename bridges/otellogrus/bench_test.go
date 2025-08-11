// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogrus

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func BenchmarkHook(b *testing.B) {
	record := &logrus.Entry{
		Data: map[string]any{
			"string": "hello",
			"int":    42,
			"float":  1.5,
			"bool":   false,
		},
		Message: "body",
		Time:    time.Now(),
		Level:   logrus.InfoLevel,
	}

	b.Run("Fire", func(b *testing.B) {
		hooks := make([]*Hook, b.N)
		for i := range hooks {
			hooks[i] = NewHook("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			_ = hooks[n].Fire(record)
		}
	})
}
