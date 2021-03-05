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
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/sdk/export/metric/metrictest"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
	"go.opentelemetry.io/otel/sdk/resource"
)

// TestDogstatsAttributes that attributes are formatted in the correct style,
// including Resources.
func TestDogstatsAttributes(t *testing.T) {
	type testCase struct {
		name       string
		resources  []attribute.KeyValue
		attributes []attribute.KeyValue
		expected   string
	}

	attributes := func(attributes ...attribute.KeyValue) []attribute.KeyValue { return attributes }

	cases := []testCase{
		{
			name:       "no attributes",
			resources:  nil,
			attributes: nil,
			expected:   "test.name:123|c\n",
		},
		{
			name:       "only resources",
			resources:  attributes(attribute.String("R", "S")),
			attributes: nil,
			expected:   "test.name:123|c|#R:S\n",
		},
		{
			name:       "only attributes",
			resources:  nil,
			attributes: attributes(attribute.String("A", "B")),
			expected:   "test.name:123|c|#A:B\n",
		},
		{
			name:       "both resources and attributes",
			resources:  attributes(attribute.String("R", "S")),
			attributes: attributes(attribute.String("A", "B")),
			expected:   "test.name:123|c|#R:S,A:B\n",
		},
		{
			resources:  attributes(attribute.String("A", "R")),
			attributes: attributes(attribute.String("A", "B")),
			expected:   "test.name:123|c|#A:R,A:B\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := resource.NewWithAttributes(tc.resources...)
			ctx := context.Background()
			checkpointSet := metrictest.NewCheckpointSet(res)

			desc := metric.NewDescriptor("test.name", metric.CounterInstrumentKind, number.Int64Kind)
			cagg, cckpt := metrictest.Unslice2(sum.New(2))
			require.NoError(t, cagg.Update(ctx, number.NewInt64Number(123), &desc))
			require.NoError(t, cagg.SynchronizedMove(cckpt, &desc))

			checkpointSet.Add(&desc, cckpt, tc.attributes...)

			var buf bytes.Buffer
			exp, err := dogstatsd.NewRawExporter(dogstatsd.Config{
				Writer: &buf,
			})
			require.Nil(t, err)

			err = exp.Export(ctx, checkpointSet)
			require.Nil(t, err)

			require.Equal(t, tc.expected, buf.String())
		})
	}
}
