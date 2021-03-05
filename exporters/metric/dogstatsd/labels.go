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

	"go.opentelemetry.io/otel/attribute"
)

// AttributeEncoder encodes metric attributes in the dogstatsd syntax.
//
// TODO: find a link for this syntax.  It's been copied out of code,
// not a specification:
//
// https://github.com/stripe/veneur/blob/master/sinks/datadog/datadog.go
type AttributeEncoder struct {
	pool sync.Pool
}

var _ attribute.Encoder = &AttributeEncoder{}
var leID = attribute.NewEncoderID()

// NewAttributeEncoder returns a new encoder for dogstatsd-syntax metric
// attributes.
func NewAttributeEncoder() *AttributeEncoder {
	return &AttributeEncoder{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

// Encode emits a string like "|#key1:value1,key2:value2".
func (e *AttributeEncoder) Encode(iter attribute.Iterator) string {
	buf := e.pool.Get().(*bytes.Buffer)
	defer e.pool.Put(buf)
	buf.Reset()

	for iter.Next() {
		e.encodeOne(buf, iter.Attribute())
	}
	return buf.String()
}

func (e *AttributeEncoder) encodeOne(buf *bytes.Buffer, kv attribute.KeyValue) {
	if buf.Len() != 0 {
		_, _ = buf.WriteRune(',')
	}
	_, _ = buf.WriteString(string(kv.Key))
	_, _ = buf.WriteRune(':')
	_, _ = buf.WriteString(kv.Value.Emit())
}

func (*AttributeEncoder) ID() attribute.EncoderID {
	return leID
}
