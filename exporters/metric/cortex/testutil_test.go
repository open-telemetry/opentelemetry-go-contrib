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
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
)

// AggregatorFor is copied from the SDK's processortest package, the
// only difference is it uses explicit histogram boundaries for the
// test.  TODO: If the API supported Hints, this code could be retired
// and the code below could use the hint to set the boundaries, this is
// obnoxious.
type testAggregatorSelector struct {
}

var testHistogramBoundaries = []float64{
	100, 500, 900,
}

func (testAggregatorSelector) AggregatorFor(desc *metric.Descriptor, aggPtrs ...*export.Aggregator) {
	switch {
	case strings.HasSuffix(desc.Name(), "_sum"):
		aggs := sum.New(len(aggPtrs))
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	case strings.HasSuffix(desc.Name(), "_mmsc"):
		aggs := minmaxsumcount.New(len(aggPtrs), desc)
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	case strings.HasSuffix(desc.Name(), "_lastvalue"):
		aggs := lastvalue.New(len(aggPtrs))
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	case strings.HasSuffix(desc.Name(), "_histogram"):
		aggs := histogram.New(len(aggPtrs), desc, histogram.WithExplicitBoundaries(testHistogramBoundaries))
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	default:
		panic(fmt.Sprint("Invalid instrument name for test AggregatorSelector: ", desc.Name()))
	}
}

func testMeter(t *testing.T) (context.Context, metric.Meter, *controller.Controller) {
	aggSel := testAggregatorSelector{}
	proc := processor.NewFactory(aggSel, export.CumulativeExportKindSelector())
	cont := controller.New(proc,
		controller.WithResource(testResource),
	)
	ctx := context.Background()

	return ctx, cont.Meter("test"), cont
}

// getSumReader returns a checkpoint set with a sum aggregation record
func getSumReader(t *testing.T, values ...int64) export.InstrumentationLibraryReader {
	ctx, meter, cont := testMeter(t)
	counter := metric.Must(meter).NewInt64Counter("metric_sum")

	for _, value := range values {
		counter.Add(ctx, value)
	}

	require.NoError(t, cont.Collect(ctx))

	return cont
}

// getLastValueReader returns a checkpoint set with a last value aggregation record
func getLastValueReader(t *testing.T, values ...int64) export.InstrumentationLibraryReader {
	ctx, meter, cont := testMeter(t)

	_ = metric.Must(meter).NewInt64GaugeObserver("metric_lastvalue", func(ctx context.Context, res metric.Int64ObserverResult) {
		for _, value := range values {
			res.Observe(value)
		}
	})

	require.NoError(t, cont.Collect(ctx))

	return cont
}

// getMMSCReader returns a checkpoint set with a minmaxsumcount aggregation record
func getMMSCReader(t *testing.T, values ...float64) export.InstrumentationLibraryReader {
	ctx, meter, cont := testMeter(t)

	histo := metric.Must(meter).NewFloat64Histogram("metric_mmsc")

	for _, value := range values {
		histo.Record(ctx, value)
	}

	require.NoError(t, cont.Collect(ctx))

	return cont
}

// getHistogramReader returns a checkpoint set with a histogram aggregation record
func getHistogramReader(t *testing.T) export.InstrumentationLibraryReader {
	ctx, meter, cont := testMeter(t)

	// Uses default boundaries
	histo := metric.Must(meter).NewFloat64Histogram("metric_histogram")

	for value := 0.; value < 1000; value++ {
		histo.Record(ctx, value+0.5)
	}

	require.NoError(t, cont.Collect(ctx))

	return cont
}

// The following variables hold expected TimeSeries values to be used in
// ConvertToTimeSeries tests.
var wantSumTimeSeries = []*prompb.TimeSeries{
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_sum",
			},
		},
		Samples: []prompb.Sample{{
			Value: 15,
			// Timestamp: this test verifies real timestamps
		}},
	},
}

var wantLastValueTimeSeries = []*prompb.TimeSeries{
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_lastvalue",
			},
		},
		Samples: []prompb.Sample{{
			Value: 5,
			// Timestamp: this test verifies real timestamps
		}},
	},
}

var wantMMSCTimeSeries = []*prompb.TimeSeries{
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_mmsc",
			},
		},
		Samples: []prompb.Sample{{
			Value: 999.999,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_mmsc_min",
			},
		},
		Samples: []prompb.Sample{{
			Value: 123.456,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_mmsc_max",
			},
		},
		Samples: []prompb.Sample{{
			Value: 876.543,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_mmsc_count",
			},
		},
		Samples: []prompb.Sample{{
			Value: 2,
			// Timestamp: this test verifies real timestamps
		}},
	},
}

var wantHistogramTimeSeries = []*prompb.TimeSeries{
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_histogram_sum",
			},
		},
		Samples: []prompb.Sample{{
			Value: 500000,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_histogram_count",
			},
		},
		Samples: []prompb.Sample{{
			Value: 1000,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_histogram",
			},
			{
				Name:  "le",
				Value: "100",
			},
		},
		Samples: []prompb.Sample{{
			Value: 100,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_histogram",
			},
			{
				Name:  "le",
				Value: "500",
			},
		},
		Samples: []prompb.Sample{{
			Value: 500,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_histogram",
			},
			{
				Name:  "le",
				Value: "900",
			},
		},
		Samples: []prompb.Sample{{
			Value: 900,
			// Timestamp: this test verifies real timestamps
		}},
	},
	{
		Labels: []prompb.Label{
			{
				Name:  "R",
				Value: "V",
			},
			{
				Name:  "__name__",
				Value: "metric_histogram",
			},
			{
				Name:  "le",
				Value: "+inf",
			},
		},
		Samples: []prompb.Sample{{
			Value: 1000,
			// Timestamp: this test verifies real timestamps
		}},
	},
}

func toMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
