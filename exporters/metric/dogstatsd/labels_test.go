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

package dogstatsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd"
	"go.opentelemetry.io/otel/attribute"
)

var testAttributes = []attribute.KeyValue{
	attribute.String("A", "B"),
	attribute.String("C", "D"),
	attribute.Float64("E", 1.5),
}

func TestAttributeSyntax(t *testing.T) {
	encoder := dogstatsd.NewAttributeEncoder()

	attributes := attribute.NewSet(testAttributes...)
	require.Equal(t, `A:B,C:D,E:1.5`, encoder.Encode(attributes.Iter()))

	kvs := []attribute.KeyValue{
		attribute.String("A", "B"),
	}
	attributes = attribute.NewSet(kvs...)
	require.Equal(t, `A:B`, encoder.Encode(attributes.Iter()))

	attributes = attribute.NewSet()
	require.Equal(t, "", encoder.Encode(attributes.Iter()))

	attributes = attribute.Set{}
	require.Equal(t, "", encoder.Encode(attributes.Iter()))
}
