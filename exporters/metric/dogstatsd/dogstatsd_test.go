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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/metric/sdkapi"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
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
			ctx := context.Background()
			cpSet := newCheckpointSet()

			desc := metric.NewDescriptor("test.name", sdkapi.CounterInstrumentKind, number.Int64Kind)
			sums := sum.New(2)
			cagg, cckpt := &sums[0], &sums[1]
			require.NoError(t, cagg.Update(ctx, number.NewInt64Number(123), &desc))
			require.NoError(t, cagg.SynchronizedMove(cckpt, &desc))

			cpSet.add(&desc, cckpt, tc.attributes...)

			var buf bytes.Buffer
			exp, err := dogstatsd.NewRawExporter(dogstatsd.Config{
				Writer: &buf,
			})
			require.Nil(t, err)

			err = exp.Export(ctx, resource.NewWithAttributes(semconv.SchemaURL, tc.resources...), cpSet)
			require.Nil(t, err)

			require.Equal(t, tc.expected, buf.String())
		})
	}
}

type mapkey struct {
	desc     *metric.Descriptor
	distinct attribute.Distinct
}

type checkpointSet struct {
	// RWMutex implements locking for the `CheckpointSet` interface.
	sync.RWMutex
	records map[mapkey]export.Record
	updates []export.Record
}

func newCheckpointSet() *checkpointSet {
	return &checkpointSet{
		records: make(map[mapkey]export.Record),
	}
}

// Add a new record to a CheckpointSet.
func (p *checkpointSet) add(desc *metric.Descriptor, newAgg export.Aggregator, labels ...attribute.KeyValue) (agg export.Aggregator, added bool) {
	elabels := attribute.NewSet(labels...)

	key := mapkey{
		desc:     desc,
		distinct: elabels.Equivalent(),
	}
	if record, ok := p.records[key]; ok {
		return record.Aggregation().(export.Aggregator), false
	}

	rec := export.NewRecord(desc, &elabels, newAgg.Aggregation(), time.Time{}, time.Time{})
	p.updates = append(p.updates, rec)
	p.records[key] = rec
	return newAgg, true
}

func (p *checkpointSet) ForEach(_ export.ExportKindSelector, f func(export.Record) error) error {
	for _, r := range p.updates {
		if err := f(r); err != nil && !errors.Is(err, aggregation.ErrNoData) {
			return err
		}
	}
	return nil
}
