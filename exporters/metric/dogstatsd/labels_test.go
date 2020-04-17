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
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var testLabels = []core.KeyValue{
	key.String("A", "B"),
	key.String("C", "D"),
	key.Float64("E", 1.5),
}

var testResources = []core.KeyValue{
	key.String("R1", "V1"),
	key.String("R2", "V2"),
}

func TestLabelSyntax(t *testing.T) {
	encoder := dogstatsd.NewLabelEncoder(resource.New())

	require.Equal(t, `|#A:B,C:D,E:1.5`, encoder.Encode(export.LabelSlice(testLabels).Iter()))

	kvs := []core.KeyValue{
		key.String("A", "B"),
	}
	require.Equal(t, `|#A:B`, encoder.Encode(export.LabelSlice(kvs).Iter()))

	require.Equal(t, "", encoder.Encode(export.LabelSlice(nil).Iter()))
}

func TestLabelResources(t *testing.T) {
	encoder := dogstatsd.NewLabelEncoder(resource.New(testResources...))

	require.Equal(t, `|#R1:V1,R2:V2,A:B,C:D,E:1.5`, encoder.Encode(export.LabelSlice(testLabels).Iter()))
}
