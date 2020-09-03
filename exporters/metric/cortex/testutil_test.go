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
	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/metrictest"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/aggregatortest"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/array"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

// getSumCheckpoint returns a checkpoint set with a sum aggregation record
func getSumCheckpoint(t *testing.T, values ...int64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.CounterKind, metric.Int64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(sum.New(2))
	for _, value := range values {
		aggregatortest.CheckedUpdate(t, agg, metric.NewInt64Number(value), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
}

// getLastValueCheckpoint returns a checkpoint set with a last value aggregation record
func getLastValueCheckpoint(t *testing.T, values ...int64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.ValueObserverKind, metric.Int64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(lastvalue.New(2))
	for _, value := range values {
		aggregatortest.CheckedUpdate(t, agg, metric.NewInt64Number(value), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
}

// getMMSCCheckpoint returns a checkpoint set with a minmaxsumcount aggregation record
func getMMSCCheckpoint(t *testing.T, values ...float64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.ValueRecorderKind, metric.Float64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(minmaxsumcount.New(2, &desc))
	for _, value := range values {
		aggregatortest.CheckedUpdate(t, agg, metric.NewFloat64Number(value), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
}

// getDistributionCheckpoint returns a checkpoint set with a distribution aggregation record
func getDistributionCheckpoint(t *testing.T) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.ValueRecorderKind, metric.Float64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(array.New(2))
	for i := 0; i < 1000; i++ {
		aggregatortest.CheckedUpdate(t, agg, metric.NewFloat64Number(float64(i)+0.5), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
}

// getHistogramCheckpoint returns a checkpoint set with a histogram aggregation record
func getHistogramCheckpoint(t *testing.T) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.ValueRecorderKind, metric.Float64NumberKind)

	// Create aggregation, add value, and update checkpointset
	boundaries := []float64{100, 500, 900}
	agg, ckpt := metrictest.Unslice2(histogram.New(2, &desc, boundaries))
	for i := 0; i < 1000; i++ {
		aggregatortest.CheckedUpdate(t, agg, metric.NewFloat64Number(float64(i)+0.5), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
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

var wantDistributionCheckpointSet = []*prompb.TimeSeries{
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
				Value: "metric_name_min",
			},
		},
		Samples: []prompb.Sample{{
			Value:     0.5,
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
			Value:     999.5,
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
				Name:  "quantile",
				Value: "0.5",
			},
		},
		Samples: []prompb.Sample{{
			Value:     500.5,
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
				Name:  "quantile",
				Value: "0.9",
			},
		},
		Samples: []prompb.Sample{{
			Value:     900.5,
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
				Name:  "quantile",
				Value: "0.99",
			},
		},
		Samples: []prompb.Sample{{
			Value:     990.5,
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
