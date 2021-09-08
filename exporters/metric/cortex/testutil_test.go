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

package cortex

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/sdkapi"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/aggregatortest"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

// getSumCheckpoint returns a checkpoint set with a sum aggregation record
func getSumCheckpoint(t *testing.T, values ...int64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	cpSet := newCheckpointSet()
	desc := metric.NewDescriptor("metric_name", sdkapi.CounterInstrumentKind, number.Int64Kind)

	// Create aggregation, add value, and update checkpointset
	sums := sum.New(2)
	agg, ckpt := &sums[0], &sums[1]
	for _, value := range values {
		aggregatortest.CheckedUpdate(t, agg, number.NewInt64Number(value), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	cpSet.add(&desc, ckpt)

	return cpSet
}

// getLastValueCheckpoint returns a checkpoint set with a last value aggregation record
func getLastValueCheckpoint(t *testing.T, values ...int64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	cpSet := newCheckpointSet()
	desc := metric.NewDescriptor("metric_name", sdkapi.GaugeObserverInstrumentKind, number.Int64Kind)

	// Create aggregation, add value, and update checkpointset
	lastvalues := lastvalue.New(2)
	agg, ckpt := &lastvalues[0], &lastvalues[1]
	for _, value := range values {
		aggregatortest.CheckedUpdate(t, agg, number.NewInt64Number(value), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	cpSet.add(&desc, ckpt)

	return cpSet
}

// getMMSCCheckpoint returns a checkpoint set with a minmaxsumcount aggregation record
func getMMSCCheckpoint(t *testing.T, values ...float64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	cpSet := newCheckpointSet()
	desc := metric.NewDescriptor("metric_name", sdkapi.HistogramInstrumentKind, number.Float64Kind)

	// Create aggregation, add value, and update checkpointset
	minmaxsumcounts := minmaxsumcount.New(2, &desc)
	agg, ckpt := &minmaxsumcounts[0], &minmaxsumcounts[1]
	for _, value := range values {
		aggregatortest.CheckedUpdate(t, agg, number.NewFloat64Number(value), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	cpSet.add(&desc, ckpt)

	return cpSet
}

// getHistogramCheckpoint returns a checkpoint set with a histogram aggregation record
func getHistogramCheckpoint(t *testing.T) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	cpSet := newCheckpointSet()
	desc := metric.NewDescriptor("metric_name", sdkapi.HistogramInstrumentKind, number.Float64Kind)

	// Create aggregation, add value, and update checkpointset
	boundaries := []float64{100, 500, 900}
	histograms := histogram.New(2, &desc, histogram.WithExplicitBoundaries(boundaries))
	agg, ckpt := &histograms[0], &histograms[1]
	for i := 0; i < 1000; i++ {
		aggregatortest.CheckedUpdate(t, agg, number.NewFloat64Number(float64(i)+0.5), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	cpSet.add(&desc, ckpt)

	return cpSet
}

// The following variables hold expected TimeSeries values to be used in
// ConvertToTimeSeries tests.
var wantSumCheckpointSet = []*prompb.TimeSeries{
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name",
			},
		},
		Samples: []prompb.Sample{{
			Value:     15,
			Timestamp: mockTime,
		}},
	},
}

var wantLastValueCheckpointSet = []*prompb.TimeSeries{
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name",
			},
		},
		Samples: []prompb.Sample{{
			Value:     5,
			Timestamp: mockTime,
		}},
	},
}

var wantMMSCCheckpointSet = []*prompb.TimeSeries{
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name",
			},
		},
		Samples: []prompb.Sample{{
			Value:     999.999,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name_min",
			},
		},
		Samples: []prompb.Sample{{
			Value:     123.456,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name_max",
			},
		},
		Samples: []prompb.Sample{{
			Value:     876.543,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name_count",
			},
		},
		Samples: []prompb.Sample{{
			Value:     2,
			Timestamp: mockTime,
		}},
	},
}

var wantHistogramCheckpointSet = []*prompb.TimeSeries{
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name_sum",
			},
		},
		Samples: []prompb.Sample{{
			Value:     500000,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name_count",
			},
		},
		Samples: []prompb.Sample{{
			Value:     1000,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name",
			},
			{
				Name:  "le",
				Value: "100",
			},
		},
		Samples: []prompb.Sample{{
			Value:     100,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name",
			},
			{
				Name:  "le",
				Value: "500",
			},
		},
		Samples: []prompb.Sample{{
			Value:     500,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name",
			},
			{
				Name:  "le",
				Value: "900",
			},
		},
		Samples: []prompb.Sample{{
			Value:     900,
			Timestamp: mockTime,
		}},
	},
	{
		Labels: []*prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_name",
			},
			{
				Name:  "le",
				Value: "+inf",
			},
		},
		Samples: []prompb.Sample{{
			Value:     1000,
			Timestamp: mockTime,
		}},
	},
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
