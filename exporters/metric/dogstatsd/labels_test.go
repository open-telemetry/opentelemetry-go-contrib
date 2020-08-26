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
	"go.opentelemetry.io/otel/label"
)

var testLabels = []label.KeyValue{
	label.String("A", "B"),
	label.String("C", "D"),
	label.Float64("E", 1.5),
}

func TestLabelSyntax(t *testing.T) {
	encoder := dogstatsd.NewLabelEncoder()

	labels := label.NewSet(testLabels...)
	require.Equal(t, `A:B,C:D,E:1.5`, encoder.Encode(labels.Iter()))

	kvs := []label.KeyValue{
		label.String("A", "B"),
	}
	labels = label.NewSet(kvs...)
	require.Equal(t, `A:B`, encoder.Encode(labels.Iter()))

	labels = label.NewSet()
	require.Equal(t, "", encoder.Encode(labels.Iter()))

	labels = label.Set{}
	require.Equal(t, "", encoder.Encode(labels.Iter()))
}
