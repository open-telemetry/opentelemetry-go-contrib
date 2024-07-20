// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgokit

import (
	"context"
	"testing"
	"time"

	gokitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
)

var (
	loggerName = "name"
	testKey    = "testKey"
	testValue  = "testValue"
	testTime   = time.Now()
)

type contextKey string

const (
	userKey contextKey = "user"
)

func TestLogger(t *testing.T) {
	// set context
	ctx := context.Background()
	ctx = context.WithValue(ctx, userKey, true)

	rec := logtest.NewRecorder()
	logger := NewOTelLogger(loggerName, WithLoggerProvider(rec))

	t.Run("Log", func(t *testing.T) {
		logger.Log(testKey, testValue, "ctx", ctx)
		got := rec.Result()[0].Records[0]
		assert.Equal(t, 1, got.AttributesLen())
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testKey, kv.Key)
			assert.Equal(t, testValue, value2Result(kv.Value))
			return true
		})
		assert.Equal(t, got.Context(), ctx)
	})

	rec.Reset()
	t.Run("Log With", func(t *testing.T) {
		childlogger := gokitlog.With(logger, "ts", testTime)
		childlogger.Log(testKey, testValue, "ctx", ctx)

		got := rec.Result()[0].Records[0]
		assert.Equal(t, testTime, got.Timestamp())
		assert.Equal(t, 1, got.AttributesLen())
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testKey, kv.Key)
			assert.Equal(t, testValue, value2Result(kv.Value))
			return true
		})
		assert.Equal(t, got.Context(), ctx)
	})
}

func value2Result(v log.Value) any {
	switch v.Kind() {
	case log.KindBool:
		return v.AsBool()
	case log.KindFloat64:
		return v.AsFloat64()
	case log.KindInt64:
		return v.AsInt64()
	case log.KindString:
		return v.AsString()
	case log.KindBytes:
		return v.AsBytes()
	case log.KindSlice:
		var s []any
		for _, val := range v.AsSlice() {
			s = append(s, value2Result(val))
		}
		return s
	case log.KindMap:
		m := make(map[string]any)
		for _, val := range v.AsMap() {
			m[val.Key] = value2Result(val.Value)
		}
		return m
	}
	return nil
}
