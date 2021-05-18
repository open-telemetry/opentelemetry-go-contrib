// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Based on https://github.com/opentracing-contrib/go-zap
package otelzap

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	otellabel "go.opentelemetry.io/otel/label"
)

const testKey = "testkey"

type stringer struct {
	string
}

func (s stringer) String() string {
	return s.string
}

func TestZapFieldsToOtelKVConversion(t *testing.T) {
	tableTest := []struct {
		zapField zapcore.Field
		otelKV   otellabel.KeyValue
	}{
		{
			zapField: zap.String(testKey, "123"),
			otelKV:   otellabel.String(testKey, "123"),
		},
		{
			zapField: zap.String(testKey, ""),
			otelKV:   otellabel.String(testKey, ""),
		},
		{
			zapField: zap.String(testKey, "123"),
			otelKV:   otellabel.String(testKey, "123"),
		},
		{
			zapField: zap.Stringer(testKey, stringer{""}),
			otelKV:   otellabel.String(testKey, ""),
		},
		{
			zapField: zap.Stringer(testKey, stringer{"123"}),
			otelKV:   otellabel.String(testKey, "123"),
		},
		{
			zapField: zap.Int(testKey, 1),
			otelKV:   otellabel.Int64(testKey, 1),
		},
		{
			zapField: zap.Int32(testKey, 1),
			otelKV:   otellabel.Int32(testKey, 1),
		},
		{
			zapField: zap.Int64(testKey, 1),
			otelKV:   otellabel.Int64(testKey, 1),
		},
		{
			zapField: zap.Uint32(testKey, 1),
			otelKV:   otellabel.Uint32(testKey, 1),
		},
		{
			zapField: zap.Uint64(testKey, 1),
			otelKV:   otellabel.Uint64(testKey, 1),
		},
		{
			zapField: zap.Duration(testKey, time.Second),
			otelKV:   otellabel.String(testKey, "1s"),
		},
		{
			zapField: zap.Float32(testKey, 1),
			otelKV:   otellabel.Float32(testKey, 1),
		},
		{
			zapField: zap.Float64(testKey, 1),
			otelKV:   otellabel.Float64(testKey, 1),
		},
		{
			zapField: zap.Bool(testKey, false),
			otelKV:   otellabel.Bool(testKey, false),
		},
		{
			zapField: zap.Bool(testKey, true),
			otelKV:   otellabel.Bool(testKey, true),
		},
		{
			zapField: zap.Error(errors.New("test error")),
			otelKV:   otellabel.String("error", "test error"),
		},
		{
			zapField: zap.Field{
				Key:  testKey,
				Type: zapcore.UnknownType,
			},
			otelKV: otellabel.String(testKey, "<nil>"),
		},
	}

	for _, test := range tableTest {
		got := zapFieldToOtel(test.zapField)
		if got.Key != test.otelKV.Key {
			t.Errorf("expected same key, got %s but expected %s", got.Key, test.otelKV.Key)
		}
		if got.Value != test.otelKV.Value {
			t.Errorf("expected same value, got %v but expected %v", got.Value, test.otelKV.Value)
		}
	}
}
