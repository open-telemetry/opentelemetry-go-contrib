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
	"fmt"
	"math"
	"time"

	otellabel "go.opentelemetry.io/otel/label"

	"go.uber.org/zap/zapcore"
)

func zapFieldsToOtel(zapFields ...zapcore.Field) []otellabel.KeyValue {
	otelKV := make([]otellabel.KeyValue, len(zapFields))
	for i, zapField := range zapFields {
		otelKV[i] = zapFieldToOtel(zapField)
	}

	return otelKV
}

func zapFieldToOtel(zapField zapcore.Field) otellabel.KeyValue {
	switch zapField.Type {
	case zapcore.BoolType:
		return otellabel.Bool(zapField.Key, zapField.Integer >= 1)
	case zapcore.Float32Type:
		return otellabel.Float32(zapField.Key, math.Float32frombits(uint32(zapField.Integer)))
	case zapcore.Float64Type:
		return otellabel.Float64(zapField.Key, math.Float64frombits(uint64(zapField.Integer)))
	case zapcore.Int64Type:
		return otellabel.Int64(zapField.Key, zapField.Integer)
	case zapcore.Int32Type:
		return otellabel.Int32(zapField.Key, int32(zapField.Integer))
	case zapcore.StringType:
		return otellabel.String(zapField.Key, zapField.String)
	case zapcore.StringerType:
		return otellabel.String(zapField.Key, zapField.Interface.(fmt.Stringer).String())
	case zapcore.Uint64Type:
		return otellabel.Uint64(zapField.Key, uint64(zapField.Integer))
	case zapcore.Uint32Type:
		return otellabel.Uint32(zapField.Key, uint32(zapField.Integer))
	case zapcore.DurationType:
		return otellabel.String(zapField.Key, time.Duration(zapField.Integer).String())
	case zapcore.ErrorType:
		return otellabel.String(zapField.Key, zapField.Interface.(error).Error())
	default:
		return otellabel.String(zapField.Key, fmt.Sprintf("%+v", zapField.Interface))
	}
}
