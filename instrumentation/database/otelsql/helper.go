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

package otelsql

import (
	"database/sql/driver"
	"fmt"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func paramsAttr(args []driver.Value) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(args))
	for i, arg := range args {
		key := "sql.arg" + strconv.Itoa(i)
		attrs = append(attrs, argToAttr(key, arg))
	}
	return attrs
}

func namedParamsAttr(args []driver.NamedValue) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(args))
	for _, arg := range args {
		var key string
		if arg.Name != "" {
			key = arg.Name
		} else {
			key = "sql.arg." + strconv.Itoa(arg.Ordinal)
		}
		attrs = append(attrs, argToAttr(key, arg.Value))
	}
	return attrs
}

func argToAttr(key string, val interface{}) attribute.KeyValue {
	switch v := val.(type) {
	case nil:
		return attribute.String(key, "")
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []byte:
		if len(v) > 256 {
			v = v[0:256]
		}
		return attribute.String(key, fmt.Sprintf("%s", v))
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > 256 {
			s = s[0:256]
		}
		return attribute.String(key, s)
	}
}

func setSpanStatus(span trace.Span, opts wrapper, err error) {
	switch err {
	case nil:
		span.SetStatus(codes.Ok, "")
		return
	case driver.ErrSkip:
		span.SetStatus(codes.Unset, err.Error())
		if opts.DisableErrSkip {
			// Suppress driver.ErrSkip since at runtime some drivers might not have
			// certain features, and an error would pollute many spans.
			span.SetStatus(codes.Ok, err.Error())
		}
	default:
		span.SetStatus(codes.Error, err.Error())
	}
}
