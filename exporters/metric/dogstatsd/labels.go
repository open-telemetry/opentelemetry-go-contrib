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

package dogstatsd

import (
	"bytes"
	"sync"

	"go.opentelemetry.io/otel/api/core"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// LabelEncoder encodes metric labels in the dogstatsd syntax.
//
// TODO: find a link for this syntax.  It's been copied out of code,
// not a specification:
//
// https://github.com/stripe/veneur/blob/master/sinks/datadog/datadog.go
type LabelEncoder struct {
	pool      sync.Pool
	resources []core.KeyValue
}

var _ export.LabelEncoder = &LabelEncoder{}
var leID = export.NewLabelEncoderID()

// NewLabelEncoder returns a new encoder for dogstatsd-syntax metric
// labels.
func NewLabelEncoder(resource *resource.Resource) *LabelEncoder {
	return &LabelEncoder{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
		resources: resource.Attributes(),
	}
}

// Encode emits a string like "|#key1:value1,key2:value2".
func (e *LabelEncoder) Encode(iter export.LabelIterator) string {
	buf := e.pool.Get().(*bytes.Buffer)
	defer e.pool.Put(buf)
	buf.Reset()

	for _, kv := range e.resources {
		e.encodeOne(buf, kv)
	}
	for iter.Next() {
		e.encodeOne(buf, iter.Label())
	}
	return buf.String()
}

func (e *LabelEncoder) encodeOne(buf *bytes.Buffer, kv core.KeyValue) {
	if buf.Len() == 0 {
		_, _ = buf.WriteString("|#")
	} else {
		_, _ = buf.WriteRune(',')
	}
	_, _ = buf.WriteString(string(kv.Key))
	_, _ = buf.WriteRune(':')
	_, _ = buf.WriteString(kv.Value.Emit())
}

func (*LabelEncoder) ID() int64 {
	return leID
}
