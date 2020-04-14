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

package statsd_test

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-go-contrib/exporters/metric/dogstatsd/internal/statsd"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	export "go.opentelemetry.io/otel/sdk/export/metric"
)

var testLabels = []core.KeyValue{
	key.New("A").String("B"),
	key.New("C").String("D"),
	key.New("E").Float64(1.5),
}

func TestLabelSyntax(t *testing.T) {
	encoder := statsd.NewLabelEncoder()

	require.Equal(t, `|#A:B,C:D,E:1.5`, encoder.Encode(export.LabelSlice(testLabels).Iter()))

	kvs := []core.KeyValue{
		key.New("A").String("B"),
	}
	require.Equal(t, `|#A:B`, encoder.Encode(export.LabelSlice(kvs).Iter()))

	require.Equal(t, "", encoder.Encode(export.LabelSlice(nil).Iter()))
}
